package networkcli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kizio806/minecraft-network/internal/pluginmanager"
)

func TestInitCmdNonInteractive(t *testing.T) {
	c := operationalConfig(t)
	var out, errOut bytes.Buffer
	app := App{Out: &out, Err: &errOut, Runner: &fakeRunner{}}

	// Setup necessary mock config and example files
	if err := os.MkdirAll(filepath.Join(c.ProjectDir, "configs/plugins"), 0750); err != nil {
		t.Fatal(err)
	}
	manifestYAML := `schema-version: 1
environment:
  minecraft-version: "1.21.11"
  backend: purpur
  proxy: velocity
  java: "25"
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
	lockYAML := `schema-version: 1
generated-by: test
plugins: {}
`
	componentsYAML := `schema-version: 1
components: {}
`
	jarBytes := []byte("PK\x05\x06\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	jarHash := fmt.Sprintf("%x", sha256.Sum256(jarBytes))

	componentsLockYAML := fmt.Sprintf(`schema-version: 1
components:
  nanolimbo:
    version: "1.12.0"
    source: "github"
    filename: "NanoLimbo.jar"
    sha256: "%s"
    download-url: "https://github.com/NanoLimbo/NanoLimbo/releases/download/v1.12.0/NanoLimbo.jar"
`, jarHash)

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

	_ = os.WriteFile(c.PluginManagerManifest(), []byte(manifestYAML), 0600)
	_ = os.WriteFile(c.PluginManagerLock(), []byte(lockYAML), 0600)
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "configs/plugins/components.yml"), []byte(componentsYAML), 0600)
	_ = os.WriteFile(filepath.Join(c.ProjectDir, "configs/plugins/components.lock.yml"), []byte(componentsLockYAML), 0600)
	pinnedEnv := "VELOCITY_VERSION=4.3.1\n" +
		"VELOCITY_IMAGE=ghcr.io/velocity@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n" +
		"PURPUR_IMAGE=ghcr.io/purpur@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n" +
		"PURPUR_LOBBY_IMAGE=ghcr.io/purpur-lobby@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n" +
		"NANOLIMBO_IMAGE=ghcr.io/nanolimbo@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n" +
		"MC_MONITOR_IMAGE=ghcr.io/mc-monitor@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n"
	_ = os.WriteFile(filepath.Join(c.ProjectDir, ".env.example"), []byte(pinnedEnv), 0600)

	// Remove .env if it exists so init creates it
	_ = os.Remove(filepath.Join(c.ProjectDir, ".env"))

	// Mock docker executable on PATH
	bin := t.TempDir()
	docker := filepath.Join(bin, "docker")
	// #nosec G306 -- mock executable test script
	_ = os.WriteFile(docker, []byte("#!/bin/sh\nexit 0\n"), 0755)
	t.Setenv("PATH", bin+string(filepath.ListSeparator)+os.Getenv("PATH"))

	args := []string{
		"--project-dir", c.ProjectDir,
		"init",
		"--non-interactive",
		"--name", "Automated SMP",
		"--locale", "pl",
		"--max-players", "64",
	}

	if err := app.Execute(context.Background(), args); err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	// Verify .env was created
	if _, err := os.Stat(filepath.Join(c.ProjectDir, ".env")); err != nil {
		t.Errorf("expected .env to be created: %v", err)
	}

	// Verify configs/network.yml contains updated settings
	content, err := os.ReadFile(filepath.Join(c.ProjectDir, "configs", "network.yml"))
	if err != nil {
		t.Fatalf("failed to read network.yml: %v", err)
	}

	if !strings.Contains(string(content), "Automated SMP") {
		t.Errorf("network.yml does not contain updated name: %s", string(content))
	}
	if !strings.Contains(string(content), "locale: pl") {
		t.Errorf("network.yml does not contain updated locale: %s", string(content))
	}
}
