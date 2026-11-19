package pluginmanager

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func createDummyJar(t *testing.T) ([]byte, string, string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("plugin.yml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("name: Test\nversion: 1.0\nmain: com.test.Main\n"))
	w2, err := zw.Create("velocity-plugin.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w2.Write([]byte(`{"id":"test","version":"1.0"}`))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	jarBytes := buf.Bytes()
	h256 := sha256.Sum256(jarBytes)
	h512 := sha512.Sum512(jarBytes)
	return jarBytes, hex.EncodeToString(h256[:]), hex.EncodeToString(h512[:])
}

func TestDownloadValidation(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "output.jar")
	ctx := context.Background()

	// Non-HTTPS
	if err := download(ctx, LockEntry{DownloadURL: "http://cdn.modrinth.com/file.jar"}, dest); err == nil {
		t.Fatal("expected error on HTTP url")
	}

	// Untrusted host
	if err := download(ctx, LockEntry{DownloadURL: "https://evil-untrusted-host.com/file.jar"}, dest); err == nil {
		t.Fatal("expected error on untrusted host")
	}

	// Invalid URL
	if err := download(ctx, LockEntry{DownloadURL: "://invalid-url"}, dest); err == nil {
		t.Fatal("expected error on malformed url")
	}
}

func TestDownloadSuccessAndIntegrity(t *testing.T) {
	jarBytes, sha256Hex, sha512Hex := createDummyJar(t)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/success.jar":
			w.Header().Set("Content-Type", "application/java-archive")
			_, _ = w.Write(jarBytes)
		case "/not-found.jar":
			http.Error(w, "not found", http.StatusNotFound)
		case "/too-large.jar":
			w.Header().Set("Content-Length", fmt.Sprintf("%d", maxArtifactSize+100))
			_, _ = w.Write([]byte("too large header"))
		}
	}))
	defer mockServer.Close()

	customClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			targetURL, _ := url.Parse(mockServer.URL + req.URL.Path)
			clone := req.Clone(req.Context())
			clone.URL = targetURL
			return http.DefaultTransport.RoundTrip(clone)
		}),
	}

	dir := t.TempDir()
	dest := filepath.Join(dir, "downloaded.jar")

	// Successful download
	entry := LockEntry{
		DownloadURL: "https://cdn.modrinth.com/success.jar",
		Filename:    "downloaded.jar",
		SHA256:      sha256Hex,
		SHA512:      sha512Hex,
		Size:        int64(len(jarBytes)),
	}
	if err := download(context.Background(), entry, dest, customClient); err != nil {
		t.Fatalf("expected successful download, got: %v", err)
	}

	gotBytes, err := os.ReadFile(dest)
	if err != nil || len(gotBytes) != len(jarBytes) {
		t.Fatalf("downloaded file mismatch: len=%d err=%v", len(gotBytes), err)
	}

	// HTTP 404 test
	entry404 := LockEntry{
		DownloadURL: "https://cdn.modrinth.com/not-found.jar",
		Filename:    "downloaded.jar",
	}
	if err := download(context.Background(), entry404, filepath.Join(dir, "404.jar"), customClient); err == nil {
		t.Fatal("expected error on 404 status code")
	}

	// Content-Length too large
	entryTooLarge := LockEntry{
		DownloadURL: "https://cdn.modrinth.com/too-large.jar",
		Filename:    "downloaded.jar",
	}
	if err := download(context.Background(), entryTooLarge, filepath.Join(dir, "large.jar"), customClient); err == nil {
		t.Fatal("expected error on too large artifact")
	}

	// Hash mismatch
	entryBadHash := LockEntry{
		DownloadURL: "https://cdn.modrinth.com/success.jar",
		Filename:    "downloaded.jar",
		SHA256:      "0000000000000000000000000000000000000000000000000000000000000000",
	}
	if err := download(context.Background(), entryBadHash, filepath.Join(dir, "badhash.jar"), customClient); err == nil {
		t.Fatal("expected error on sha256 mismatch")
	}
}

func TestDownloadRedirectChecks(t *testing.T) {
	// Test redirect policy logic directly
	checkRedirect := DefaultHTTPClient.CheckRedirect

	// Under limit
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://cdn.modrinth.com/sub", nil)
	via := []*http.Request{req}
	if err := checkRedirect(req, via); err != nil {
		t.Fatalf("unexpected redirect error: %v", err)
	}

	// Too many redirects
	via3 := []*http.Request{req, req, req}
	if err := checkRedirect(req, via3); err == nil {
		t.Fatal("expected error on >= 3 redirects")
	}

	// Redirect to non-HTTPS
	insecureReq, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://cdn.modrinth.com/sub", nil)
	if err := checkRedirect(insecureReq, via); err == nil {
		t.Fatal("expected error on redirect to HTTP")
	}

	// Redirect to untrusted host
	untrustedReq, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://evil.com/sub", nil)
	if err := checkRedirect(untrustedReq, via); err == nil {
		t.Fatal("expected error on redirect to untrusted host")
	}
}
