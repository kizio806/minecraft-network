package networkcli

import (
	"context"
	"fmt"
	"io"
	"os"
)

var (
	Version   = "dev"
	GitCommit = "none"
	BuildDate = "unknown"
)

type App struct {
	Out, Err io.Writer
	Runner   Runner
}

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }

func (a App) Execute(ctx context.Context, args []string) error {
	if a.Out == nil {
		a.Out = os.Stdout
	}
	if a.Err == nil {
		a.Err = os.Stderr
	}
	if a.Runner == nil {
		a.Runner = ExecRunner{}
	}
	project, output, yes, dry, follow, monitoring, lines, rest, err := parseGlobal(args)
	if err != nil {
		return &ExitError{2, err}
	}
	if len(rest) == 0 || rest[0] == "help" || rest[0] == "--help" {
		a.help()
		return nil
	}
	if rest[0] == "--version" || rest[0] == "version" {
		fmt.Fprintln(a.Out, Version)
		return nil
	}
	cfg, err := Load(project, output)
	if err != nil {
		return &ExitError{3, err}
	}
	cmd := rest[0]
	rest = rest[1:]
	switch cmd {
	case "init":
		return a.initCmd(ctx, cfg, rest)
	case "doctor":
		return a.doctor(ctx, cfg)
	case "config":
		return a.config(cfg, rest)
	case "start", "stop", "restart":
		return a.lifecycle(ctx, cfg, cmd, rest, monitoring)
	case "console":
		return a.console(ctx, cfg, rest)
	case "status":
		return a.status(ctx, cfg)
	case "logs":
		return a.logs(ctx, cfg, rest, follow, lines)
	case "health":
		return a.nativeHealth(ctx, cfg)
	case "install":
		lock, e := AcquireLock(cfg.ProjectDir)
		if e != nil {
			return &ExitError{4, e}
		}
		defer lock.Close()
		return a.nativeInstall(ctx, cfg)
	case "backup":
		return a.backup(ctx, cfg, rest)
	case "restore":
		return a.restore(ctx, cfg, rest, yes)
	case "update":
		return a.update(ctx, cfg, rest, dry, yes)
	case "plugins":
		return a.plugins(ctx, cfg, rest, dry)
	default:
		return &ExitError{2, fmt.Errorf("unknown command %q", cmd)}
	}
}

func (a App) help() {
	fmt.Fprintln(a.Out, `network [global flags] <command>
Commands: init [--non-interactive], doctor, config validate|show, install, start|stop|restart [--monitoring] [service], console <service>, status, logs [service], health, backup [list], restore <id> --yes, update check|--dry-run|--apply, plugins <command>
Global flags: --project-dir PATH --output table|json --yes --dry-run --follow --monitoring --lines N`)
}

func (a App) compose(ctx context.Context, cfg Config, args ...string) (string, error) {
	return a.composeWithProfile(ctx, cfg, "", args...)
}

func (a App) composeWithProfile(ctx context.Context, cfg Config, profile string, args ...string) (string, error) {
	var base []string
	if profile != "" {
		base = []string{"compose", "-f", cfg.ComposeFile(), "--profile", profile}
	} else {
		base = []string{"compose", "-f", cfg.ComposeFile()}
	}
	return a.Runner.Run(ctx, cfg.ProjectDir, "docker", append(base, args...)...)
}
