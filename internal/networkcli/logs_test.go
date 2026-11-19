package networkcli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLogs(t *testing.T) {
	c := operationalConfig(t)
	var out, errOut bytes.Buffer
	runner := &fakeRunner{
		output: map[string]string{
			"docker compose -f " + c.ComposeFile() + " logs --tail 50 --timestamps --follow proxy": "2026-01-01 [INFO] Proxy log line\n",
		},
	}
	app := App{Out: &out, Err: &errOut, Runner: runner}

	// Unsupported service
	if err := app.logs(context.Background(), c, []string{"unsupported_svc"}, false, 100); err == nil {
		t.Fatal("expected error on unsupported service")
	}

	// Valid log execution
	if err := app.logs(context.Background(), c, []string{"proxy"}, true, 50); err != nil {
		t.Fatalf("unexpected logs error: %v", err)
	}
	if !strings.Contains(out.String(), "Proxy log line") {
		t.Fatalf("expected log output, got: %s", out.String())
	}

	// Compose failure
	runner.failContains = "logs"
	if err := app.logs(context.Background(), c, []string{"proxy"}, false, 100); err == nil {
		t.Fatal("expected error on compose failure")
	}
}
