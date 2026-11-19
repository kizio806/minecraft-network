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

func TestSyncHelpers(t *testing.T) {
	root := t.TempDir()
	mgr := &Manager{root: root}
	ctx := context.Background()

	// ensureStopped: missing docker-compose.yml
	if err := mgr.ensureStopped(ctx, []string{"proxy"}); err == nil {
		t.Fatal("expected error when docker-compose.yml is missing")
	}

	// Create docker-compose.yml
	_ = os.WriteFile(filepath.Join(root, "docker-compose.yml"), []byte("services: {}\n"), 0600)

	// Mock docker script that reports running services
	bin := t.TempDir()
	docker := filepath.Join(bin, "docker")
	script := "#!/bin/sh\ncase \"$*\" in *running*) echo proxy; exit 0 ;; esac\nexit 0\n"
	_ = os.WriteFile(docker, []byte(script), 0755)
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+oldPath)

	// ensureStopped: service is running
	if err := mgr.ensureStopped(ctx, []string{"proxy"}); err == nil {
		t.Fatal("expected error when target service is running")
	}

	// ensureStopped: other service running, target is stopped
	if err := mgr.ensureStopped(ctx, []string{"lobby"}); err != nil {
		t.Fatalf("unexpected error when target service is stopped: %v", err)
	}

	// validateConfigDir
	validConfigDir := filepath.Join(root, "configs/proxy/plugins/testplugin")
	_ = os.MkdirAll(validConfigDir, 0750)
	_ = os.WriteFile(filepath.Join(validConfigDir, "config.yml"), []byte("setting: 1\n"), 0600)

	if err := mgr.validateConfigDir("configs/proxy/plugins/testplugin"); err != nil {
		t.Fatalf("unexpected validateConfigDir error: %v", err)
	}

	// validateConfigDir: escaping root
	if err := mgr.validateConfigDir("../../outside"); err == nil {
		t.Fatal("expected error on escaping root")
	}

	// validateConfigDir: invalid YAML
	invalidYmlDir := filepath.Join(root, "configs/proxy/plugins/bad")
	_ = os.MkdirAll(invalidYmlDir, 0750)
	_ = os.WriteFile(filepath.Join(invalidYmlDir, "config.yml"), []byte("key: [unclosed list"), 0600)
	if err := mgr.validateConfigDir("configs/proxy/plugins/bad"); err == nil {
		t.Fatal("expected error on invalid YAML in config dir")
	}

	// installOne
	stagedFile := filepath.Join(root, "staged.jar")
	_ = os.WriteFile(stagedFile, []byte("staged-jar-content"), 0600)
	rollbackRoot := filepath.Join(root, ".rollback")
	entry := LockEntry{Filename: "installed.jar"}

	c, err := mgr.installOne(stagedFile, entry, "proxy", rollbackRoot)
	if err != nil {
		t.Fatalf("unexpected installOne error: %v", err)
	}
	if c.existed {
		t.Fatal("expected c.existed to be false on fresh install")
	}
	content, _ := os.ReadFile(c.destination)
	if string(content) != "staged-jar-content" {
		t.Fatalf("unexpected installed content: %s", string(content))
	}

	// installOne when destination exists (backup created)
	_ = os.WriteFile(stagedFile, []byte("updated-jar-content"), 0600)
	c2, err := mgr.installOne(stagedFile, entry, "proxy", rollbackRoot)
	if err != nil {
		t.Fatalf("unexpected installOne overwrite error: %v", err)
	}
	if !c2.existed {
		t.Fatal("expected c2.existed to be true on overwrite")
	}
	backupData, _ := os.ReadFile(c2.backup)
	if string(backupData) != "staged-jar-content" {
		t.Fatalf("expected backup to contain staged-jar-content, got: %s", string(backupData))
	}

	// installOne: unsupported target
	if _, err := mgr.installOne(stagedFile, entry, "unsupported_tgt", rollbackRoot); err == nil {
		t.Fatal("expected error on unsupported target")
	}

	// writeInstalled
	l := Lockfile{SchemaVersion: 1, Plugins: map[string]LockEntry{"p1": entry}}
	if err := mgr.writeInstalled(l); err != nil {
		t.Fatalf("writeInstalled failed: %v", err)
	}
	installed, err := mgr.readInstalled()
	if err != nil || len(installed.Plugins) != 1 {
		t.Fatalf("readInstalled failed: %v", err)
	}

	// installConfigTree tests
	cfgTreeDir := filepath.Join(root, "configs/proxy/plugins/mytree")
	_ = os.MkdirAll(cfgTreeDir, 0750)
	_ = os.WriteFile(filepath.Join(cfgTreeDir, "a.yml"), []byte("k: v\n"), 0600)

	// Unsafe targetDir
	if _, err := mgr.installConfigTree("configs/proxy/plugins/mytree", "../escape", "proxy", rollbackRoot); err == nil {
		t.Fatal("expected error on escaping targetDir")
	}
	if _, err := mgr.installConfigTree("configs/proxy/plugins/mytree", ".", "proxy", rollbackRoot); err == nil {
		t.Fatal("expected error on dot targetDir")
	}

	// Escaping configDir
	if _, err := mgr.installConfigTree("../../outside", "mytree", "proxy", rollbackRoot); err == nil {
		t.Fatal("expected error on escaping configDir")
	}

	// configDir is a regular file
	regularFile := filepath.Join(root, "not_a_dir.txt")
	_ = os.WriteFile(regularFile, []byte("data"), 0600)
	if _, err := mgr.installConfigTree("not_a_dir.txt", "mytree", "proxy", rollbackRoot); err == nil {
		t.Fatal("expected error when configDir is a file")
	}

	// Valid installConfigTree run
	changes, err := mgr.installConfigTree("configs/proxy/plugins/mytree", "mytree", "proxy", rollbackRoot)
	if err != nil {
		t.Fatalf("unexpected installConfigTree error: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	installedContent, _ := os.ReadFile(changes[0].destination)
	if string(installedContent) != "k: v\n" {
		t.Fatalf("unexpected tree installed content: %s", string(installedContent))
	}
}

func TestSyncExecution(t *testing.T) {
	root := t.TempDir()
	manifestDir := filepath.Join(root, "configs/plugins")
	_ = os.MkdirAll(manifestDir, 0750)
	manifestPath := filepath.Join(manifestDir, "manifest.yml")
	lockPath := filepath.Join(manifestDir, "lock.yml")

	m, l := validFixture()
	// Disable plugin so it doesn't attempt download over network
	p := m.Plugins["example"]
	p.Enabled = false
	m.Plugins["example"] = p

	mb, _ := yaml.Marshal(m)
	lb, _ := yaml.Marshal(l)
	_ = os.WriteFile(manifestPath, mb, 0600)
	_ = os.WriteFile(lockPath, lb, 0600)
	_ = os.WriteFile(filepath.Join(root, "docker-compose.yml"), []byte("services: {}\n"), 0600)

	bin := t.TempDir()
	docker := filepath.Join(bin, "docker")
	_ = os.WriteFile(docker, []byte("#!/bin/sh\nexit 0\n"), 0755)
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+oldPath)

	mgr := New(manifestPath, lockPath)
	ctx := context.Background()
	var out bytes.Buffer

	// Sync dry-run (calls Plan)
	if err := mgr.Sync(ctx, &out, false); err != nil {
		t.Fatalf("unexpected Sync dry-run error: %v", err)
	}

	// Sync apply with disabled plugins (no network needed, writes installed state)
	out.Reset()
	if err := mgr.Sync(ctx, &out, true); err != nil {
		t.Fatalf("unexpected Sync apply error: %v", err)
	}

	// Sync apply with invalid config dir
	p.Enabled = true
	p.ConfigDir = "../../invalid_escaped_dir"
	m.Plugins["example"] = p
	mb, _ = yaml.Marshal(m)
	_ = os.WriteFile(manifestPath, mb, 0600)

	if err := mgr.Sync(ctx, &out, true); err == nil {
		t.Fatal("expected error on invalid ConfigDir in Sync")
	}

	// Full Sync apply with an enabled plugin, valid config dir, and mock download
	jarBytes, sha256Hex, sha512Hex := createDummyJar(t)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/java-archive")
		_, _ = w.Write(jarBytes)
	}))
	defer mockServer.Close()

	oldTransport := DefaultHTTPClient.Transport
	DefaultHTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		targetURL, _ := url.Parse(mockServer.URL + req.URL.Path)
		clone := req.Clone(req.Context())
		clone.URL = targetURL
		return http.DefaultTransport.RoundTrip(clone)
	})
	defer func() { DefaultHTTPClient.Transport = oldTransport }()

	cfgSourceDir := filepath.Join(root, "configs/proxy/plugins/example")
	_ = os.MkdirAll(cfgSourceDir, 0750)
	_ = os.WriteFile(filepath.Join(cfgSourceDir, "config.yml"), []byte("enabled: true\n"), 0600)

	p.Enabled = true
	p.ConfigDir = "configs/proxy/plugins/example"
	p.ConfigTargetDir = "example"
	m.Plugins["example"] = p

	lEntry := l.Plugins["example"]
	lEntry.SHA256 = sha256Hex
	lEntry.SHA512 = sha512Hex
	lEntry.Size = int64(len(jarBytes))
	lEntry.DownloadURL = "https://cdn.modrinth.com/p1/v1/example.jar"
	l.Plugins["example"] = lEntry

	mb, _ = yaml.Marshal(m)
	lb, _ = yaml.Marshal(l)
	_ = os.WriteFile(manifestPath, mb, 0600)
	_ = os.WriteFile(lockPath, lb, 0600)

	out.Reset()
	if err := mgr.Sync(ctx, &out, true); err != nil {
		t.Fatalf("unexpected full Sync apply error: %v", err)
	}
	if !strings.Contains(out.String(), "SYNCED\texample\t1.0.0") {
		t.Fatalf("expected SYNCED output, got: %s", out.String())
	}
}
