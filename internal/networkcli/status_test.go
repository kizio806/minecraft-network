package networkcli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestStatus(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	jsonComposeOut := `{"Service":"proxy","State":"running","Health":"healthy","Status":"Up 2 hours","Name":"proxy_1"}
{"Service":"lobby","State":"running","Health":"healthy","Status":"Up 2 hours","Name":"lobby_1"}
`
	runner := &fakeRunner{
		output: map[string]string{
			"docker compose -f " + c.ComposeFile() + " ps --format json": jsonComposeOut,
		},
	}
	app := App{Out: &out, Runner: runner}

	// Table output
	if err := app.status(context.Background(), c); err != nil {
		t.Fatalf("unexpected status error: %v", err)
	}
	if !strings.Contains(out.String(), "Minecraft Network") || !strings.Contains(out.String(), "proxy") {
		t.Fatalf("unexpected table output: %s", out.String())
	}

	// JSON output
	out.Reset()
	c.Output = "json"
	if err := app.status(context.Background(), c); err != nil {
		t.Fatalf("unexpected json status error: %v", err)
	}
	if !strings.Contains(out.String(), `"schema_version":1`) || !strings.Contains(out.String(), `"Service":"proxy"`) {
		t.Fatalf("unexpected json output: %s", out.String())
	}

	// Compose error
	runner.failContains = "ps"
	if err := app.status(context.Background(), c); err == nil {
		t.Fatal("expected status error on compose failure")
	}

	// Invalid JSON output
	runner.failContains = ""
	runner.output["docker compose -f "+c.ComposeFile()+" ps --format json"] = "not valid json"
	if err := app.status(context.Background(), c); err == nil {
		t.Fatal("expected parse error on invalid json")
	}
}
