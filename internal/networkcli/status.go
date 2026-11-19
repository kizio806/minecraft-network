package networkcli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kizio806/minecraft-network/internal/compose"
)

func (a App) status(ctx context.Context, cfg Config) error {
	out, err := a.compose(ctx, cfg, "ps", "--format", "json")
	if err != nil {
		return &ExitError{5, err}
	}
	rows, parseErr := compose.ParseComposeRows(out)
	if parseErr != nil {
		return &ExitError{5, parseErr}
	}
	if cfg.Output == "json" {
		return json.NewEncoder(a.Out).Encode(map[string]any{"schema_version": 1, "services": rows})
	}
	fmt.Fprintln(a.Out, "Minecraft Network")
	for _, row := range rows {
		fmt.Fprintf(a.Out, "%-16s %-10s %s\n", row.Service, row.State, row.Health)
	}
	return nil
}
