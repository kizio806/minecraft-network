package pluginmanager

import (
	"archive/zip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id":"v1","project_id":"p1","version_number":"1.0.0","version_type":"release","date_published":"2026-01-01T00:00:00Z","game_versions":["1.21.11"],"loaders":["velocity"],"files":[{"url":"https://cdn.modrinth.com/p/v.jar","filename":"v.jar","size":3,"hashes":{"sha512":"abc"}}]}`))
	}))
	defer server.Close()
	e, err := (ModrinthProvider{BaseURL: server.URL, Client: server.Client()}).Resolve(context.Background(), "p1", "v1")
	if err != nil || e.VersionID != "v1" || e.SHA512 != "abc" {
		t.Fatalf("unexpected metadata: %+v %v", e, err)
	}
}

func TestGitHubProviderUsesExactTagAndDigest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.0.0","assets":[{"name":"plugin.jar","browser_download_url":"https://github.com/o/r/releases/download/v1.0.0/plugin.jar","size":3,"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`))
	}))
	defer server.Close()
	e, err := (GitHubProvider{BaseURL: server.URL, Client: server.Client()}).Resolve(context.Background(), "o/r#plugin.jar", "v1.0.0")
	if err != nil || e.SHA256 == "" || e.VersionID != "v1.0.0" {
		t.Fatalf("unexpected metadata: %+v %v", e, err)
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
	source := filepath.Join(root, "configs", "velocity", "plugins", "example")
	if err := os.MkdirAll(source, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "config.yml"), []byte("enabled: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	rollbackRoot := filepath.Join(root, ".rollback")
	changes, err := m.installConfigTree("configs/velocity/plugins/example", "example", "proxy", rollbackRoot)
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
	if _, err := m.installConfigTree("configs/velocity/plugins/example", "example", "proxy", rollbackRoot); err == nil {
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
	sourceDir := filepath.Join(root, "configs/purpur/plugins/deluxemenus/gui_menus")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	navPath := filepath.Join(sourceDir, "navigation.yml")
	if err := os.WriteFile(navPath, []byte("menu_title: 'Nav'\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mainCfg := filepath.Join(root, "configs/purpur/plugins/deluxemenus/config.yml")
	if err := os.WriteFile(mainCfg, []byte("debug: false\n"), 0644); err != nil {
		t.Fatal(err)
	}

	m := &Manager{root: root}
	rollbackRoot := filepath.Join(root, "rollback")
	_, err := m.installConfigTree("configs/purpur/plugins/deluxemenus", "DeluxeMenus", "lobby", rollbackRoot)
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
		{"configs/purpur/plugins/essentialsx", "Essentials"},
		{"configs/purpur/plugins/essentialsxspawn", "EssentialsSpawn"},
		{"configs/purpur/plugins/luckperms", "LuckPerms"},
		{"configs/purpur/plugins/placeholderapi", "PlaceholderAPI"},
		{"configs/purpur/plugins/multiverse-core", "Multiverse-Core"},
		{"configs/purpur/plugins/worldguardextraflags", "WorldGuardExtraFlags"},
		{"configs/purpur/plugins/worldguard", "WorldGuard"},
		{"configs/purpur/plugins/worldedit", "WorldEdit"},
		{"configs/purpur/plugins/commandpanels", "CommandPanels"},
		{"configs/purpur/plugins/decentholograms", "DecentHolograms"},
		{"configs/purpur/plugins/skinsrestorer", "SkinsRestorer"},
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
		if err := os.WriteFile(cfgFile, []byte("dummy: true\n"), 0644); err != nil {
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
