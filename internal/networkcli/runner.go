package networkcli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type Runner interface {
	Run(context.Context, string, string, ...string) (string, error)
	RunInput(context.Context, string, string, io.Reader, ...string) (string, error)
}

type ExecRunner struct{}

func (ExecRunner) RunInput(ctx context.Context, dir, name string, input io.Reader, args ...string) (string, error) {
	allowed := map[string]bool{"docker": true}
	if !allowed[name] {
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

func (ExecRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, error) {
	allowed := map[string]bool{"docker": true}
	if !allowed[name] {
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

func operationContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	return context.WithTimeout(parent, timeout)
}
