package networkcli

import (
	"context"
	"fmt"
	"strconv"
)

func (a App) logs(ctx context.Context, cfg Config, services []string, follow bool, lines int) error {
	for _, s := range services {
		if !validService(s) {
			return &ExitError{2, fmt.Errorf("unsupported service %q", s)}
		}
	}
	args := []string{"logs", "--tail", strconv.Itoa(lines), "--timestamps"}
	if follow {
		args = append(args, "--follow")
	}
	args = append(args, services...)
	out, err := a.compose(ctx, cfg, args...)
	if out != "" {
		fmt.Fprint(a.Out, out)
	}
	if err != nil {
		return &ExitError{5, err}
	}
	return nil
}
