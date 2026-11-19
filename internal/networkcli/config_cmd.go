package networkcli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

func (a App) config(cfg Config, args []string) error {
	if len(args) != 1 || (args[0] != "show" && args[0] != "validate") {
		return &ExitError{2, errors.New("config requires show or validate")}
	}
	if args[0] == "validate" {
		if err := cfg.ValidateSecrets(); err != nil {
			return &ExitError{3, err}
		}
		if err := cfg.ValidateEnvironmentFile(); err != nil {
			return &ExitError{3, err}
		}
		if a.Runner == nil {
			a.Runner = ExecRunner{}
		}
		if _, err := a.compose(context.Background(), cfg, "config", "--quiet"); err != nil {
			return &ExitError{3, err}
		}
		fmt.Fprintln(a.Out, "configuration valid")
		return nil
	}
	return json.NewEncoder(a.Out).Encode(cfg)
}
