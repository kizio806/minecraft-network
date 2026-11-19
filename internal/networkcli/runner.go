package networkcli

import (
	"context"
	"time"

	"github.com/kizio806/minecraft-network/internal/compose"
)

// Runner abstracts external process execution.
type Runner = compose.Runner

// ExecRunner executes processes using os/exec.
type ExecRunner = compose.ExecRunner

func operationContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	return context.WithTimeout(ctx, timeout)
}
