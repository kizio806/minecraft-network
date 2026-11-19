package compose_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

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

func (m *mockRunner) RunStream(_ context.Context, _ string, out, errOut io.Writer, name string, args ...string) error {
	m.lastCmd = name
	m.lastArgs = args
	if out != nil && m.response != "" {
		_, _ = out.Write([]byte(m.response))
	}
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

	// Test with profile
	out, err = client.Compose(context.Background(), "monitoring", "ps")
	if err != nil {
		t.Fatalf("unexpected error with profile: %v", err)
	}
	if strings.Join(runner.lastArgs, " ") != "compose -f /app/docker-compose.yml --profile monitoring ps" {
		t.Fatalf("unexpected profile args: %v", runner.lastArgs)
	}

	// Test ComposeStream
	var streamOut, streamErr bytes.Buffer
	err = client.ComposeStream(context.Background(), &streamOut, &streamErr, "monitoring", "logs")
	if err != nil {
		t.Fatalf("unexpected error in ComposeStream: %v", err)
	}
	if streamOut.String() != "ok\n" {
		t.Fatalf("unexpected stream output: %s", streamOut.String())
	}

	// Test NewClient default runner
	defaultClient := compose.NewClient(nil, "/app", "/app/docker-compose.yml")
	if defaultClient.Runner == nil {
		t.Fatal("expected default ExecRunner when nil runner provided")
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

	// Malformed JSON line
	_, err = compose.ParseComposeRows("not-json")
	if err == nil || !strings.Contains(err.Error(), "invalid Compose status JSON") {
		t.Fatalf("expected invalid JSON error, got: %v", err)
	}
}

func TestCheckHealth(t *testing.T) {
	runner := &mockRunner{}
	client := compose.NewClient(runner, "/app", "/app/docker-compose.yml")

	// 1. All managed services running and healthy
	runner.response = `{"Service":"proxy","State":"running","Health":"healthy","Status":"Up"}
{"Service":"auth","State":"running","Health":"healthy","Status":"Up"}
{"Service":"lobby","State":"running","Health":"healthy","Status":"Up"}
{"Service":"survival","State":"running","Health":"healthy","Status":"Up"}
`
	status, err := client.CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.AllOK {
		t.Fatal("expected AllOK to be true when all services healthy")
	}
	if len(status.Services) != 4 {
		t.Fatalf("expected 4 services, got %d", len(status.Services))
	}

	// 2. One managed service missing (survival omitted)
	runner.response = `{"Service":"proxy","State":"running","Health":"healthy"}
{"Service":"auth","State":"running","Health":"healthy"}
{"Service":"lobby","State":"running","Health":"healthy"}
`
	status, err = client.CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.AllOK {
		t.Fatal("expected AllOK to be false when survival service is missing")
	}

	// 3. One managed service not running (exited)
	runner.response = `{"Service":"proxy","State":"running","Health":"healthy"}
{"Service":"auth","State":"exited","Health":""}
{"Service":"lobby","State":"running","Health":"healthy"}
{"Service":"survival","State":"running","Health":"healthy"}
`
	status, err = client.CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.AllOK {
		t.Fatal("expected AllOK to be false when auth is exited")
	}

	// 4. One managed service unhealthy
	runner.response = `{"Service":"proxy","State":"running","Health":"unhealthy"}
{"Service":"auth","State":"running","Health":"healthy"}
{"Service":"lobby","State":"running","Health":"healthy"}
{"Service":"survival","State":"running","Health":"healthy"}
`
	status, err = client.CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.AllOK {
		t.Fatal("expected AllOK to be false when proxy is unhealthy")
	}

	// 5. Compose execution error
	runner.err = errors.New("compose daemon unreachable")
	runner.response = ""
	_, err = client.CheckHealth(context.Background())
	if err == nil {
		t.Fatal("expected error on compose failure, got nil")
	}

	// 6. Parse JSON error
	runner.err = nil
	runner.response = "invalid json line"
	_, err = client.CheckHealth(context.Background())
	if err == nil {
		t.Fatal("expected error on invalid json, got nil")
	}
}

func TestServiceValidation(t *testing.T) {
	services := []string{"proxy", "auth", "lobby", "survival", "postgres", "node-exporter", "prometheus", "grafana", "uptime-kuma", "minecraft-exporter"}
	for _, s := range services {
		if !compose.ValidService(s) {
			t.Errorf("expected %s to be a valid service", s)
		}
	}
	if compose.ValidService("unknown-svc") {
		t.Error("expected unknown-svc to be invalid")
	}

	monitoringServices := []string{"node-exporter", "prometheus", "grafana", "uptime-kuma", "minecraft-exporter"}
	for _, s := range monitoringServices {
		if !compose.IsMonitoringService(s) {
			t.Errorf("expected %s to be a monitoring service", s)
		}
	}
	nonMonitoring := []string{"proxy", "auth", "lobby", "survival", "postgres", "random"}
	for _, s := range nonMonitoring {
		if compose.IsMonitoringService(s) {
			t.Errorf("expected %s NOT to be a monitoring service", s)
		}
	}

	consoleServices := []string{"proxy", "lobby", "survival"}
	for _, s := range consoleServices {
		if !compose.IsConsoleService(s) {
			t.Errorf("expected %s to be a console service", s)
		}
	}
	nonConsole := []string{"auth", "postgres", "grafana", "prometheus"}
	for _, s := range nonConsole {
		if compose.IsConsoleService(s) {
			t.Errorf("expected %s NOT to be a console service", s)
		}
	}
}

func TestExecRunner_AllowlistAndExecution(t *testing.T) {
	runner := compose.ExecRunner{}
	ctx := context.Background()

	// 1. Rejected binaries
	for _, badBin := range []string{"rm", "bash", "sh", "curl", "python"} {
		if _, err := runner.Run(ctx, "", badBin, "-v"); err == nil || !strings.Contains(err.Error(), "not allowed") {
			t.Errorf("expected Run to reject binary %q, got: %v", badBin, err)
		}
		if _, err := runner.RunInput(ctx, "", badBin, strings.NewReader(""), "-v"); err == nil || !strings.Contains(err.Error(), "not allowed") {
			t.Errorf("expected RunInput to reject binary %q, got: %v", badBin, err)
		}
		if err := runner.RunInteractive(ctx, "", badBin, strings.NewReader(""), nil, nil, "-v"); err == nil || !strings.Contains(err.Error(), "not allowed") {
			t.Errorf("expected RunInteractive to reject binary %q, got: %v", badBin, err)
		}
		if err := runner.RunStream(ctx, "", nil, nil, badBin, "-v"); err == nil || !strings.Contains(err.Error(), "not allowed") {
			t.Errorf("expected RunStream to reject binary %q, got: %v", badBin, err)
		}
	}

	// 2. Allowed binary execution error handling (e.g. invalid docker subcommand)
	_, err := runner.Run(ctx, "", "docker", "non-existent-subcommand-xyz")
	if err == nil {
		t.Fatal("expected error on invalid docker subcommand, got nil")
	}

	_, err = runner.RunInput(ctx, "", "docker", strings.NewReader("input"), "non-existent-subcommand-xyz")
	if err == nil {
		t.Fatal("expected error on invalid docker subcommand with input, got nil")
	}

	var streamOut, streamErr bytes.Buffer
	err = runner.RunStream(ctx, "", &streamOut, &streamErr, "docker", "non-existent-subcommand-xyz")
	if err == nil {
		t.Fatal("expected error on invalid docker subcommand stream, got nil")
	}

	err = runner.RunInteractive(ctx, "", "docker", strings.NewReader(""), &streamOut, &streamErr, "non-existent-subcommand-xyz")
	if err == nil {
		t.Fatal("expected error on invalid docker subcommand interactive, got nil")
	}

	// 3. Context cancellation handling
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = runner.Run(canceledCtx, "", "docker", "version")
	if err == nil {
		t.Fatal("expected error with canceled context, got nil")
	}

	timedOutCtx, timeoutCancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer timeoutCancel()
	time.Sleep(1 * time.Millisecond)
	_, err = runner.Run(timedOutCtx, "", "docker", "version")
	if err == nil {
		t.Fatal("expected error with timed out context, got nil")
	}
}
