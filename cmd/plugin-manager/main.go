package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/kizio806/minecraft-network/internal/pluginmanager"
)

func main() {
	manifestPath := flag.String("manifest", "configs/plugins/manifest.yml", "plugin manifest path")
	lockPath := flag.String("lock", "configs/plugins/lock.yml", "plugin lockfile path")
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
		os.Exit(2)
	}

	manager := pluginmanager.New(*manifestPath, *lockPath)
	ctx := context.Background()
	var err error
	switch flag.Arg(0) {
	case "validate":
		err = manager.Validate()
	case "list":
		err = manager.List(os.Stdout)
	case "plan":
		err = manager.Plan(os.Stdout)
	case "verify":
		err = manager.VerifyInstalled()
	case "update-check":
		err = manager.UpdateCheck(ctx, os.Stdout)
	case "sync":
		apply := flag.NArg() == 2 && flag.Arg(1) == "--apply"
		if flag.NArg() == 2 && flag.Arg(1) != "--dry-run" && !apply {
			err = fmt.Errorf("sync accepts --dry-run or --apply")
			break
		}
		err = manager.Sync(ctx, os.Stdout, apply)
	case "components":
		apply := flag.NArg() == 2 && flag.Arg(1) == "--apply"
		if flag.NArg() == 2 && flag.Arg(1) != "--dry-run" && !apply {
			err = fmt.Errorf("components accepts --dry-run or --apply")
			break
		}
		err = manager.SyncComponents(ctx, os.Stdout, apply)
	case "rollback":
		if flag.NArg() != 1 {
			err = fmt.Errorf("rollback requires an explicit installed-state backup")
			break
		}
		err = manager.Rollback()
	default:
		usage()
		err = fmt.Errorf("unknown command %q", flag.Arg(0))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "plugin-manager:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("usage: plugin-manager [--manifest path] [--lock path] validate|list|plan|verify|update-check|sync --dry-run|sync --apply|components --dry-run|components --apply|rollback")
}
