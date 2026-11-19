package compose_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/kizio806/minecraft-network/internal/compose"
)

type mockRunner struct {
	lastCmd  string
	lastArgs []string
	response string
	err      error
}

func (m *mockRunner) Run(_ context.Context, _, name string, args ...string) (string, error) {
	m.lastCmd = name
	m.lastArgs = args
	return m.response, m.err
}

func (m *mockRunner) RunInput(_ context.Context, _, name string, _ io.Reader, args ...string) (string, error) {
	m.lastCmd = name
	m.lastArgs = args
	return m.response, m.err
}

func (m *mockRunner) RunInteractive(_ context.Context, _, name string, _ io.Reader, _, _ io.Writer, args ...string) error {
	m.lastCmd = name
	m.lastArgs = args
	return m.err
}

func (m *mockRunner) RunStream(_ context.Context, _ string, _, _ io.Writer, name string, args ...string) error {
	m.lastCmd = name
	m.lastArgs = args
	return m.err
}

func TestComposeClient(t *testing.T) {
	runner := &mockRunner{response: "ok\n"}
	client := compose.NewClient(runner, "/app", "/app/docker-compose.yml")

	out, err := client.Compose(context.Background(), "", "up", "-d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ok\n" {
		t.Fatalf("unexpected output: %s", out)
	}
	if strings.Join(runner.lastArgs, " ") != "compose -f /app/docker-compose.yml up -d" {
		t.Fatalf("unexpected args: %v", runner.lastArgs)
	}
}

func TestParseComposeRows(t *testing.T) {
	raw := `{"Service":"proxy","State":"running","Health":"healthy","Status":"Up","Name":"proxy_1"}
{"Service":"auth","State":"running","Health":"healthy","Status":"Up","Name":"auth_1"}`

	rows, err := compose.ParseComposeRows(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Service != "proxy" || rows[0].Health != "healthy" {
		t.Fatalf("unexpected row data: %+v", rows[0])
	}
}

func TestServiceValidation(t *testing.T) {
	if !compose.ValidService("proxy") || !compose.ValidService("survival") {
		t.Error("expected proxy and survival to be valid services")
	}
	if compose.ValidService("unknown-svc") {
		t.Error("expected unknown-svc to be invalid")
	}
	if !compose.IsConsoleService("proxy") || compose.IsConsoleService("postgres") {
		t.Error("console service check failed")
	}
}
