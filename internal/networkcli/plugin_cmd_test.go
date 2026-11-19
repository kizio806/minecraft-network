package networkcli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPluginCommand(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	app := App{Out: &out}

	// Create valid manifest & lockfile
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
    enabled: true
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
plugins:
  example:
    project: p1
    version: "1.0.0"
    version-id: v1
    source: modrinth
    download-url: "https://cdn.modrinth.com/p1/v1/example.jar"
    filename: "example.jar"
    sha256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    game-versions: ["1.21.11"]
    loaders: ["velocity"]
    dependencies: []
`
	if err := os.WriteFile(c.PluginManagerManifest(), []byte(manifestYAML), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.PluginManagerLock(), []byte(lockYAML), 0600); err != nil {
		t.Fatal(err)
	}

	// Empty args
	if err := app.plugins(context.Background(), c, []string{}, false); err == nil {
		t.Fatal("expected error on empty args")
	}

	// Validate
	if err := app.plugins(context.Background(), c, []string{"validate"}, false); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}

	// List
	out.Reset()
	if err := app.plugins(context.Background(), c, []string{"list"}, false); err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}

	// Plan
	out.Reset()
	if err := app.plugins(context.Background(), c, []string{"plan"}, false); err != nil {
		t.Fatalf("unexpected plan error: %v", err)
	}

	// Update check
	out.Reset()
	if err := app.plugins(context.Background(), c, []string{"update-check"}, false); err != nil {
		t.Fatalf("unexpected update-check error: %v", err)
	}

	// Unknown subcommand
	if err := app.plugins(context.Background(), c, []string{"unknown-subcmd"}, false); err == nil {
		t.Fatal("expected error on unknown subcommand")
	}

	// Sync dry-run
	out.Reset()
	if err := app.plugins(context.Background(), c, []string{"sync"}, true); err != nil {
		t.Fatalf("unexpected sync dry-run error: %v", err)
	}

	// Verify installed (missing jar error expected)
	if err := app.plugins(context.Background(), c, []string{"verify"}, false); err == nil {
		t.Fatal("expected error on missing jar for verify")
	}

	// Rollback when no rollback.yml exists
	if err := app.plugins(context.Background(), c, []string{"rollback"}, false); err == nil {
		t.Fatal("expected error on missing rollback snapshot")
	}

	// Components dry-run (creates mock components.lock.yml)
	compLock := `schema-version: 1
components:
  nanolimbo:
    version: "1.12.0"
    tag: v1.12.0
    source: github
    repository: Nan1t/NanoLimbo
    download-url: "https://example.com/NanoLimbo.jar"
    filename: "NanoLimbo.jar"
    sha256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    minecraft: ["1.21.11"]
`
	if err := os.WriteFile(filepath.Join(c.ProjectDir, "configs/plugins/components.lock.yml"), []byte(compLock), 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.plugins(context.Background(), c, []string{"components"}, true); err != nil {
		t.Fatalf("unexpected components dry-run error: %v", err)
	}
}
