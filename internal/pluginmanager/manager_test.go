package pluginmanager

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validFixture() (Manifest, Lockfile) {
	return Manifest{SchemaVersion: 1, Environment: Environment{MinecraftVersion: "1.21.11", Backend: "purpur", Proxy: "velocity", Java: "25"}, Plugins: map[string]PluginSpec{
			"example": {DisplayName: "Example", Enabled: true, Source: "modrinth", Project: "project", Version: "1.0.0", VersionID: "version-id", Platform: "velocity", Servers: []string{"proxy"}, ReleaseChannel: "release", UpdatePolicy: "manual", RequiredJava: "25"},
		}}, Lockfile{SchemaVersion: 1, GeneratedBy: "test", Plugins: map[string]LockEntry{
			"example": {Project: "project", Version: "1.0.0", VersionID: "version-id", Source: "modrinth", DownloadURL: "https://cdn.modrinth.com/data/project/version-id/example.jar", Filename: "example.jar", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", GameVersions: []string{"1.21.11"}, Loaders: []string{"velocity"}, Dependencies: []string{}},
		}}
}

func TestValidateAcceptsPinnedManifest(t *testing.T) {
	m, l := validFixture()
	if err := Validate(m, l); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsFloatingVersion(t *testing.T) {
	m, l := validFixture()
	p := m.Plugins["example"]
	p.Version = "latest"
	m.Plugins["example"] = p
	if err := Validate(m, l); err == nil {
		t.Fatal("expected latest to be rejected")
	}
}

func TestDependencyCycle(t *testing.T) {
	m, l := validFixture()
	a := m.Plugins["example"]
	a.Dependencies = []string{"other"}
	m.Plugins["example"] = a
	m.Plugins["other"] = PluginSpec{Enabled: true, Source: "modrinth", Project: "other", Version: "1", VersionID: "v", Platform: "velocity", Servers: []string{"proxy"}, UpdatePolicy: "manual", Dependencies: []string{"example"}}
	l.Plugins["other"] = LockEntry{Project: "other", Version: "1", VersionID: "v", Source: "modrinth", DownloadURL: "https://cdn.modrinth.com/other/v/other.jar", Filename: "other.jar", SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	if err := Validate(m, l); err == nil {
		t.Fatal("expected cycle to be rejected")
	}
}

func TestModrinthProviderUsesExactVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id":"v1","project_id":"p1","version_number":"1.0.0","version_type":"release","date_published":"2026-01-01T00:00:00Z","game_versions":["1.21.11"],"loaders":["velocity"],"files":[{"url":"https://cdn.modrinth.com/p/v.jar","filename":"v.jar","size":3,"hashes":{"sha512":"abc"}}]}`))
	}))
	defer server.Close()
	e, err := (ModrinthProvider{BaseURL: server.URL, Client: server.Client()}).Resolve(context.Background(), "p1", "v1")
	if err != nil || e.VersionID != "v1" || e.SHA512 != "abc" {
		t.Fatalf("unexpected metadata: %+v %v", e, err)
	}

	// Floating latest check
	if _, err := (ModrinthProvider{}).Resolve(context.Background(), "p1", "latest"); err == nil {
		t.Fatal("expected error on latest version")
	}
	if _, err := (ModrinthProvider{}).Resolve(context.Background(), "latest", "v1"); err == nil {
		t.Fatal("expected error on latest project")
	}

	// HTTP error
	serverErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer serverErr.Close()
	if _, err := (ModrinthProvider{BaseURL: serverErr.URL, Client: serverErr.Client()}).Resolve(context.Background(), "p1", "v1"); err == nil {
		t.Fatal("expected error on HTTP 500")
	}
}

func TestGitHubProviderUsesExactTagAndDigest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.0.0","assets":[{"name":"plugin.jar","browser_download_url":"https://github.com/o/r/releases/download/v1.0.0/plugin.jar","size":3,"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`))
	}))
	defer server.Close()
	e, err := (GitHubProvider{BaseURL: server.URL, Client: server.Client()}).Resolve(context.Background(), "o/r#plugin.jar", "v1.0.0")
	if err != nil || e.SHA256 == "" || e.VersionID != "v1.0.0" {
		t.Fatalf("unexpected metadata: %+v %v", e, err)
	}

	// Invalid project syntax
	if _, err := (GitHubProvider{}).Resolve(context.Background(), "owner/repo-without-hash", "v1.0.0"); err == nil {
		t.Fatal("expected error on invalid github project syntax")
	}
	if _, err := (GitHubProvider{}).Resolve(context.Background(), "o/r#plugin.jar", "latest"); err == nil {
		t.Fatal("expected error on latest version in github provider")
	}

	// Asset not found
	if _, err := (GitHubProvider{BaseURL: server.URL, Client: server.Client()}).Resolve(context.Background(), "o/r#missing.jar", "v1.0.0"); err == nil {
		t.Fatal("expected error on missing release asset")
	}
}

func TestVerifyFileRejectsWrongHash(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.jar")
	if err := os.WriteFile(p, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	err := verifyFile(p, LockEntry{SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	if err == nil {
		t.Fatal("expected hash mismatch")
	}
}

func TestInstallConfigTreeCopiesAndRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	m := &Manager{root: root}
	source := filepath.Join(root, "configs", "proxy", "plugins", "example")
	if err := os.MkdirAll(source, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "config.yml"), []byte("enabled: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	rollbackRoot := filepath.Join(root, ".rollback")
	changes, err := m.installConfigTree("configs/proxy/plugins/example", "example", "proxy", rollbackRoot)
	if err != nil || len(changes) != 1 {
		t.Fatalf("copy failed: %v, changes=%d", err, len(changes))
	}
	destination := filepath.Join(root, "servers/proxy/plugins/example/config.yml")
	if b, err := os.ReadFile(destination); err != nil || string(b) != "enabled: true\n" {
		t.Fatalf("unexpected copied config: %q, %v", b, err)
	}
	if err := os.Symlink(filepath.Join(root, "outside"), filepath.Join(source, "unsafe.yml")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.installConfigTree("configs/proxy/plugins/example", "example", "proxy", rollbackRoot); err == nil {
		t.Fatal("expected symlink rejection")
	}
}

func TestTargetRootsIncludesLobbyAndSurvival(t *testing.T) {
	if targetRoots["lobby"] != "servers/lobby/plugins" {
		t.Errorf("expected servers/lobby/plugins, got %s", targetRoots["lobby"])
	}
	if targetRoots["survival"] != "servers/survival/plugins" {
		t.Errorf("expected servers/survival/plugins, got %s", targetRoots["survival"])
	}
}

func TestInspectJarDistinguishesPlatforms(t *testing.T) {
	dir := t.TempDir()

	// Create mock paper plugin jar
	paperJar := filepath.Join(dir, "paper.jar")
	f, err := os.Create(paperJar)
	if err != nil {
		t.Fatal(err)
	}
	zw := zipWriter(f, "paper-plugin.yml")
	zw.Close()
	f.Close()

	if err := inspectJar(paperJar, "paper"); err != nil {
		t.Errorf("expected paper descriptor accepted, got %v", err)
	}
	if err := inspectJar(paperJar, "velocity"); err == nil {
		t.Error("expected paper descriptor rejected for velocity platform")
	}

	// Create mock velocity plugin jar
	veloJar := filepath.Join(dir, "velocity.jar")
	f2, err := os.Create(veloJar)
	if err != nil {
		t.Fatal(err)
	}
	zw2 := zipWriter(f2, "velocity-plugin.json")
	zw2.Close()
	f2.Close()

	if err := inspectJar(veloJar, "velocity"); err != nil {
		t.Errorf("expected velocity descriptor accepted, got %v", err)
	}
	if err := inspectJar(veloJar, "paper"); err == nil {
		t.Error("expected velocity descriptor rejected for paper platform")
	}
}

func zipWriter(f *os.File, entryName string) io.Closer {
	zw := zip.NewWriter(f)
	w, _ := zw.Create(entryName)
	_, _ = w.Write([]byte("name: test\nversion: 1.0\n"))
	return zw
}

func TestConfigTargetDirPreservesExactCaseAndNestedStructure(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "configs/lobby/plugins/deluxemenus/gui_menus")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	navPath := filepath.Join(sourceDir, "navigation.yml")
	if err := os.WriteFile(navPath, []byte("menu_title: 'Nav'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	mainCfg := filepath.Join(root, "configs/lobby/plugins/deluxemenus/config.yml")
	if err := os.WriteFile(mainCfg, []byte("debug: false\n"), 0600); err != nil {
		t.Fatal(err)
	}

	m := &Manager{root: root}
	rollbackRoot := filepath.Join(root, "rollback")
	_, err := m.installConfigTree("configs/lobby/plugins/deluxemenus", "DeluxeMenus", "lobby", rollbackRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify exact case preservation and nested structure
	expectedNav := filepath.Join(root, "servers/lobby/plugins/DeluxeMenus/gui_menus/navigation.yml")
	if _, err := os.Stat(expectedNav); err != nil {
		t.Fatalf("expected nested file at %s, got: %v", expectedNav, err)
	}

	expectedMain := filepath.Join(root, "servers/lobby/plugins/DeluxeMenus/config.yml")
	if _, err := os.Stat(expectedMain); err != nil {
		t.Fatalf("expected main config at %s, got: %v", expectedMain, err)
	}

	// Verify it was NOT installed into survival
	survivalPath := filepath.Join(root, "servers/survival/plugins/DeluxeMenus")
	if _, err := os.Stat(survivalPath); !os.IsNotExist(err) {
		t.Errorf("config should not exist in survival, but stat returned: %v", err)
	}
}

func TestRuntimePluginDirectoryCaseSensitivityMappings(t *testing.T) {
	cases := []struct {
		sourceDir string
		targetDir string
	}{
		{"configs/lobby/plugins/essentialsx", "Essentials"},
		{"configs/lobby/plugins/essentialsxspawn", "EssentialsSpawn"},
		{"configs/lobby/plugins/luckperms", "LuckPerms"},
		{"configs/lobby/plugins/placeholderapi", "PlaceholderAPI"},
		{"configs/lobby/plugins/worldedit", "WorldEdit"},
		{"configs/survival/plugins/coreprotect", "CoreProtect"},
		{"configs/lobby/plugins/decentholograms", "DecentHolograms"},
		{"configs/lobby/plugins/skinsrestorer", "SkinsRestorer"},
	}

	root := t.TempDir()
	m := &Manager{root: root}
	rollbackRoot := filepath.Join(root, "rollback")

	for _, tc := range cases {
		src := filepath.Join(root, tc.sourceDir)
		if err := os.MkdirAll(src, 0755); err != nil {
			t.Fatal(err)
		}
		cfgFile := filepath.Join(src, "config.yml")
		if err := os.WriteFile(cfgFile, []byte("dummy: true\n"), 0600); err != nil {
			t.Fatal(err)
		}

		if _, err := m.installConfigTree(tc.sourceDir, tc.targetDir, "lobby", rollbackRoot); err != nil {
			t.Fatalf("failed for %s -> %s: %v", tc.sourceDir, tc.targetDir, err)
		}

		dest := filepath.Join(root, "servers/lobby/plugins", tc.targetDir, "config.yml")
		if _, err := os.Stat(dest); err != nil {
			t.Errorf("expected destination %s to exist, err: %v", dest, err)
		}
	}
}

func TestManagerListPlanVerify(t *testing.T) {
	root := t.TempDir()
	manifestDir := filepath.Join(root, "configs/plugins")
	if err := os.MkdirAll(manifestDir, 0750); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(manifestDir, "manifest.yml")
	lockPath := filepath.Join(manifestDir, "lock.yml")

	manifestContent := `schema-version: 1
environment:
  minecraft-version: '1.21.11'
  backend: purpur
  proxy: velocity
  java: '25'
plugins:
  p_enabled:
    display-name: Enabled
    enabled: true
    source: modrinth
    project: proj1
    version: "1.0.0"
    version-id: v1
    platform: velocity
    servers: [proxy]
    release-channel: release
    update-policy: manual
    required-java: "25"
  p_disabled:
    display-name: Disabled
    enabled: false
    source: modrinth
    project: proj2
    version: "2.0.0"
    version-id: v2
    platform: velocity
    servers: [proxy]
    release-channel: release
    update-policy: manual
    required-java: "25"
`
	lockContent := `schema-version: 1
generated-by: test
plugins:
  p_enabled:
    project: proj1
    version: "1.0.0"
    version-id: v1
    source: modrinth
    download-url: https://cdn.modrinth.com/p1.jar
    filename: p1.jar
    sha256: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
    game-versions: [1.21.11]
    loaders: [velocity]
  p_disabled:
    project: proj2
    version: "2.0.0"
    version-id: v2
    source: modrinth
    download-url: https://cdn.modrinth.com/p2.jar
    filename: p2.jar
    sha256: abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789
    game-versions: [1.21.11]
    loaders: [velocity]
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, []byte(lockContent), 0600); err != nil {
		t.Fatal(err)
	}

	mgr := New(manifestPath, lockPath)

	// Validate
	if err := mgr.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// List
	var listBuf bytes.Buffer
	if err := mgr.List(&listBuf); err != nil {
		t.Fatalf("List failed: %v", err)
	}
	out := listBuf.String()
	if !strings.Contains(out, "p_enabled\tenabled\t1.0.0\tv1") {
		t.Errorf("List output missing p_enabled: %s", out)
	}
	if !strings.Contains(out, "p_disabled\tdisabled\t2.0.0\tv2") {
		t.Errorf("List output missing p_disabled: %s", out)
	}

	// Plan (initial - nothing installed)
	var planBuf bytes.Buffer
	if err := mgr.Plan(&planBuf); err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	planOut := planBuf.String()
	if !strings.Contains(planOut, "INSTALL\tp_enabled\t1.0.0") {
		t.Errorf("expected INSTALL for p_enabled: %s", planOut)
	}
	if !strings.Contains(planOut, "REMOVE\tp_disabled\tdisabled") {
		t.Errorf("expected REMOVE for p_disabled: %s", planOut)
	}

	// Setup installed state for UPDATE test
	pmDir := filepath.Join(root, ".plugin-manager")
	_ = os.MkdirAll(pmDir, 0700)
	installedYml := `schema-version: 1
plugins:
  p_enabled:
    version: "0.9.0"
    sha256: "different_hash_000000000000000000000000000000000000000000000000000000000"
`
	_ = os.WriteFile(filepath.Join(pmDir, "installed.yml"), []byte(installedYml), 0600)
	planBuf.Reset()
	if err := mgr.Plan(&planBuf); err != nil {
		t.Fatalf("Plan failed on update: %v", err)
	}
	if !strings.Contains(planBuf.String(), "UPDATE\tp_enabled\t1.0.0") {
		t.Errorf("expected UPDATE for p_enabled: %s", planBuf.String())
	}

	// Setup installed state for OK test
	installedMatching := `schema-version: 1
plugins:
  p_enabled:
    version: "1.0.0"
    sha256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
`
	_ = os.WriteFile(filepath.Join(pmDir, "installed.yml"), []byte(installedMatching), 0600)
	planBuf.Reset()
	if err := mgr.Plan(&planBuf); err != nil {
		t.Fatalf("Plan failed on match: %v", err)
	}
	if !strings.Contains(planBuf.String(), "OK\tp_enabled\t1.0.0") {
		t.Errorf("expected OK for p_enabled: %s", planBuf.String())
	}

	// VerifyInstalled - file missing
	if err := mgr.VerifyInstalled(); err == nil {
		t.Fatal("expected error on VerifyInstalled when file is missing")
	}

	// VerifyInstalled - create file with correct content & velocity descriptor
	targetFile := filepath.Join(root, "servers/proxy/plugins/p1.jar")
	_ = os.MkdirAll(filepath.Dir(targetFile), 0750)
	tf, err := os.Create(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(tf)
	w, _ := zw.Create("velocity-plugin.json")
	_, _ = w.Write([]byte(`{"id":"p_enabled"}`))
	_ = zw.Close()
	_ = tf.Close()

	// Compute real sha256 of targetFile and update lockfile
	h, _ := fileutilCopyHash(targetFile)
	updatedLock := strings.ReplaceAll(lockContent, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", h)
	_ = os.WriteFile(lockPath, []byte(updatedLock), 0600)

	if err := mgr.VerifyInstalled(); err != nil {
		t.Fatalf("VerifyInstalled failed on valid matching file: %v", err)
	}
}

func fileutilCopyHash(p string) (string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
