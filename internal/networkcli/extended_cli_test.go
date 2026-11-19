package networkcli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kizio806/minecraft-network/internal/pluginmanager"
)

func TestPrepareRuntimeConfigs_AllBranches(t *testing.T) {
	c := operationalConfig(t)

	// 1. Missing forwarding_secret
	_ = os.Remove(filepath.Join(c.ProjectDir, "secrets/forwarding_secret"))
	if err := prepareRuntimeConfigs(c); err == nil {
		t.Fatal("expected error on missing forwarding_secret")
	}

	// 2. Empty forwarding_secret
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "secrets/forwarding_secret"), []byte("   \n"), 0600)
	if err := prepareRuntimeConfigs(c); err == nil {
		t.Fatal("expected error on empty forwarding_secret")
	}

	// 3. Newline inside forwarding_secret
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "secrets/forwarding_secret"), []byte("abc\ndef"), 0600)
	if err := prepareRuntimeConfigs(c); err == nil {
		t.Fatal("expected error on forwarding_secret containing newline")
	}

	// Restore valid forwarding_secret
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "secrets/forwarding_secret"), []byte("forwarding-secret-1234"), 0600)

	// 4. Missing luckperms_password
	_ = os.Remove(filepath.Join(c.ProjectDir, "secrets/luckperms_password"))
	if err := prepareRuntimeConfigs(c); err == nil {
		t.Fatal("expected error on missing luckperms_password")
	}

	// 5. Empty luckperms_password
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "secrets/luckperms_password"), []byte(""), 0600)
	if err := prepareRuntimeConfigs(c); err == nil {
		t.Fatal("expected error on empty luckperms_password")
	}

	// Restore valid luckperms_password
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "secrets/luckperms_password"), []byte("luckperms-pw-1234"), 0600)

	// 6. Missing veloauth_password
	_ = os.Remove(filepath.Join(c.ProjectDir, "secrets/veloauth_password"))
	if err := prepareRuntimeConfigs(c); err == nil {
		t.Fatal("expected error on missing veloauth_password")
	}

	// 7. Empty veloauth_password
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "secrets/veloauth_password"), []byte("\r\n"), 0600)
	if err := prepareRuntimeConfigs(c); err == nil {
		t.Fatal("expected error on empty veloauth_password")
	}

	// Restore valid veloauth_password
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "secrets/veloauth_password"), []byte("veloauth-pw-1234"), 0600)

	// 8. Add optional secrets: ultimateteams_password and libertybans_password
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "secrets/ultimateteams_password"), []byte("uteams-pw-1234"), 0600)
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "secrets/libertybans_password"), []byte("liberty-pw-1234"), 0600)

	// 9. Add extra server config files: paper-world-defaults.yml and bukkit.yml
	for _, s := range []string{"lobby", "survival"} {
		_ = os.WriteFile(filepath.Join(c.ProjectDir, "configs", s, "paper-world-defaults.yml"), []byte("world: default\n"), 0600)
		_ = os.WriteFile(filepath.Join(c.ProjectDir, "configs", s, "bukkit.yml"), []byte("settings:\n  allow-end: true\n"), 0600)
	}

	// 10. Add plugin manifest with ConfigDir and ConfigTargetDir
	if err := os.MkdirAll(filepath.Join(c.ProjectDir, "configs/plugins"), 0750); err != nil {
		t.Fatal(err)
	}
	manifestWithPlugin := `schema-version: 1
environment:
  minecraft-version: '1.21.11'
  backend: purpur
  proxy: velocity
  java: '25'
plugins:
  customplug:
    display-name: CustomPlugin
    enabled: true
    source: modrinth
    project: p-custom
    version: "1.0.0"
    version-id: v1
    platform: paper
    servers: [lobby, survival]
    config-dir: configs/common/plugins/customplug
    config-target-dir: CustomPlug
    release-channel: release
    update-policy: manual
    required-java: "25"
`
	_ = os.WriteFile(c.PluginManagerManifest(), []byte(manifestWithPlugin), 0600)

	// Setup common config source
	commonSrc := filepath.Join(c.ProjectDir, "configs/common/plugins/customplug")
	_ = os.MkdirAll(commonSrc, 0750)
	_ = os.WriteFile(filepath.Join(commonSrc, "config.yml"), []byte("network: __NETWORK_NAME__\nsecret: __FORWARDING_SECRET__\n"), 0600)

	// Setup server-specific config source for survival: configs/survival/plugins/customplug
	serverSpecificSrc := filepath.Join(c.ProjectDir, "configs/survival/plugins/customplug")
	_ = os.MkdirAll(serverSpecificSrc, 0750)
	_ = os.WriteFile(filepath.Join(serverSpecificSrc, "config.yml"), []byte("server_override: survival\nnetwork: __NETWORK_NAME__\n"), 0600)

	// Create staleDir to test stale cleanup: servers/lobby/plugins/customplug
	staleDir := filepath.Join(c.ProjectDir, "servers/lobby/plugins/customplug")
	_ = os.MkdirAll(staleDir, 0750)
	_ = os.WriteFile(filepath.Join(staleDir, "old.txt"), []byte("stale"), 0600)

	// 11. Add survival datapacks directory with zip and non-zip and sub-dir
	dpSrc := filepath.Join(c.ProjectDir, "configs/survival/datapacks")
	_ = os.MkdirAll(filepath.Join(dpSrc, "subfolder"), 0750)
	_ = os.WriteFile(filepath.Join(dpSrc, "mypack.zip"), []byte("PK\x05\x06datapackcontents"), 0600)
	_ = os.WriteFile(filepath.Join(dpSrc, "README.txt"), []byte("ignore me"), 0600)

	// 12. Create existing server plugin config files with various extensions (.conf, .json, .properties, .jar)
	proxyPluginsDir := filepath.Join(c.ProjectDir, "servers/proxy/plugins")
	_ = os.MkdirAll(proxyPluginsDir, 0750)
	_ = os.WriteFile(filepath.Join(proxyPluginsDir, "test.conf"), []byte("title = \"__NETWORK_NAME__\"\nsecret = \"__VELOAUTH_PASSWORD__\"\n"), 0600)
	_ = os.WriteFile(filepath.Join(proxyPluginsDir, "test.properties"), []byte("motd=__NETWORK_NAME__\n"), 0600)
	_ = os.WriteFile(filepath.Join(proxyPluginsDir, "test.json"), []byte("{\"name\": \"__NETWORK_NAME__\"}\n"), 0600)
	_ = os.WriteFile(filepath.Join(proxyPluginsDir, "ignored.jar"), []byte("dummyjar"), 0600)
	_ = os.WriteFile(filepath.Join(proxyPluginsDir, "nochange.conf"), []byte("static = true\n"), 0600)

	// Run prepareRuntimeConfigs
	if err := prepareRuntimeConfigs(c); err != nil {
		t.Fatalf("unexpected prepareRuntimeConfigs error: %v", err)
	}

	// Verify stale dir was cleaned up
	if _, err := os.Stat(staleDir); !os.IsNotExist(err) {
		t.Fatalf("expected stale directory to be removed: %s", staleDir)
	}

	// Verify target custom dir was created
	destLobby := filepath.Join(c.ProjectDir, "servers/lobby/plugins/CustomPlug/config.yml")
	if b, err := os.ReadFile(destLobby); err != nil || !strings.Contains(string(b), "network: Test Network") {
		t.Fatalf("unexpected lobby config: %s (err: %v)", string(b), err)
	}

	// Verify survival override was applied
	destSurvival := filepath.Join(c.ProjectDir, "servers/survival/plugins/CustomPlug/config.yml")
	if b, err := os.ReadFile(destSurvival); err != nil || !strings.Contains(string(b), "server_override: survival") {
		t.Fatalf("unexpected survival override: %s (err: %v)", string(b), err)
	}

	// Verify datapack was copied
	dpDst := filepath.Join(c.ProjectDir, "servers/survival/world/datapacks/mypack.zip")
	if _, err := os.Stat(dpDst); err != nil {
		t.Fatalf("expected datapack to be copied: %v", err)
	}

	// Verify placeholder replacement in proxy plugin configs
	confBytes, err := os.ReadFile(filepath.Join(proxyPluginsDir, "test.conf"))
	if err != nil || !strings.Contains(string(confBytes), "title = \"Test Network\"") || !strings.Contains(string(confBytes), "secret = \"veloauth-pw-1234\"") {
		t.Fatalf("unexpected test.conf replacement: %s", string(confBytes))
	}

	// Test binary file in renderConfigFile
	binSrc := filepath.Join(c.ProjectDir, "configs/icon.png")
	binDst := filepath.Join(c.ProjectDir, "servers/proxy/icon.png")
	_ = os.WriteFile(binSrc, []byte("\x89PNG\r\n\x1a\n"), 0600)
	settings, _ := loadNetworkSettings(c.ProjectDir)
	if err := renderConfigFile(binSrc, binDst, settings, nil); err != nil {
		t.Fatalf("renderConfigFile binary failed: %v", err)
	}
	if b, err := os.ReadFile(binDst); err != nil || string(b) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("unexpected binary destination content: %s", string(b))
	}

	// Test unclosed placeholder token in renderConfigFile
	badSrc := filepath.Join(c.ProjectDir, "configs/bad.txt")
	badDst := filepath.Join(c.ProjectDir, "servers/proxy/bad.txt")
	_ = os.WriteFile(badSrc, []byte("Unfinished __NETWORK_MISSING placeholder"), 0600)
	if err := renderConfigFile(badSrc, badDst, settings, nil); err == nil {
		t.Fatal("expected error on unclosed placeholder prefix")
	}

	// Test missing source file in renderConfigFile
	if err := renderConfigFile(filepath.Join(c.ProjectDir, "missing.txt"), badDst, settings, nil); err == nil {
		t.Fatal("expected error on missing source file")
	}
}

func TestNativeInstall_ErrorPaths(t *testing.T) {
	c := operationalConfig(t)
	runner := &fakeRunner{}
	app := App{Out: io.Discard, Runner: runner}

	// 1. Secrets validation error
	_ = os.Remove(filepath.Join(c.ProjectDir, "secrets/postgres_password"))
	_ = os.Chmod(filepath.Join(c.ProjectDir, "secrets"), 0755) // Invalid permissions for ValidateSecrets
	if err := app.nativeInstall(context.Background(), c); err == nil {
		t.Fatal("expected error on invalid secrets directory permissions")
	}
	_ = os.Chmod(filepath.Join(c.ProjectDir, "secrets"), 0700)
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "secrets/postgres_password"), []byte("pw\n"), 0600)

	// 2. Environment file validation error
	_ = os.WriteFile(filepath.Join(c.ProjectDir, ".env"), []byte("INVALID_ENV\n"), 0600)
	if err := app.nativeInstall(context.Background(), c); err == nil {
		t.Fatal("expected error on invalid .env")
	}

	// Restore valid .env
	validEnv := "VELOCITY_VERSION=4.3.1\nVELOCITY_IMAGE=x/y@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\nPURPUR_IMAGE=x/y@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\nPURPUR_LOBBY_IMAGE=x/y@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee\nNANOLIMBO_IMAGE=x/y@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc\nMC_MONITOR_IMAGE=x/y@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd\n"
	_ = os.WriteFile(filepath.Join(c.ProjectDir, ".env"), []byte(validEnv), 0600)

	// 3. Compose failure
	runner.err = errors.New("compose config error")
	if err := app.nativeInstall(context.Background(), c); err == nil {
		t.Fatal("expected error on compose failure")
	}
	runner.err = nil

	// 4. Missing plugin manifest causes Sync failure
	_ = os.Remove(c.PluginManagerManifest())
	if err := app.nativeInstall(context.Background(), c); err == nil {
		t.Fatal("expected error on missing plugin manifest")
	}
}

func TestInitCmd_Extended(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	app := App{Out: &out, Runner: &fakeRunner{}}

	// 1. Help flag
	out.Reset()
	if err := app.initCmd(context.Background(), c, []string{"--help"}); err != nil {
		t.Fatalf("unexpected error on --help: %v", err)
	}
	if !strings.Contains(out.String(), "Usage: network init [flags]") {
		t.Fatalf("expected usage info: %s", out.String())
	}

	out.Reset()
	if err := app.initCmd(context.Background(), c, []string{"-h"}); err != nil {
		t.Fatalf("unexpected error on -h: %v", err)
	}

	// 2. Flags: --name, --domain, --theme, --locale, --max-players (valid and invalid)
	// Create mock manifest and lock
	_ = os.MkdirAll(filepath.Join(c.ProjectDir, "configs/plugins"), 0750)
	manifestYAML := `schema-version: 1
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
	lockYAML := "schema-version: 1\ngenerated-by: test\nplugins: {}\n"

	jarBytes := []byte("PK\x05\x06\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	jarHash := fmt.Sprintf("%x", sha256.Sum256(jarBytes))

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/java-archive")
		_, _ = w.Write(jarBytes)
	}))
	defer mockServer.Close()

	oldTransport := pluginmanager.DefaultHTTPClient.Transport
	pluginmanager.DefaultHTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		targetURL, _ := url.Parse(mockServer.URL + req.URL.Path)
		clone := req.Clone(req.Context())
		clone.URL = targetURL
		return http.DefaultTransport.RoundTrip(clone)
	})
	defer func() { pluginmanager.DefaultHTTPClient.Transport = oldTransport }()

	compLock := fmt.Sprintf(`schema-version: 1
components:
  nanolimbo:
    version: "1.12.0"
    source: "github"
    filename: "NanoLimbo.jar"
    sha256: "%s"
    download-url: "https://github.com/NanoLimbo/NanoLimbo/releases/download/v1.12.0/NanoLimbo.jar"
`, jarHash)

	_ = os.WriteFile(c.PluginManagerManifest(), []byte(manifestYAML), 0600)
	_ = os.WriteFile(c.PluginManagerLock(), []byte(lockYAML), 0600)
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "configs/plugins/components.lock.yml"), []byte(compLock), 0600)

	// Mock docker executable on PATH
	bin := t.TempDir()
	docker := filepath.Join(bin, "docker")
	_ = os.WriteFile(docker, []byte("#!/bin/sh\nexit 0\n"), 0755)
	t.Setenv("PATH", bin+string(filepath.ListSeparator)+os.Getenv("PATH"))

	out.Reset()
	initArgs := []string{
		"--non-interactive",
		"--name", "My Cool Network",
		"--domain", "mc.example.com",
		"--theme", "cyberpunk-neon",
		"--locale", "pl",
		"--max-players", "250",
		"--max-players", "not-a-number", // should be ignored, keeping 250
	}
	if err := app.initCmd(context.Background(), c, initArgs); err != nil {
		t.Fatalf("unexpected initCmd error: %v", err)
	}

	// Verify configs/network.yml updated
	netCfg, err := loadNetworkSettings(c.ProjectDir)
	if err != nil {
		t.Fatalf("loadNetworkSettings failed: %v", err)
	}
	if netCfg.Network.Name != "My Cool Network" || netCfg.Network.Domain != "mc.example.com" || netCfg.Network.MaxPlayers != 250 {
		t.Fatalf("unexpected network settings: %+v", netCfg)
	}

	// 3. Lock failure during init
	lockFile, err := AcquireLock(c.ProjectDir)
	if err != nil {
		t.Fatal(err)
	}
	defer lockFile.Close()

	if err := app.initCmd(context.Background(), c, []string{"--non-interactive"}); err == nil {
		t.Fatal("expected lock contention error during initCmd")
	}
}

func TestDoctor_Extended(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer

	// Create required manifest, lockfile, env.example mocks
	_ = os.MkdirAll(filepath.Join(c.ProjectDir, "configs/plugins"), 0750)
	_ = os.WriteFile(c.PluginManagerManifest(), []byte("schema-version: 1\n"), 0600)
	_ = os.WriteFile(c.PluginManagerLock(), []byte("schema-version: 1\n"), 0600)
	_ = os.WriteFile(filepath.Join(c.ProjectDir, ".env.example"), []byte("VELOCITY_VERSION=4.3.1\n"), 0600)

	runner := &fakeRunner{
		output: map[string]string{
			"docker version --format {{.Server.Version}}":       "26.1.4",
			"docker compose -f " + c.ComposeFile() + " version": "2.27.1",
		},
	}
	app := App{Out: &out, Runner: runner}

	// 1. Missing compose file
	_ = os.Remove(c.ComposeFile())
	if err := app.doctor(context.Background(), c); err == nil {
		t.Fatal("expected doctor error when docker-compose.yml is missing")
	}
	// Restore compose file
	_ = os.WriteFile(c.ComposeFile(), []byte("include: []\n"), 0600)

	// 2. Missing .env.example
	_ = os.Remove(filepath.Join(c.ProjectDir, ".env.example"))
	if err := app.doctor(context.Background(), c); err == nil {
		t.Fatal("expected doctor error when .env.example is missing")
	}
	_ = os.WriteFile(filepath.Join(c.ProjectDir, ".env.example"), []byte("VELOCITY_VERSION=4.3.1\n"), 0600)

	// 3. Compose version command failure
	runner.output["docker compose -f "+c.ComposeFile()+" version"] = ""
	runner.failContains = "docker compose"
	if err := app.doctor(context.Background(), c); err == nil {
		t.Fatal("expected doctor error when compose version fails")
	}
	runner.failContains = ""

	// 4. Invalid .env permissions
	_ = os.Chmod(filepath.Join(c.ProjectDir, ".env"), 0666)
	if err := app.doctor(context.Background(), c); err == nil {
		t.Fatal("expected doctor error on insecure .env permissions")
	}
	_ = os.Chmod(filepath.Join(c.ProjectDir, ".env"), 0600)
}

func TestConfigCmd_Extended(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	runner := &fakeRunner{}
	app := App{Out: &out, Runner: runner}

	// Validate with compose failure
	runner.err = errors.New("compose validation failed")
	if err := app.config(c, []string{"validate"}); err == nil {
		t.Fatal("expected error on compose validation failure")
	}
	runner.err = nil

	// Validate with App.Runner == nil (initializes ExecRunner)
	// We mock docker on PATH to test real ExecRunner
	bin := t.TempDir()
	docker := filepath.Join(bin, "docker")
	_ = os.WriteFile(docker, []byte("#!/bin/sh\nexit 0\n"), 0755)
	t.Setenv("PATH", bin+string(filepath.ListSeparator)+os.Getenv("PATH"))

	emptyApp := App{Out: &out}
	if err := emptyApp.config(c, []string{"validate"}); err != nil {
		t.Fatalf("unexpected validate error with nil runner: %v", err)
	}

	// Validate with bad environment file
	_ = os.WriteFile(filepath.Join(c.ProjectDir, ".env"), []byte("VELOCITY_IMAGE=itzg/minecraft:latest\n"), 0600)
	if err := app.config(c, []string{"validate"}); err == nil {
		t.Fatal("expected error on invalid .env in config validate")
	}
}

func TestHealth_Extended(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer

	// 1. Compose config failure
	runner := &fakeRunner{err: errors.New("compose config error")}
	app := App{Out: &out, Runner: runner}
	if err := app.nativeHealth(context.Background(), c); err == nil {
		t.Fatal("expected error on compose failure")
	}

	// 2. Degraded services check in json format
	c.Output = "json"
	runner.err = nil
	runner.output = map[string]string{
		"docker compose -f " + c.ComposeFile() + " config --quiet": "",
		"docker compose -f " + c.ComposeFile() + " ps --format json": "{\"Service\":\"proxy\",\"State\":\"running\",\"Health\":\"healthy\"}\n" +
			"{\"Service\":\"auth\",\"State\":\"running\",\"Health\":\"healthy\"}\n" +
			"{\"Service\":\"lobby\",\"State\":\"running\",\"Health\":\"unhealthy\"}\n" +
			"{\"Service\":\"survival\",\"State\":\"running\",\"Health\":\"healthy\"}\n",
	}
	out.Reset()
	if err := app.nativeHealth(context.Background(), c); err == nil {
		t.Fatal("expected error on degraded service")
	}
	if !strings.Contains(out.String(), `"healthy":false`) {
		t.Fatalf("expected json health report with healthy: false, got: %s", out.String())
	}
}

func TestLifecycle_Extended(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	runner := &fakeRunner{output: map[string]string{}}
	app := App{Out: &out, Runner: runner}

	// 1. Invalid service name
	if err := app.lifecycle(context.Background(), c, "start", []string{"invalid_service"}); err == nil {
		t.Fatal("expected error on unsupported service")
	}

	// 2. Secrets validation failure
	_ = os.Remove(filepath.Join(c.ProjectDir, "secrets/postgres_password"))
	if err := app.lifecycle(context.Background(), c, "start", []string{"proxy"}); err == nil {
		t.Fatal("expected error on missing secret")
	}
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "secrets/postgres_password"), []byte("pw\n"), 0600)

	// 3. Environment validation failure
	_ = os.WriteFile(filepath.Join(c.ProjectDir, ".env"), []byte("VELOCITY_IMAGE=latest\n"), 0600)
	if err := app.lifecycle(context.Background(), c, "start", []string{"proxy"}); err == nil {
		t.Fatal("expected error on invalid .env")
	}
	validEnv := "VELOCITY_VERSION=4.3.1\nVELOCITY_IMAGE=x/y@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\nPURPUR_IMAGE=x/y@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\nPURPUR_LOBBY_IMAGE=x/y@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee\nNANOLIMBO_IMAGE=x/y@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc\nMC_MONITOR_IMAGE=x/y@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd\n"
	_ = os.WriteFile(filepath.Join(c.ProjectDir, ".env"), []byte(validEnv), 0600)

	// 4. Action: restart
	out.Reset()
	if err := app.lifecycle(context.Background(), c, "restart", []string{"proxy"}); err != nil {
		t.Fatalf("unexpected restart error: %v", err)
	}
	if !strings.Contains(out.String(), "restart completed: proxy") {
		t.Fatalf("unexpected restart output: %s", out.String())
	}

	// 5. Action: stop
	out.Reset()
	if err := app.lifecycle(context.Background(), c, "stop", []string{"lobby"}); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
	if !strings.Contains(out.String(), "stop completed: lobby") {
		t.Fatalf("unexpected stop output: %s", out.String())
	}

	// 6. Monitoring service triggers monitoring profile
	runner.calls = nil
	if err := app.lifecycle(context.Background(), c, "stop", []string{"grafana"}); err != nil {
		t.Fatalf("unexpected stop grafana error: %v", err)
	}
	foundMonitoringProfile := false
	for _, call := range runner.calls {
		s := strings.Join(call, " ")
		if strings.Contains(s, "--profile monitoring") {
			foundMonitoringProfile = true
			break
		}
	}
	if !foundMonitoringProfile {
		t.Fatal("expected --profile monitoring for grafana")
	}

	// 7. Compose failure during lifecycle
	runner.err = errors.New("compose error")
	if err := app.lifecycle(context.Background(), c, "stop", []string{"proxy"}); err == nil {
		t.Fatal("expected error on compose failure")
	}
}

func TestLogs_Extended(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	runner := &fakeRunner{output: map[string]string{
		"docker compose -f " + c.ComposeFile() + " logs --tail 50 --timestamps --follow proxy": "proxy logs line 1\n",
	}}
	app := App{Out: &out, Runner: runner}

	// 1. Invalid service name
	if err := app.logs(context.Background(), c, []string{"invalid_service"}, false, 100); err == nil {
		t.Fatal("expected error on unsupported service")
	}

	// 2. Valid logs call with follow and custom tail
	out.Reset()
	if err := app.logs(context.Background(), c, []string{"proxy"}, true, 50); err != nil {
		t.Fatalf("unexpected logs error: %v", err)
	}
	if !strings.Contains(out.String(), "proxy logs line 1") {
		t.Fatalf("expected logs output: %s", out.String())
	}

	// 3. Compose failure
	runner.err = errors.New("compose logs error")
	if err := app.logs(context.Background(), c, []string{"proxy"}, false, 100); err == nil {
		t.Fatal("expected error on compose logs failure")
	}
}

func TestConsole_Extended(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	runner := &fakeRunner{}
	app := App{Out: &out, Runner: runner}

	// 1. RunInteractive error
	runner.err = errors.New("terminal attach failed")
	if err := app.console(context.Background(), c, []string{"proxy"}); err == nil {
		t.Fatal("expected error on RunInteractive failure")
	}
}

func TestUpdate_Extended(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	runner := &fakeRunner{output: map[string]string{
		"docker compose -f " + c.ComposeFile() + " ps --format json": "{\"Service\":\"proxy\",\"State\":\"running\",\"Health\":\"healthy\"}\n{\"Service\":\"auth\",\"State\":\"running\",\"Health\":\"healthy\"}\n{\"Service\":\"lobby\",\"State\":\"running\",\"Health\":\"healthy\"}\n{\"Service\":\"survival\",\"State\":\"running\",\"Health\":\"healthy\"}\n",
	}}
	app := App{Out: &out, Runner: runner}

	// 1. Update check with missing .env
	_ = os.Remove(filepath.Join(c.ProjectDir, ".env"))
	if err := app.updateCheck(context.Background(), c); err == nil {
		t.Fatal("expected error on missing .env in updateCheck")
	}

	// Restore valid .env
	validEnv := "VELOCITY_VERSION=4.3.1\nVELOCITY_IMAGE=x/y@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\nPURPUR_IMAGE=x/y@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\nPURPUR_LOBBY_IMAGE=x/y@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee\nNANOLIMBO_IMAGE=x/y@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc\nMC_MONITOR_IMAGE=x/y@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd\n"
	_ = os.WriteFile(filepath.Join(c.ProjectDir, ".env"), []byte(validEnv), 0600)

	// 2. Update check text output
	_ = os.MkdirAll(filepath.Join(c.ProjectDir, "configs/plugins"), 0750)
	manifestYAML := `schema-version: 1
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
	lockYAML := "schema-version: 1\ngenerated-by: test\nplugins: {}\n"
	_ = os.WriteFile(c.PluginManagerManifest(), []byte(manifestYAML), 0600)
	_ = os.WriteFile(c.PluginManagerLock(), []byte(lockYAML), 0600)

	out.Reset()
	c.Output = "table"
	if err := app.updateCheck(context.Background(), c); err != nil {
		t.Fatalf("unexpected updateCheck table error: %v", err)
	}
	if !strings.Contains(out.String(), "VELOCITY_IMAGE") || !strings.Contains(out.String(), "Plugin check:") {
		t.Fatalf("unexpected updateCheck output: %s", out.String())
	}

	// 3. nativeUpdate: dry=true with compose failure
	runner.err = errors.New("compose config error")
	if err := app.nativeUpdate(context.Background(), c, true, false); err == nil {
		t.Fatal("expected error on compose config failure during dry-run update")
	}
	runner.err = nil

	// 4. nativeUpdate: neither dry nor yes
	if err := app.nativeUpdate(context.Background(), c, false, false); err == nil {
		t.Fatal("expected error when neither dry nor yes")
	}

	// 5. nativeUpdate: up failure
	runner.failContains = " up "
	if err := app.nativeUpdate(context.Background(), c, false, true); err == nil {
		t.Fatal("expected error on compose up failure")
	}
	runner.failContains = ""
}

func TestPlugins_Extended(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	app := App{Out: &out}

	// Create valid manifest & lock
	_ = os.MkdirAll(filepath.Join(c.ProjectDir, "configs/plugins"), 0750)
	manifestYAML := `schema-version: 1
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
	lockYAML := "schema-version: 1\ngenerated-by: test\nplugins: {}\n"
	_ = os.WriteFile(c.PluginManagerManifest(), []byte(manifestYAML), 0600)
	_ = os.WriteFile(c.PluginManagerLock(), []byte(lockYAML), 0600)

	// Lock contention for sync
	lockFile, err := AcquireLock(c.ProjectDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.plugins(context.Background(), c, []string{"sync"}, false); err == nil {
		t.Fatal("expected lock error on plugins sync")
	}

	// Lock contention for rollback
	if err := app.plugins(context.Background(), c, []string{"rollback"}, false); err == nil {
		t.Fatal("expected lock error on plugins rollback")
	}

	// Lock contention for components
	if err := app.plugins(context.Background(), c, []string{"components"}, false); err == nil {
		t.Fatal("expected lock error on plugins components")
	}
	lockFile.Close()

	// Empty args
	if err := app.nativePlugin(context.Background(), c, []string{}, false); err == nil {
		t.Fatal("expected error on empty nativePlugin args")
	}
}

func TestFlags_Extended(t *testing.T) {
	// 1. --project-dir missing value
	if _, _, _, _, _, _, _, _, err := parseGlobal([]string{"--project-dir"}); err == nil {
		t.Fatal("expected error on missing --project-dir value")
	}

	// 2. --output missing value
	if _, _, _, _, _, _, _, _, err := parseGlobal([]string{"--output"}); err == nil {
		t.Fatal("expected error on missing --output value")
	}

	// 3. --lines missing value
	if _, _, _, _, _, _, _, _, err := parseGlobal([]string{"--lines"}); err == nil {
		t.Fatal("expected error on missing --lines value")
	}

	// 4. --lines negative or invalid value
	if _, _, _, _, _, _, _, _, err := parseGlobal([]string{"--lines", "-5"}); err == nil {
		t.Fatal("expected error on negative --lines")
	}
	if _, _, _, _, _, _, _, _, err := parseGlobal([]string{"--lines", "invalid"}); err == nil {
		t.Fatal("expected error on non-integer --lines")
	}

	// 5. Valid global flags
	proj, out, yes, dry, follow, mon, lines, rest, err := parseGlobal([]string{
		"--project-dir", "/tmp/p",
		"--output", "json",
		"--yes",
		"--dry-run",
		"--follow",
		"--monitoring",
		"--lines", "200",
		"start", "proxy",
	})
	if err != nil {
		t.Fatalf("unexpected parseGlobal error: %v", err)
	}
	if proj != "/tmp/p" || out != "json" || !yes || !dry || !follow || !mon || lines != 200 || len(rest) != 2 || rest[0] != "start" || rest[1] != "proxy" {
		t.Fatalf("unexpected parsed values: proj=%s out=%s yes=%t dry=%t follow=%t mon=%t lines=%d rest=%v",
			proj, out, yes, dry, follow, mon, lines, rest)
	}

	// 6. stripFlag
	args := []string{"start", "--yes", "proxy"}
	stripped, found := stripFlag(args, "--yes")
	if !found || len(stripped) != 2 || stripped[0] != "start" || stripped[1] != "proxy" {
		t.Fatalf("unexpected stripFlag result: found=%t stripped=%v", found, stripped)
	}
}

func TestRunner_OperationContext(t *testing.T) {
	// timeout <= 0 defaults to 30 min
	ctx, cancel := operationContext(context.Background(), 0)
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) < 25*time.Minute {
		t.Fatalf("expected default ~30m timeout, got deadline: %v", deadline)
	}

	// timeout > 0
	ctx2, cancel2 := operationContext(context.Background(), 5*time.Second)
	defer cancel2()
	deadline2, ok2 := ctx2.Deadline()
	if !ok2 || time.Until(deadline2) > 6*time.Second {
		t.Fatalf("expected ~5s timeout, got deadline: %v", deadline2)
	}
}

func TestStatus_Extended(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer

	// 1. Compose error
	runner := &fakeRunner{err: errors.New("compose ps failed")}
	app := App{Out: &out, Runner: runner}
	if err := app.status(context.Background(), c); err == nil {
		t.Fatal("expected error on compose ps failure")
	}

	// 2. Malformed json error
	runner.err = nil
	runner.output = map[string]string{
		"docker compose -f " + c.ComposeFile() + " ps --format json": "not-valid-json",
	}
	if err := app.status(context.Background(), c); err == nil {
		t.Fatal("expected error on malformed json ps output")
	}
}

func TestRestore_ExtendedPaths(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	runner := &fakeRunner{}
	app := App{Out: &out, Runner: runner}

	// 1. Missing backup arg
	if err := app.restore(context.Background(), c, []string{}, true); err == nil {
		t.Fatal("expected error on missing backup argument")
	}

	// 2. Missing --yes flag
	if err := app.restore(context.Background(), c, []string{"backup.tar.gz.age"}, false); err == nil {
		t.Fatal("expected error on missing --yes flag")
	}

	// 3. Absolute path / path with slash
	if err := app.restore(context.Background(), c, []string{"/tmp/backups/backup.tar.gz.age", "--yes"}, false); err == nil {
		// Restore should fail because file doesn't exist, but it tested path branch
		t.Fatal("expected restore error on non-existent archive")
	}

	// 4. Relative path without slash
	if err := app.restore(context.Background(), c, []string{"backup.tar.gz.age", "--yes"}, false); err == nil {
		t.Fatal("expected restore error on non-existent relative archive")
	}
}

func TestBackup_NoArgs(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	app := App{Out: &out, Runner: &fakeRunner{output: map[string]string{}}}

	if err := app.backup(context.Background(), c, []string{}); err != nil {
		t.Fatalf("unexpected backup error: %v", err)
	}
	if !strings.Contains(out.String(), "Backup created:") {
		t.Fatalf("expected backup created message, got: %s", out.String())
	}
}

func TestAppExecute_AllCommands(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	runner := &fakeRunner{
		output: map[string]string{
			"docker compose -f " + c.ComposeFile() + " config --quiet": "",
			"docker compose -f " + c.ComposeFile() + " ps --format json": "{\"Service\":\"proxy\",\"State\":\"running\",\"Health\":\"healthy\"}\n" +
				"{\"Service\":\"auth\",\"State\":\"running\",\"Health\":\"healthy\"}\n" +
				"{\"Service\":\"lobby\",\"State\":\"running\",\"Health\":\"healthy\"}\n" +
				"{\"Service\":\"survival\",\"State\":\"running\",\"Health\":\"healthy\"}\n",
		},
	}
	app := App{Out: &out, Runner: runner}

	// 1. health via Execute
	out.Reset()
	if err := app.Execute(context.Background(), []string{"--project-dir", c.ProjectDir, "health"}); err != nil {
		t.Fatalf("unexpected health via Execute error: %v", err)
	}

	// 2. start, stop, restart via Execute
	for _, cmd := range []string{"start", "stop", "restart"} {
		out.Reset()
		if err := app.Execute(context.Background(), []string{"--project-dir", c.ProjectDir, cmd, "proxy"}); err != nil {
			t.Fatalf("unexpected %s via Execute error: %v", cmd, err)
		}
	}

	// 3. install via Execute with lock contention
	lock, err := AcquireLock(c.ProjectDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Execute(context.Background(), []string{"--project-dir", c.ProjectDir, "install"}); err == nil {
		t.Fatal("expected error on locked install via Execute")
	}
	lock.Close()
}
