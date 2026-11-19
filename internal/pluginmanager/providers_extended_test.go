package pluginmanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModrinthProvider_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	// 1. Identity mismatch (project_id mismatch)
	serverMismatch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id":"v1","project_id":"wrong_project","version_number":"1.0.0","files":[{"url":"https://cdn.modrinth.com/p/v.jar","filename":"v.jar","size":3,"hashes":{"sha512":"abc"}}]}`))
	}))
	defer serverMismatch.Close()

	pMismatch := ModrinthProvider{BaseURL: serverMismatch.URL, Client: serverMismatch.Client()}
	if _, err := pMismatch.Resolve(ctx, "correct_project", "v1"); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatalf("expected identity mismatch error, got: %v", err)
	}

	// 2. Empty files list
	serverNoFiles := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id":"v1","project_id":"p1","version_number":"1.0.0","files":[]}`))
	}))
	defer serverNoFiles.Close()

	pNoFiles := ModrinthProvider{BaseURL: serverNoFiles.URL, Client: serverNoFiles.Client()}
	if _, err := pNoFiles.Resolve(ctx, "p1", "v1"); err == nil || !strings.Contains(err.Error(), "no files") {
		t.Fatalf("expected no files error, got: %v", err)
	}

	// 3. Malformed JSON
	serverBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`not a json object`))
	}))
	defer serverBadJSON.Close()

	pBadJSON := ModrinthProvider{BaseURL: serverBadJSON.URL, Client: serverBadJSON.Client()}
	if _, err := pBadJSON.Resolve(ctx, "p1", "v1"); err == nil {
		t.Fatal("expected error on malformed JSON, got nil")
	}
}

func TestGitHubProvider_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	// 1. Tag name mismatch
	serverTagMismatch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v2.0.0","assets":[{"name":"plugin.jar","browser_download_url":"https://github.com/o/r/releases/download/v1.0.0/plugin.jar","size":3,"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`))
	}))
	defer serverTagMismatch.Close()

	pTag := GitHubProvider{BaseURL: serverTagMismatch.URL, Client: serverTagMismatch.Client()}
	if _, err := pTag.Resolve(ctx, "o/r#plugin.jar", "v1.0.0"); err == nil || !strings.Contains(err.Error(), "tag identity mismatch") {
		t.Fatalf("expected tag identity mismatch error, got: %v", err)
	}

	// 2. Asset missing SHA-256 digest
	serverNoDigest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.0.0","assets":[{"name":"plugin.jar","browser_download_url":"https://github.com/o/r/releases/download/v1.0.0/plugin.jar","size":3,"digest":""}]}`))
	}))
	defer serverNoDigest.Close()

	pNoDigest := GitHubProvider{BaseURL: serverNoDigest.URL, Client: serverNoDigest.Client()}
	if _, err := pNoDigest.Resolve(ctx, "o/r#plugin.jar", "v1.0.0"); err == nil || !strings.Contains(err.Error(), "no SHA-256 digest") {
		t.Fatalf("expected no SHA-256 digest error, got: %v", err)
	}

	// 3. Asset not found in release
	serverAssetNotFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.0.0","assets":[{"name":"other.jar","browser_download_url":"https://github.com/o/r/releases/download/v1.0.0/other.jar","size":3,"digest":"sha256:aaa"}]}`))
	}))
	defer serverAssetNotFound.Close()

	pNotFound := GitHubProvider{BaseURL: serverAssetNotFound.URL, Client: serverAssetNotFound.Client()}
	if _, err := pNotFound.Resolve(ctx, "o/r#plugin.jar", "v1.0.0"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected asset not found error, got: %v", err)
	}

	// 4. HTTP error status
	serverHTTPError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer serverHTTPError.Close()

	pHTTPError := GitHubProvider{BaseURL: serverHTTPError.URL, Client: serverHTTPError.Client()}
	if _, err := pHTTPError.Resolve(ctx, "o/r#plugin.jar", "v1.0.0"); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("expected HTTP 404 error, got: %v", err)
	}

	// 5. Malformed JSON
	serverBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer serverBadJSON.Close()

	pBadJSON := GitHubProvider{BaseURL: serverBadJSON.URL, Client: serverBadJSON.Client()}
	if _, err := pBadJSON.Resolve(ctx, "o/r#plugin.jar", "v1.0.0"); err == nil {
		t.Fatal("expected error on malformed JSON, got nil")
	}
}
