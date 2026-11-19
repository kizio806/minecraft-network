package networkcli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecRunner(t *testing.T) {
	bin := t.TempDir()
	docker := filepath.Join(bin, "docker")
	script := `#!/bin/sh
if [ "$1" = "input" ]; then cat; exit 0; fi
if [ "$1" = "interactive" ]; then cat; exit 0; fi
if [ "$1" = "stream" ]; then echo "stream-stdout"; echo "stream-stderr" >&2; exit 0; fi
if [ "$1" = "fail" ]; then echo "failing now" >&2; exit 1; fi
echo hello world
`
	// #nosec G306 -- mock test executable script
	if err := os.WriteFile(docker, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+oldPath)

	runner := ExecRunner{}
	ctx := context.Background()

	// Successful Run
	out, err := runner.Run(ctx, "", "docker", "version")
	if err != nil || strings.TrimSpace(out) != "hello world" {
		t.Fatalf("unexpected Run result: %s %v", out, err)
	}

	// Command not allowed across all runner methods
	if _, err := runner.Run(ctx, "", "echo"); err == nil {
		t.Fatal("expected error on unallowed command in Run")
	}
	if _, err := runner.RunInput(ctx, "", "echo", nil); err == nil {
		t.Fatal("expected error on unallowed command in RunInput")
	}
	if err := runner.RunInteractive(ctx, "", "echo", nil, nil, nil); err == nil {
		t.Fatal("expected error on unallowed command in RunInteractive")
	}
	if err := runner.RunStream(ctx, "", nil, nil, "echo"); err == nil {
		t.Fatal("expected error on unallowed command in RunStream")
	}

	// Command failure in Run
	if _, err := runner.Run(ctx, "", "docker", "fail"); err == nil {
		t.Fatal("expected error on failing command in Run")
	}

	// Command failure in RunInput
	if _, err := runner.RunInput(ctx, "", "docker", strings.NewReader(""), "fail"); err == nil {
		t.Fatal("expected error on failing command in RunInput")
	}

	// RunInput
	input := strings.NewReader("input-data\n")
	out, err = runner.RunInput(ctx, "", "docker", input, "input")
	if err != nil || strings.TrimSpace(out) != "input-data" {
		t.Fatalf("unexpected RunInput result: %s %v", out, err)
	}

	// RunInteractive
	var stdIn bytes.Buffer
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer
	stdIn.WriteString("interactive-test\n")
	err = runner.RunInteractive(ctx, "", "docker", &stdIn, &stdOut, &stdErr, "interactive")
	if err != nil || strings.TrimSpace(stdOut.String()) != "interactive-test" {
		t.Fatalf("unexpected RunInteractive result: %s %v", stdOut.String(), err)
	}

	// RunStream
	var streamOut bytes.Buffer
	var streamErr bytes.Buffer
	err = runner.RunStream(ctx, "", &streamOut, &streamErr, "docker", "stream")
	if err != nil || strings.TrimSpace(streamOut.String()) != "stream-stdout" || strings.TrimSpace(streamErr.String()) != "stream-stderr" {
		t.Fatalf("unexpected RunStream result: out=%s err=%s, error=%v", streamOut.String(), streamErr.String(), err)
	}

	// OperationContext
	opCtx, cancel := operationContext(ctx, 10*time.Second)
	defer cancel()
	deadline, ok := opCtx.Deadline()
	if !ok || deadline.Before(time.Now()) {
		t.Fatal("expected valid context deadline")
	}

	// Default timeout when timeout <= 0
	defCtx, defCancel := operationContext(ctx, 0)
	defer defCancel()
	defDeadline, ok := defCtx.Deadline()
	if !ok || defDeadline.Before(deadline) {
		t.Fatal("expected default 30 minute timeout")
	}
}
