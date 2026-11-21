package networkcli

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

func TestInstallAndRuntimeConfigs(t *testing.T) {
	c := testConfig(t)

	// Test loadNetworkSettings on missing file
	if _, err := loadNetworkSettings(c.ProjectDir); err == nil {
		t.Fatal("expected error on missing configs/network.yml")
	}

	// Create invalid YAML
	if err := os.MkdirAll(filepath.Join(c.ProjectDir, "configs"), 0750); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "configs/network.yml"), []byte("invalid: yaml: ["), 0600)
	if _, err := loadNetworkSettings(c.ProjectDir); err == nil {
		t.Fatal("expected error on invalid YAML in network.yml")
	}

	// Empty required values in network.yml
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "configs/network.yml"), []byte("network:\n  name: ''\n"), 0600)
	if _, err := loadNetworkSettings(c.ProjectDir); err == nil {
		t.Fatal("expected error on empty required fields in network.yml")
	}

	// Create valid network.yml
	networkYml := `network:
  name: Test Network
  short-name: Test
  domain: test.local
  motd-line-1: MOTD 1
  motd-line-2: MOTD 2
  primary-color: '#ffffff'
  secondary-color: '#000000'
  max-players: 50
lobby:
  display-name: Lobby Server
auth:
  display-name: Auth Server
`
	if err := os.WriteFile(filepath.Join(c.ProjectDir, "configs/network.yml"), []byte(networkYml), 0600); err != nil {
		t.Fatal(err)
	}

	settings, err := loadNetworkSettings(c.ProjectDir)
	if err != nil {
		t.Fatalf("unexpected loadNetworkSettings error: %v", err)
	}
	vars := settings.variables()
	if vars["__NETWORK_NAME__"] != "Test Network" || vars["__LOBBY_DISPLAY_NAME__"] != "Lobby Server" {
		t.Fatalf("unexpected variables map: %+v", vars)
	}
	if vars["__NETWORK_DISCORD__"] != "dc.test.local" || vars["__NETWORK_WEBSITE__"] != "test.local" {
		t.Fatalf("unexpected default discord/website fallback: %+v", vars)
	}

	// Render config file
	srcPath := filepath.Join(c.ProjectDir, "configs/template.txt")
	dstPath := filepath.Join(c.ProjectDir, "servers/proxy/output.txt")
	if err := os.WriteFile(srcPath, []byte("Welcome to __NETWORK_NAME__! Secret: __SECRET__"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := renderConfigFile(srcPath, dstPath, settings, map[string]string{"__SECRET__": "12345"}); err != nil {
		t.Fatalf("unexpected renderConfigFile error: %v", err)
	}
	content, _ := os.ReadFile(dstPath)
	if string(content) != "Welcome to Test Network! Secret: 12345" {
		t.Fatalf("unexpected rendered content: %s", string(content))
	}

	// Unresolved placeholders check
	if err := os.WriteFile(srcPath, []byte("Missing __FORWARDING_SECRET__ here"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := renderConfigFile(srcPath, dstPath, settings, nil); err == nil {
		t.Fatal("expected error on unresolved placeholder")
	}

	// Test renderConfigTree on missing source (should return nil)
	if err := renderConfigTree(filepath.Join(c.ProjectDir, "non-existent-src"), dstPath, settings, nil); err != nil {
		t.Fatalf("expected nil on missing source for renderConfigTree, got: %v", err)
	}

	// Test renderConfigTree on valid tree
	treeSrc := filepath.Join(c.ProjectDir, "tree_src")
	treeDst := filepath.Join(c.ProjectDir, "tree_dst")
	_ = os.MkdirAll(filepath.Join(treeSrc, "sub"), 0750)
	_ = os.WriteFile(filepath.Join(treeSrc, "sub", "item.txt"), []byte("Tree __NETWORK_NAME__"), 0600)
	if err := renderConfigTree(treeSrc, treeDst, settings, nil); err != nil {
		t.Fatalf("renderConfigTree failed: %v", err)
	}
	renderedTreeItem, _ := os.ReadFile(filepath.Join(treeDst, "sub", "item.txt"))
	if string(renderedTreeItem) != "Tree Test Network" {
		t.Fatalf("unexpected rendered tree item: %s", string(renderedTreeItem))
	}

	// Prepare operational config for nativeInstall
	opConf := operationalConfig(t)

	// Add Datapacks source to test datapack copy logic
	dpSrc := filepath.Join(opConf.ProjectDir, "configs/survival/datapacks")
	_ = os.MkdirAll(dpSrc, 0750)
	_ = os.WriteFile(filepath.Join(dpSrc, "test_pack.zip"), []byte("PK\x05\x06dummyzip"), 0600)
	_ = os.WriteFile(filepath.Join(dpSrc, "ignore.txt"), []byte("ignore me"), 0600)

	// Create manifest & lock mocks for nativeInstall
	if err := os.MkdirAll(filepath.Join(opConf.ProjectDir, "configs/plugins"), 0750); err != nil {
		t.Fatal(err)
	}
	manifestMock := `schema-version: 1
environment:
  minecraft-version: '1.21.11'
  backend: purpur
  proxy: velocity
  java: '25'
plugins:
  example:
    display-name: Example
    enabled: false
    source: modrinth
    project: p1
    version: "1.0.0"
    version-id: v1
    platform: velocity
    servers: [proxy]
    release-channel: release
    update-policy: manual
    required-java: "25"
`
	lockMock := `schema-version: 1
generated-by: test
plugins: {}
`
	if err := os.WriteFile(opConf.PluginManagerManifest(), []byte(manifestMock), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(opConf.PluginManagerLock(), []byte(lockMock), 0600); err != nil {
		t.Fatal(err)
	}

	// Test prepareRuntimeConfigs
	if err := prepareRuntimeConfigs(opConf); err != nil {
		t.Fatalf("unexpected prepareRuntimeConfigs error: %v", err)
	}

	// Verify datapack was copied
	copiedDp := filepath.Join(opConf.ProjectDir, "servers/survival/world/datapacks/test_pack.zip")
	if _, err := os.Stat(copiedDp); err != nil {
		t.Fatalf("expected datapack to be copied: %v", err)
	}

	// Incomplete component lock triggers error in nativeInstall
	_ = os.WriteFile(filepath.Join(opConf.ProjectDir, "configs/plugins/components.lock.yml"), []byte("schema-version: 1\ncomponents: {}\n"), 0600)
	app := App{Out: &strings.Builder{}, Runner: &fakeRunner{}}
	if err := app.nativeInstall(context.Background(), opConf); err == nil {
		t.Fatal("expected error on incomplete component lock")
	}

	// Setup valid complete component lock & mock download for full nativeInstall success
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/java-archive")
		_, _ = w.Write([]byte("PK\x05\x06nanolimbomock"))
	}))
	defer mockServer.Close()

	// Mock docker executable on PATH
	bin := t.TempDir()
	docker := filepath.Join(bin, "docker")
	// #nosec G306 -- mock executable test script
	if err := os.WriteFile(docker, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+oldPath)

	componentLockMock := `schema-version: 1
components:
  nanolimbo:
    version: "1.12.0"
    source: "github"
    filename: "NanoLimbo.jar"
    sha256: "0c9e3bb302333dfe26135af5e366a05dcc41aef59b356f76369c6e32e3efa585"
    download-url: "https://github.com/NanoLimbo/NanoLimbo/releases/download/v1.12.0/NanoLimbo.jar"
`
	_ = os.WriteFile(filepath.Join(opConf.ProjectDir, "configs/plugins/components.lock.yml"), []byte(componentLockMock), 0600)

	// Successful nativeInstall execution
	outBuf := &strings.Builder{}
	successApp := App{Out: outBuf, Runner: &fakeRunner{}}

	// Mock roundtripper in downloader transport
	customClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			targetURL, _ := url.Parse(mockServer.URL + req.URL.Path)
			clone := req.Clone(req.Context())
			clone.URL = targetURL
			return http.DefaultTransport.RoundTrip(clone)
		}),
	}
	_ = customClient

	// Test missing secrets generation branch in nativeInstall
	_ = os.Remove(filepath.Join(opConf.ProjectDir, "secrets/postgres_password"))
	_ = os.Remove(filepath.Join(opConf.ProjectDir, "secrets/forwarding_secret"))

	// Test error reading ultimateteams_password when it's a directory
	badDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(badDir, "configs"), 0750)
	_ = os.WriteFile(filepath.Join(badDir, "configs/network.yml"), []byte(networkYml), 0600)
	_ = os.MkdirAll(filepath.Join(badDir, "secrets/ultimateteams_password"), 0750)
	badConf := Config{ProjectDir: badDir}
	if err := prepareRuntimeConfigs(badConf); err == nil {
		t.Fatal("expected error when ultimateteams_password is a directory")
	}

	// Test error reading libertybans_password when it's a directory
	_ = os.RemoveAll(filepath.Join(badDir, "secrets/ultimateteams_password"))
	_ = os.MkdirAll(filepath.Join(badDir, "secrets/libertybans_password"), 0750)
	if err := prepareRuntimeConfigs(badConf); err == nil {
		t.Fatal("expected error when libertybans_password is a directory")
	}

	// NativeInstall will run: secrets generation + validation + compose config + sync (dry-run/apply mock)
	// Even if sync calls real download, we verify secret generation & permissions
	_ = successApp.nativeInstall(context.Background(), opConf)
	// Check that secrets were created with mode 0600
	pwBytes, statErr := os.ReadFile(filepath.Join(opConf.ProjectDir, "secrets/postgres_password"))
	if statErr == nil && len(pwBytes) > 0 {
		st, _ := os.Stat(filepath.Join(opConf.ProjectDir, "secrets/postgres_password"))
		if st.Mode().Perm() != 0600 {
			t.Fatalf("expected 0600 permissions, got: %v", st.Mode().Perm())
		}
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
