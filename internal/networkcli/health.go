package networkcli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kizio806/minecraft-network/internal/compose"
)

func (a App) nativeHealth(ctx context.Context, c Config) error {
	if _, err := a.compose(ctx, c, "config", "--quiet"); err != nil {
		return &ExitError{3, err}
	}
	if err := c.ValidateSecrets(); err != nil {
		return &ExitError{3, err}
	}
	client := compose.NewClient(a.Runner, c.ProjectDir, c.ComposeFile())
	status, err := client.CheckHealth(ctx)
	if err != nil {
		return &ExitError{5, err}
	}
	if c.Output == "json" {
		if err := json.NewEncoder(a.Out).Encode(map[string]any{"schema_version": 1, "services": status.Services, "healthy": status.AllOK}); err != nil {
			return &ExitError{5, err}
		}
	} else {
		for _, row := range status.Services {
			fmt.Fprintf(a.Out, "%-16s %-10s %s\n", row.Service, row.State, row.Health)
		}
	}
	if !status.AllOK {
		return &ExitError{5, fmt.Errorf("one or more services are degraded")}
	}
	return nil
}
