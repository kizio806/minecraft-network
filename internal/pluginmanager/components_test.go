package pluginmanager

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSyncComponents(t *testing.T) {
	root := t.TempDir()
	mgr := &Manager{root: root}
	ctx := context.Background()
	var out bytes.Buffer

	// Missing components.lock.yml
	if err := mgr.SyncComponents(ctx, &out, false); err == nil {
		t.Fatal("expected error on missing components.lock.yml")
	}

	// Create configs/plugins
	cfgDir := filepath.Join(root, "configs/plugins")
	_ = os.MkdirAll(cfgDir, 0750)
	lockPath := filepath.Join(cfgDir, "components.lock.yml")

	// Invalid schema version
	invalidLock := ComponentLockfile{SchemaVersion: 99}
	b, _ := yaml.Marshal(invalidLock)
	_ = os.WriteFile(lockPath, b, 0600)
	if err := mgr.SyncComponents(ctx, &out, false); err == nil {
		t.Fatal("expected error on invalid schema version")
	}

	// Incomplete nanolimbo entry
	incompleteLock := ComponentLockfile{
		SchemaVersion: 1,
		Components: map[string]ComponentLockEntry{
			"nanolimbo": {Source: "invalid"},
		},
	}
	b, _ = yaml.Marshal(incompleteLock)
	_ = os.WriteFile(lockPath, b, 0600)
	if err := mgr.SyncComponents(ctx, &out, false); err == nil {
		t.Fatal("expected error on incomplete nanolimbo entry")
	}

	// Missing nanolimbo entry
	emptyLock := ComponentLockfile{
		SchemaVersion: 1,
		Components:    map[string]ComponentLockEntry{},
	}
	b, _ = yaml.Marshal(emptyLock)
	_ = os.WriteFile(lockPath, b, 0600)
	if err := mgr.SyncComponents(ctx, &out, false); err == nil {
		t.Fatal("expected error on missing nanolimbo entry")
	}

	jarBytes, sha256Hex, _ := createDummyJar(t)

	// Valid dry run (apply = false)
	validLock := ComponentLockfile{
		SchemaVersion: 1,
		Components: map[string]ComponentLockEntry{
			"nanolimbo": {
				Version:     "1.12.0",
				Source:      "github",
				Filename:    "NanoLimbo.jar",
				SHA256:      sha256Hex,
				DownloadURL: "https://github.com/NanoLimbo/NanoLimbo/releases/download/v1.12.0/NanoLimbo.jar",
			},
		},
	}
	b, _ = yaml.Marshal(validLock)
	_ = os.WriteFile(lockPath, b, 0600)

	out.Reset()
	if err := mgr.SyncComponents(ctx, &out, false); err != nil {
		t.Fatalf("unexpected dry-run error: %v", err)
	}
	if !strings.Contains(out.String(), "INSTALL\tnanolimbo\t1.12.0") {
		t.Fatalf("unexpected dry-run output: %s", out.String())
	}

	// Mock docker-compose.yml so ensureStopped passes
	_ = os.WriteFile(filepath.Join(root, "docker-compose.yml"), []byte("services: {}\n"), 0600)

	// Mock docker executable in PATH
	bin := filepath.Join(root, "bin")
	_ = os.MkdirAll(bin, 0755)
	docker := filepath.Join(bin, "docker")
	// #nosec G306 -- mock executable test script
	_ = os.WriteFile(docker, []byte("#!/bin/sh\nexit 0\n"), 0755)
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+oldPath)

	// Setup mock download server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/java-archive")
		_, _ = w.Write(jarBytes)
	}))
	defer mockServer.Close()

	// Redirect default client transport to mock server for github downloads in test
	oldTransport := DefaultHTTPClient.Transport
	DefaultHTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		targetURL, _ := url.Parse(mockServer.URL + req.URL.Path)
		clone := req.Clone(req.Context())
		clone.URL = targetURL
		return http.DefaultTransport.RoundTrip(clone)
	})
	defer func() { DefaultHTTPClient.Transport = oldTransport }()

	// Pre-create destination file to test backup logic in apply mode
	destDir := filepath.Join(root, "servers/auth")
	_ = os.MkdirAll(destDir, 0750)
	destFile := filepath.Join(destDir, "NanoLimbo.jar")
	_ = os.WriteFile(destFile, []byte("pre-existing NanoLimbo"), 0600)

	out.Reset()
	if err := mgr.SyncComponents(ctx, &out, true); err != nil {
		t.Fatalf("expected successful SyncComponents apply, got: %v", err)
	}
	if !strings.Contains(out.String(), "SYNCED\tnanolimbo\t1.12.0") {
		t.Fatalf("unexpected apply output: %s", out.String())
	}

	// Verify backup was created
	backupPath := filepath.Join(root, ".plugin-manager/rollback/nanolimbo-NanoLimbo.jar")
	backupBytes, err := os.ReadFile(backupPath)
	if err != nil || string(backupBytes) != "pre-existing NanoLimbo" {
		t.Fatalf("expected backup to exist, err=%v content=%s", err, string(backupBytes))
	}

	// Verify new installed file
	installedBytes, err := os.ReadFile(destFile)
	if err != nil || len(installedBytes) != len(jarBytes) {
		t.Fatalf("expected installed file to match dummy jar, err=%v len=%d", err, len(installedBytes))
	}
}
