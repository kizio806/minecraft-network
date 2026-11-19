package networkcli

import (
	"context"
	"errors"
	"fmt"

	"github.com/kizio806/minecraft-network/internal/pluginmanager"
)

func (a App) plugins(ctx context.Context, cfg Config, args []string, dry bool) error {
	args, localDry := stripFlag(args, "--dry-run")
	dry = dry || localDry
	if len(args) == 0 {
		return &ExitError{2, errors.New("plugins requires a command")}
	}
	return a.nativePlugin(ctx, cfg, args, dry)
}

func (a App) nativePlugin(ctx context.Context, c Config, args []string, dry bool) error {
	if len(args) == 0 {
		return &ExitError{2, errors.New("plugins requires a command")}
	}
	pm := pluginmanager.New(c.PluginManagerManifest(), c.PluginManagerLock())
	switch args[0] {
	case "validate":
		return pm.Validate()
	case "list":
		return pm.List(a.Out)
	case "plan":
		return pm.Plan(a.Out)
	case "verify":
		return pm.VerifyInstalled()
	case "update-check":
		return pm.UpdateCheck(ctx, a.Out)
	case "sync":
		lock, e := AcquireLock(c.ProjectDir)
		if e != nil {
			return &ExitError{4, e}
		}
		defer lock.Close()
		apply := len(args) > 1 && args[1] == "--apply"
		return pm.Sync(ctx, a.Out, apply && !dry)
	case "rollback":
		lock, e := AcquireLock(c.ProjectDir)
		if e != nil {
			return &ExitError{4, e}
		}
		defer lock.Close()
		return pm.Rollback()
	case "components":
		lock, e := AcquireLock(c.ProjectDir)
		if e != nil {
			return &ExitError{4, e}
		}
		defer lock.Close()
		apply := len(args) > 1 && args[1] == "--apply"
		return pm.SyncComponents(ctx, a.Out, apply && !dry)
	default:
		return &ExitError{2, fmt.Errorf("unknown plugin command %q", args[0])}
	}
}
