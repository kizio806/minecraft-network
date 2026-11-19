package pluginmanager

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestManagerMethods(t *testing.T) {
	root := t.TempDir()
	manifestDir := filepath.Join(root, "configs/plugins")
	if err := os.MkdirAll(manifestDir, 0750); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(manifestDir, "manifest.yml")
	lockPath := filepath.Join(manifestDir, "lock.yml")

	m, l := validFixture()
	mb, _ := yaml.Marshal(m)
	lb, _ := yaml.Marshal(l)
	_ = os.WriteFile(manifestPath, mb, 0600)
	_ = os.WriteFile(lockPath, lb, 0600)

	mgr := New(manifestPath, lockPath)
	if mgr.root != root {
		t.Fatalf("unexpected root: %s, expected: %s", mgr.root, root)
	}

	// Validate
	if err := mgr.Validate(); err != nil {
		t.Fatalf("unexpected Validate error: %v", err)
	}

	// List
	var listBuf bytes.Buffer
	if err := mgr.List(&listBuf); err != nil {
		t.Fatalf("unexpected List error: %v", err)
	}
	if !strings.Contains(listBuf.String(), "example") || !strings.Contains(listBuf.String(), "enabled") {
		t.Fatalf("unexpected List output: %s", listBuf.String())
	}

	// Plan (no installed.yml yet -> INSTALL)
	var planBuf bytes.Buffer
	if err := mgr.Plan(&planBuf); err != nil {
		t.Fatalf("unexpected Plan error: %v", err)
	}
	if !strings.Contains(planBuf.String(), "INSTALL\texample") {
		t.Fatalf("unexpected Plan output: %s", planBuf.String())
	}

	// Plan with installed.yml matching -> OK
	pmDir := filepath.Join(root, ".plugin-manager")
	_ = os.MkdirAll(pmDir, 0700)
	installedYAML, _ := yaml.Marshal(InstalledState{Plugins: l.Plugins})
	_ = os.WriteFile(filepath.Join(pmDir, "installed.yml"), installedYAML, 0600)

	planBuf.Reset()
	if err := mgr.Plan(&planBuf); err != nil {
		t.Fatalf("unexpected Plan error: %v", err)
	}
	if !strings.Contains(planBuf.String(), "OK\texample") {
		t.Fatalf("unexpected Plan output: %s", planBuf.String())
	}

	// UpdateCheck
	var updateBuf bytes.Buffer
	if err := mgr.UpdateCheck(context.Background(), &updateBuf); err != nil {
		t.Fatalf("unexpected UpdateCheck error: %v", err)
	}
	if !strings.Contains(updateBuf.String(), "example") {
		t.Fatalf("unexpected UpdateCheck output: %s", updateBuf.String())
	}

	// VerifyInstalled (missing artifact file -> error)
	if err := mgr.VerifyInstalled(); err == nil {
		t.Fatal("expected VerifyInstalled to fail when artifact file is missing")
	}

	// VerifyInstalled with valid file
	proxyPluginsDir := filepath.Join(root, "servers/proxy/plugins")
	_ = os.MkdirAll(proxyPluginsDir, 0750)
	targetJar := filepath.Join(proxyPluginsDir, l.Plugins["example"].Filename)
	_ = os.WriteFile(targetJar, []byte("valid-jar-data"), 0600)
	// Update lockfile entry to match hash of "valid-jar-data"
	l.Plugins["example"] = LockEntry{
		Project:      "project",
		Version:      "1.0.0",
		VersionID:    "version-id",
		Source:       "modrinth",
		DownloadURL:  "https://cdn.modrinth.com/project/version-id/example.jar",
		Filename:     "example.jar",
		SHA256:       "4bb91820ee5ce12ebcb4528b73a5913b0fbb16204846c57745c20043dbc61071",
		GameVersions: []string{"1.21.11"},
		Loaders:      []string{"velocity"},
		Dependencies: []string{},
	}
	lb, _ = yaml.Marshal(l)
	_ = os.WriteFile(lockPath, lb, 0600)

	if err := mgr.VerifyInstalled(); err != nil {
		t.Fatalf("unexpected VerifyInstalled error: %v", err)
	}
}
