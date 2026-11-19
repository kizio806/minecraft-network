package networkcli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/kizio806/minecraft-network/internal/backup"
	"github.com/kizio806/minecraft-network/internal/compose"
)

func (a App) backup(ctx context.Context, cfg Config, args []string) error {
	if len(args) == 1 && args[0] == "list" {
		matches, _ := filepath.Glob(filepath.Join(cfg.BackupDir, "minecraft-network-*.tar.gz.age"))
		for _, m := range matches {
			fmt.Fprintln(a.Out, m)
		}
		return nil
	}
	return a.nativeBackup(ctx, cfg)
}

func (a App) nativeBackup(ctx context.Context, c Config) error {
	client := compose.NewClient(a.Runner, c.ProjectDir, c.ComposeFile())
	mgr := backup.NewManager(c, client, a.Runner)
	archive, hash, err := mgr.Create(ctx)
	if err != nil {
		return &ExitError{4, err}
	}
	fmt.Fprintf(a.Out, "Backup created: %s\nSHA-256: %s\n", archive, hash)
	return nil
}
