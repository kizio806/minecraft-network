package networkcli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kizio806/minecraft-network/internal/backup"
	"github.com/kizio806/minecraft-network/internal/compose"
)

func (a App) restore(ctx context.Context, cfg Config, args []string, yes bool) error {
	args, localYes := stripFlag(args, "--yes")
	yes = yes || localYes
	if len(args) != 1 {
		return &ExitError{2, errors.New("restore requires backup path or id")}
	}
	if !yes {
		return &ExitError{2, errors.New("restore requires --yes")}
	}
	path := args[0]
	if !strings.Contains(path, "/") {
		path = filepath.Join(cfg.BackupDir, path)
	}
	return a.nativeRestore(ctx, cfg, path, yes)
}

func (a App) nativeRestore(ctx context.Context, c Config, archive string, yes bool) error {
	if !yes {
		return &ExitError{2, errors.New("restore requires --yes")}
	}
	client := compose.NewClient(a.Runner, c.ProjectDir, c.ComposeFile())
	mgr := backup.NewManager(c, client, a.Runner)
	if err := mgr.Restore(ctx, archive); err != nil {
		return &ExitError{ExitRestoreError, err}
	}
	fmt.Fprintln(a.Out, "Restore completed; run network health before accepting traffic.")
	return nil
}
