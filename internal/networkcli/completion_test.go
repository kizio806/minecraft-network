package networkcli_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kizio806/minecraft-network/internal/networkcli"
)

func TestCompletionCommand(t *testing.T) {
	ctx := context.Background()

	t.Run("help and usage", func(t *testing.T) {
		var out, errOut bytes.Buffer
		app := networkcli.App{Out: &out, Err: &errOut}

		if err := app.Execute(ctx, []string{"completion"}); err != nil {
			t.Fatalf("unexpected error on empty completion: %v", err)
		}
		if !strings.Contains(out.String(), "Usage: network completion [bash|zsh|fish]") {
			t.Fatalf("expected usage in output: %s", out.String())
		}

		out.Reset()
		if err := app.Execute(ctx, []string{"completion", "--help"}); err != nil {
			t.Fatalf("unexpected error on completion --help: %v", err)
		}
		if !strings.Contains(out.String(), "Installation:") {
			t.Fatalf("expected installation guide in output: %s", out.String())
		}
	})

	t.Run("bash completion", func(t *testing.T) {
		var out, errOut bytes.Buffer
		app := networkcli.App{Out: &out, Err: &errOut}

		if err := app.Execute(ctx, []string{"completion", "bash"}); err != nil {
			t.Fatalf("unexpected error generating bash completion: %v", err)
		}
		script := out.String()
		if !strings.Contains(script, "_network_completion()") {
			t.Errorf("expected _network_completion function in script: %s", script)
		}
		if !strings.Contains(script, "console_services=\"proxy lobby survival\"") {
			t.Errorf("expected console services definition: %s", script)
		}
		if !strings.Contains(script, "complete -F _network_completion network") {
			t.Errorf("expected complete registration: %s", script)
		}
	})

	t.Run("zsh completion", func(t *testing.T) {
		var out, errOut bytes.Buffer
		app := networkcli.App{Out: &out, Err: &errOut}

		if err := app.Execute(ctx, []string{"completion", "zsh"}); err != nil {
			t.Fatalf("unexpected error generating zsh completion: %v", err)
		}
		script := out.String()
		if !strings.Contains(script, "#compdef network") {
			t.Errorf("expected #compdef header: %s", script)
		}
		if !strings.Contains(script, "_describe -t commands") {
			t.Errorf("expected command description in zsh script: %s", script)
		}
	})

	t.Run("fish completion", func(t *testing.T) {
		var out, errOut bytes.Buffer
		app := networkcli.App{Out: &out, Err: &errOut}

		if err := app.Execute(ctx, []string{"completion", "fish"}); err != nil {
			t.Fatalf("unexpected error generating fish completion: %v", err)
		}
		script := out.String()
		if !strings.Contains(script, "complete -c network") {
			t.Errorf("expected complete -c network in fish script: %s", script)
		}
		if !strings.Contains(script, "__fish_seen_subcommand_from console") {
			t.Errorf("expected console subcommand rule in fish script: %s", script)
		}
	})

	t.Run("unsupported shell", func(t *testing.T) {
		var out, errOut bytes.Buffer
		app := networkcli.App{Out: &out, Err: &errOut}

		err := app.Execute(ctx, []string{"completion", "powershell"})
		if err == nil {
			t.Fatal("expected error for unsupported shell, got nil")
		}
		var exitErr *networkcli.ExitError
		if !errors.As(err, &exitErr) || exitErr.Code != 2 {
			t.Fatalf("expected ExitError with code 2, got: %v", err)
		}
		if !strings.Contains(exitErr.Error(), "unsupported shell \"powershell\"") {
			t.Errorf("unexpected error message: %v", exitErr)
		}
	})
}
