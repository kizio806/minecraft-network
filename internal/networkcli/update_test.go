package networkcli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateCommand(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	app := App{Out: &out, Runner: &fakeRunner{}}

	// Missing flags
	if err := app.update(context.Background(), c, []string{}, false, false); err == nil {
		t.Fatal("expected error on update without flags")
	}

	// Update check json
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

	c.Output = "json"
	if err := app.update(context.Background(), c, []string{"check"}, false, false); err != nil {
		t.Fatalf("unexpected update check json error: %v", err)
	}
	if !strings.Contains(out.String(), `"images":`) {
		t.Fatalf("expected json update report: %s", out.String())
	}

	// Dry run update
	c.Output = "table"
	out.Reset()
	if err := app.update(context.Background(), c, []string{}, true, false); err != nil {
		t.Fatalf("unexpected dry-run error: %v", err)
	}
	if !strings.Contains(out.String(), "DRY-RUN update") {
		t.Fatalf("expected dry run output: %s", out.String())
	}
}
