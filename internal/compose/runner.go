package compose

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Runner abstracts command execution for testability and isolation.
type Runner interface {
	Run(ctx context.Context, dir, name string, args ...string) (string, error)
	RunInput(ctx context.Context, dir, name string, input io.Reader, args ...string) (string, error)
	RunInteractive(ctx context.Context, dir, name string, in io.Reader, out, errOut io.Writer, args ...string) error
	RunStream(ctx context.Context, dir string, out, errOut io.Writer, name string, args ...string) error
}

// ExecRunner implements Runner using os/exec with an allowlist of permitted binaries.
type ExecRunner struct{}

var allowedBinaries = map[string]bool{
	"docker": true,
}

// RunInteractive executes a command with interactive standard streams.
func (ExecRunner) RunInteractive(ctx context.Context, dir, name string, in io.Reader, out, errOut io.Writer, args ...string) error {
	if !allowedBinaries[name] {
		return fmt.Errorf("command %q is not allowed", name)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}

// RunInput executes a command with piped input and returns captured combined output.
func (ExecRunner) RunInput(ctx context.Context, dir, name string, input io.Reader, args ...string) (string, error) {
	if !allowedBinaries[name] {
		return "", fmt.Errorf("command %q is not allowed", name)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = input
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	if err != nil {
		return out.String(), fmt.Errorf("%s %v: %w: %s", name, args, err, strings.TrimSpace(out.String()))
	}
	return out.String(), nil
}

// Run executes a command and returns captured combined output.
func (ExecRunner) Run(ctx context.Context, dir, name string, args ...string) (string, error) {
	if !allowedBinaries[name] {
		return "", fmt.Errorf("command %q is not allowed", name)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	if err != nil {
		return out.String(), fmt.Errorf("%s %v: %w: %s", name, args, err, strings.TrimSpace(out.String()))
	}
	return out.String(), nil
}

// RunStream executes a command directing stdout and stderr to specific streams.
func (ExecRunner) RunStream(ctx context.Context, dir string, out, errOut io.Writer, name string, args ...string) error {
	if !allowedBinaries[name] {
		return fmt.Errorf("command %q is not allowed", name)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}
