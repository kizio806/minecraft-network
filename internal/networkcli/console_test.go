package networkcli

import (
	"bytes"
	"context"
	"testing"
)

func TestConsole(t *testing.T) {
	c := operationalConfig(t)
	var out, errOut bytes.Buffer
	runner := &fakeRunner{}
	app := App{Out: &out, Err: &errOut, Runner: runner}

	// No service specified
	if err := app.console(context.Background(), c, []string{}); err == nil {
		t.Fatal("expected error on empty args")
	}

	// Multiple services
	if err := app.console(context.Background(), c, []string{"lobby", "proxy"}); err == nil {
		t.Fatal("expected error on multiple args")
	}

	// Unsupported service
	if err := app.console(context.Background(), c, []string{"postgres"}); err == nil {
		t.Fatal("expected error on unsupported service")
	}

	// Valid service
	if err := app.console(context.Background(), c, []string{"lobby"}); err != nil {
		t.Fatalf("unexpected console error: %v", err)
	}

	// Runner error
	runner.failContains = "attach"
	if err := app.console(context.Background(), c, []string{"proxy"}); err == nil {
		t.Fatal("expected runner error")
	}
}

func TestIsConsoleService(t *testing.T) {
	if !isConsoleService("proxy") || !isConsoleService("lobby") || !isConsoleService("survival") {
		t.Fatal("expected proxy, lobby, survival to be console services")
	}
	if isConsoleService("postgres") || isConsoleService("unknown") {
		t.Fatal("expected postgres and unknown to not be console services")
	}
}
