package networkcli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/kizio806/minecraft-network/internal/compose"
)

var isConsoleService = compose.IsConsoleService

func (a App) console(ctx context.Context, cfg Config, args []string) error {
	if len(args) == 0 {
		return &ExitError{2, errors.New("console requires a service name (e.g. lobby, survival, proxy)")}
	}
	if len(args) > 1 {
		return &ExitError{2, errors.New("console accepts only one service name")}
	}
	service := args[0]
	if !isConsoleService(service) {
		return &ExitError{2, fmt.Errorf("console is only supported for game servers: proxy, lobby, survival (got %q)", service)}
	}
	attachArgs := []string{"compose", "-f", cfg.ComposeFile(), "attach", "--detach-keys=ctrl-p,ctrl-q", service}
	if err := a.Runner.RunInteractive(ctx, cfg.ProjectDir, "docker", os.Stdin, a.Out, a.Err, attachArgs...); err != nil {
		return &ExitError{3, err}
	}
	return nil
}
