package networkcli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctor(t *testing.T) {
	c := operationalConfig(t)

	// Create required manifest and lockfile mocks
	if err := os.MkdirAll(filepath.Join(c.ProjectDir, "configs/plugins"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.PluginManagerManifest(), []byte("schema-version: 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.PluginManagerLock(), []byte("schema-version: 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.ProjectDir, ".env.example"), []byte("VELOCITY_VERSION=4.3.1\n"), 0600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	runner := &fakeRunner{
		output: map[string]string{
			"docker version --format {{.Server.Version}}": "26.1.4",
			"docker compose -f " + c.ComposeFile() + " version": "2.27.1",
		},
	}
	app := App{Out: &out, Runner: runner}

	// Successful doctor check
	if err := app.doctor(context.Background(), c); err != nil {
		t.Fatalf("unexpected doctor failure: %v", err)
	}
	if !strings.Contains(out.String(), "docker           OK") || !strings.Contains(out.String(), "secrets          OK") {
		t.Fatalf("expected doctor table output, got: %s", out.String())
	}

	// JSON output
	out.Reset()
	c.Output = "json"
	if err := app.doctor(context.Background(), c); err != nil {
		t.Fatalf("unexpected doctor json failure: %v", err)
	}
	if !strings.Contains(out.String(), `"Status":"OK"`) {
		t.Fatalf("expected doctor json output, got: %s", out.String())
	}

	// Failed docker check
	c.Output = "table"
	runner.failContains = "version"
	if err := app.doctor(context.Background(), c); err == nil {
		t.Fatal("expected doctor failure when docker version fails")
	}

	// Failed secrets check
	runner.failContains = ""
	_ = os.Remove(filepath.Join(c.ProjectDir, "secrets/postgres_password"))
	if err := app.doctor(context.Background(), c); err == nil {
		t.Fatal("expected doctor failure when secrets are missing")
	}
}
