package pluginmanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownload_ExtendedErrors(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "target.jar")

	t.Run("http 404", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "missing", http.StatusNotFound)
		}))
		defer ts.Close()

		u, _ := url.Parse(ts.URL)
		entry := LockEntry{
			DownloadURL: "https://cdn.modrinth.com/test.jar",
			Filename:    "test.jar",
		}
		client := ts.Client()
		client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = u.Scheme
			req.URL.Host = u.Host
			return http.DefaultTransport.RoundTrip(req)
		})

		err := download(context.Background(), entry, dest, client)
		if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
			t.Fatalf("expected HTTP 404 error, got: %v", err)
		}
	})

	t.Run("exceeds max size", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "105000000")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("too large header"))
		}))
		defer ts.Close()

		u, _ := url.Parse(ts.URL)
		entry := LockEntry{
			DownloadURL: "https://cdn.modrinth.com/large.jar",
			Filename:    "large.jar",
		}
		client := ts.Client()
		client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = u.Scheme
			req.URL.Host = u.Host
			return http.DefaultTransport.RoundTrip(req)
		})

		err := download(context.Background(), entry, dest, client)
		if err == nil || !strings.Contains(err.Error(), "exceeds maximum size") {
			t.Fatalf("expected exceeds maximum size error, got: %v", err)
		}
	})

	t.Run("checksum failure", func(t *testing.T) {
		jarBytes, _, _ := createDummyJar(t)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(jarBytes)
		}))
		defer ts.Close()

		u, _ := url.Parse(ts.URL)
		entry := LockEntry{
			DownloadURL: "https://cdn.modrinth.com/badhash.jar",
			Filename:    "badhash.jar",
			SHA256:      "0000000000000000000000000000000000000000000000000000000000000000",
		}
		client := ts.Client()
		client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = u.Scheme
			req.URL.Host = u.Host
			return http.DefaultTransport.RoundTrip(req)
		})

		err := download(context.Background(), entry, dest, client)
		if err == nil || !strings.Contains(err.Error(), "download integrity") {
			t.Fatalf("expected download integrity error, got: %v", err)
		}
	})
}

func TestSyncComponents_Extended(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "configs", "plugins", "manifest.yml")
	lockPath := filepath.Join(dir, "configs", "plugins", "lock.yml")
	_ = os.MkdirAll(filepath.Dir(manifestPath), 0755)

	mgr := New(manifestPath, lockPath)

	// Missing component lockfile
	var buf strings.Builder
	if err := mgr.SyncComponents(context.Background(), &buf, false); err == nil {
		t.Fatal("expected error on missing components lockfile, got nil")
	}

	// Invalid schema version
	compLockPath := filepath.Join(dir, "configs", "plugins", "components.lock.yml")
	_ = os.WriteFile(compLockPath, []byte("schema-version: 99\ncomponents: {}\n"), 0600)
	if err := mgr.SyncComponents(context.Background(), &buf, false); err == nil || !strings.Contains(err.Error(), "schema-version must be 1") {
		t.Fatalf("expected schema-version must be 1 error, got: %v", err)
	}

	// Empty components
	_ = os.WriteFile(compLockPath, []byte("schema-version: 1\ncomponents: {}\n"), 0600)
	if err := mgr.SyncComponents(context.Background(), &buf, false); err == nil || !strings.Contains(err.Error(), "has no components defined") {
		t.Fatalf("expected has no components defined error, got: %v", err)
	}

	// Incomplete component entry
	incompleteYAML := `schema-version: 1
components:
  datapack1:
    version: "1.0.0"
    filename: ""
`
	_ = os.WriteFile(compLockPath, []byte(incompleteYAML), 0600)
	if err := mgr.SyncComponents(context.Background(), &buf, false); err == nil || !strings.Contains(err.Error(), "lock entry is incomplete") {
		t.Fatalf("expected lock entry is incomplete error, got: %v", err)
	}

	// Dry run (apply=false) with both nanolimbo and a survival datapack
	validComponentsYAML := `schema-version: 1
components:
  nanolimbo:
    version: "1.12.0"
    filename: "NanoLimbo.jar"
    sha256: "0000000000000000000000000000000000000000000000000000000000000000"
    download-url: "https://example.com/NanoLimbo.jar"
  world-pack:
    version: "2.0.0"
    filename: "world-pack.zip"
    sha256: "1111111111111111111111111111111111111111111111111111111111111111"
    download-url: "https://example.com/world-pack.zip"
`
	_ = os.WriteFile(compLockPath, []byte(validComponentsYAML), 0600)
	buf.Reset()
	if err := mgr.SyncComponents(context.Background(), &buf, false); err != nil {
		t.Fatalf("unexpected dry-run SyncComponents error: %v", err)
	}
	outStr := buf.String()
	if !strings.Contains(outStr, "INSTALL\tnanolimbo\t1.12.0") || !strings.Contains(outStr, "INSTALL\tworld-pack\t2.0.0") {
		t.Fatalf("unexpected dry run output: %s", outStr)
	}

	// Also verify componentDestination for survival datapack
	dest, svc := mgr.componentDestination("world-pack", ComponentLockEntry{Filename: "world-pack.zip"})
	if svc != "survival" || !strings.HasSuffix(dest, "servers/survival/world/datapacks/world-pack.zip") {
		t.Fatalf("unexpected componentDestination for datapack: dest=%s, svc=%s", dest, svc)
	}
}

func TestManager_Extended(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yml")
	lockPath := filepath.Join(dir, "lock.yml")

	// Missing manifest or lockfile during load
	mgr := New(manifestPath, lockPath)
	if err := mgr.Validate(); err == nil {
		t.Fatal("expected error validating non-existent files, got nil")
	}

	// UpdateCheck
	_ = os.WriteFile(manifestPath, []byte(`schema-version: 1
environment:
  minecraft-version: "1.21.11"
  backend: purpur
  proxy: velocity
  java: "25"
plugins:
  custom:
    display-name: Custom
    enabled: true
    source: modrinth
    project: custom-proj
    version: "1.0.0"
    version-id: "v1.0.0"
    platform: velocity
    servers: [proxy]
    update-policy: manual
    required-java: "25"
    release-channel: release
`), 0600)
	_ = os.WriteFile(lockPath, []byte(`schema-version: 1
generated-by: test
plugins:
  custom:
    project: custom-proj
    version: "1.0.0"
    version-id: "v1.0.0"
    source: modrinth
    download-url: https://cdn.modrinth.com/custom.jar
    filename: custom.jar
    sha256: 0000000000000000000000000000000000000000000000000000000000000000
    game-versions: ["1.21.11"]
    loaders: ["velocity"]
`), 0600)

	mgr2 := New(manifestPath, lockPath)
	var out strings.Builder
	if err := mgr2.UpdateCheck(context.Background(), &out); err != nil {
		t.Fatalf("unexpected error in UpdateCheck: %v", err)
	}
	if !strings.Contains(out.String(), "custom\tlocked=1.0.0") {
		t.Errorf("unexpected output from UpdateCheck: %s", out.String())
	}
}
