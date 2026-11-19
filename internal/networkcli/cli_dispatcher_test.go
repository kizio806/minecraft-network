package networkcli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppExecute(t *testing.T) {
	c := operationalConfig(t)
	var out, errOut bytes.Buffer
	app := App{Out: &out, Err: &errOut, Runner: &fakeRunner{}}

	// Test ExitError methods
	exitErr := &ExitError{Code: 42, Err: errors.New("custom error")}
	if exitErr.Error() != "custom error" || exitErr.Unwrap() == nil {
		t.Fatal("unexpected ExitError behavior")
	}

	// Help and version
	if err := app.Execute(context.Background(), []string{"--help"}); err != nil {
		t.Fatalf("unexpected help error: %v", err)
	}
	if !strings.Contains(out.String(), "network [global flags] <command>") {
		t.Fatalf("unexpected help output: %s", out.String())
	}

	out.Reset()
	if err := app.Execute(context.Background(), []string{"--version"}); err != nil {
		t.Fatalf("unexpected version error: %v", err)
	}
	if !strings.Contains(out.String(), Version) {
		t.Fatalf("unexpected version output: %s", out.String())
	}

	// Nil writers and runner default initialization
	emptyApp := App{}
	if err := emptyApp.Execute(context.Background(), []string{"--help"}); err != nil {
		t.Fatalf("unexpected emptyApp help error: %v", err)
	}

	// Invalid global flags
	if err := app.Execute(context.Background(), []string{"--lines", "bad"}); err == nil {
		t.Fatal("expected error on invalid global flags")
	}

	// Unknown command
	if err := app.Execute(context.Background(), []string{"--project-dir", c.ProjectDir, "unknown_cmd"}); err == nil {
		t.Fatal("expected error on unknown command")
	}

	// Doctor via Execute
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
  test:
    display-name: Test
    enabled: true
    source: modrinth
    project: p
    version: "1"
    version-id: v
    platform: velocity
    servers: [proxy]
    release-channel: release
    update-policy: manual
    required-java: "25"
`
	lockYAML := `schema-version: 1
generated-by: test
plugins:
  test:
    project: p
    version: "1"
    version-id: v
    source: modrinth
    download-url: "https://cdn.modrinth.com/p/v/test.jar"
    filename: "test.jar"
    sha256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    game-versions: ["1.21.11"]
    loaders: ["velocity"]
    dependencies: []
`
	_ = os.WriteFile(c.PluginManagerManifest(), []byte(manifestYAML), 0600)
	_ = os.WriteFile(c.PluginManagerLock(), []byte(lockYAML), 0600)
	_ = os.WriteFile(filepath.Join(c.ProjectDir, ".env.example"), []byte("VELOCITY_VERSION=4.3.1\n"), 0600)
	out.Reset()
	if err := app.Execute(context.Background(), []string{"--project-dir", c.ProjectDir, "doctor"}); err != nil {
		t.Fatalf("unexpected execute doctor error: %v", err)
	}

	// Config show via Execute
	out.Reset()
	if err := app.Execute(context.Background(), []string{"--project-dir", c.ProjectDir, "config", "show"}); err != nil {
		t.Fatalf("unexpected execute config show error: %v", err)
	}

	// Status via Execute
	out.Reset()
	runner := &fakeRunner{
		output: map[string]string{
			"docker compose -f " + c.ComposeFile() + " ps --format json": `{"Service":"proxy","State":"running","Health":"healthy","Status":"Up 1h","Name":"p"}`,
		},
	}
	app.Runner = runner
	if err := app.Execute(context.Background(), []string{"--project-dir", c.ProjectDir, "status"}); err != nil {
		t.Fatalf("unexpected execute status error: %v", err)
	}

	// Logs via Execute
	out.Reset()
	if err := app.Execute(context.Background(), []string{"--project-dir", c.ProjectDir, "logs", "proxy"}); err != nil {
		t.Fatalf("unexpected execute logs error: %v", err)
	}

	// Console via Execute
	if err := app.Execute(context.Background(), []string{"--project-dir", c.ProjectDir, "console", "proxy"}); err != nil {
		t.Fatalf("unexpected execute console error: %v", err)
	}

	// Backup list via Execute
	out.Reset()
	if err := app.Execute(context.Background(), []string{"--project-dir", c.ProjectDir, "backup", "list"}); err != nil {
		t.Fatalf("unexpected execute backup list error: %v", err)
	}

	// Restore missing yes via Execute
	if err := app.Execute(context.Background(), []string{"--project-dir", c.ProjectDir, "restore", "dummy.tar.gz.age"}); err == nil {
		t.Fatal("expected restore error without --yes")
	}

	// Update check via Execute
	out.Reset()
	if err := app.Execute(context.Background(), []string{"--project-dir", c.ProjectDir, "update", "check"}); err != nil {
		t.Fatalf("unexpected execute update check error: %v", err)
	}

	// Plugins list via Execute
	out.Reset()
	manifestYAML = `schema-version: 1
environment:
  minecraft-version: "1.21.11"
  backend: purpur
  proxy: velocity
  java: "25"
plugins:
  test:
    display-name: Test
    enabled: true
    source: modrinth
    project: p
    version: "1"
    version-id: v
    platform: velocity
    servers: [proxy]
    release-channel: release
    update-policy: manual
    required-java: "25"
`
	lockYAML = `schema-version: 1
generated-by: test
plugins:
  test:
    project: p
    version: "1"
    version-id: v
    source: modrinth
    download-url: "https://cdn.modrinth.com/p/v/test.jar"
    filename: "test.jar"
    sha256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    game-versions: ["1.21.11"]
    loaders: ["velocity"]
    dependencies: []
`
	_ = os.WriteFile(c.PluginManagerManifest(), []byte(manifestYAML), 0600)
	_ = os.WriteFile(c.PluginManagerLock(), []byte(lockYAML), 0600)
	if err := app.Execute(context.Background(), []string{"--project-dir", c.ProjectDir, "plugins", "list"}); err != nil {
		t.Fatalf("unexpected execute plugins list error: %v", err)
	}
}
