package networkcli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigCommand(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	runner := &fakeRunner{}
	app := App{Out: &out, Runner: runner}

	// Invalid args
	if err := app.config(c, []string{}); err == nil {
		t.Fatal("expected error on empty args")
	}
	if err := app.config(c, []string{"invalid"}); err == nil {
		t.Fatal("expected error on invalid arg")
	}

	// Show config
	if err := app.config(c, []string{"show"}); err != nil {
		t.Fatalf("unexpected show error: %v", err)
	}
	if !strings.Contains(out.String(), `"ProjectDir"`) {
		t.Fatalf("expected json config output, got: %s", out.String())
	}

	// Validate config success
	out.Reset()
	if err := app.config(c, []string{"validate"}); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
	if !strings.Contains(out.String(), "configuration valid") {
		t.Fatalf("expected configuration valid message, got: %s", out.String())
	}

	// Validate config failure (bad secrets)
	_ = os.Remove(filepath.Join(c.ProjectDir, "secrets/postgres_password"))
	if err := app.config(c, []string{"validate"}); err == nil {
		t.Fatal("expected validate error with missing secret")
	}
}
