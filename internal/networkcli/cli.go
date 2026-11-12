package networkcli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kizio806/minecraft-network/internal/pluginmanager"
)

var Version = "dev"

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
	project, output, yes, dry, follow, lines, rest, err := parseGlobal(args)
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
	case "doctor":
		return a.doctor(ctx, cfg)
	case "config":
		return a.config(cfg, rest)
	case "start", "stop", "restart":
		return a.lifecycle(ctx, cfg, cmd, rest)
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

func parseGlobal(args []string) (project, output string, yes, dry, follow bool, lines int, rest []string, err error) {
	lines = 100
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project-dir":
			if i+1 >= len(args) {
				return "", "", false, false, false, 0, nil, errors.New("--project-dir requires a value")
			}
			i++
			project = args[i]
		case "--output":
			if i+1 >= len(args) {
				return "", "", false, false, false, 0, nil, errors.New("--output requires a value")
			}
			i++
			output = args[i]
		case "--yes":
			yes = true
		case "--dry-run":
			dry = true
		case "--follow":
			follow = true
		case "--lines":
			if i+1 >= len(args) {
				return "", "", false, false, false, 0, nil, errors.New("--lines requires a value")
			}
			i++
			lines, err = strconv.Atoi(args[i])
			if err != nil || lines < 0 {
				return "", "", false, false, false, 0, nil, errors.New("--lines must be a non-negative integer")
			}
		case "--version":
			rest = append(rest, args[i:]...)
			return
		default:
			rest = append(rest, args[i:]...)
			return
		}
	}
	return
}

func (a App) help() {
	fmt.Fprintln(a.Out, `network [global flags] <command>
Commands: doctor, config validate|show, install, start|stop|restart [service], status, logs [service], health, backup [list], restore <id> --yes, update check|--dry-run|--apply, plugins <command>
Global flags: --project-dir PATH --output table|json --yes --dry-run --follow --lines N`)
}

func (a App) compose(ctx context.Context, cfg Config, args ...string) (string, error) {
	return a.Runner.Run(ctx, cfg.ProjectDir, "docker", append([]string{"compose", "-f", cfg.ComposeFile()}, args...)...)
}
func validService(s string) bool {
	switch s {
	case "proxy", "auth", "lobby", "survival", "postgres", "redis", "node-exporter", "cadvisor", "prometheus", "grafana", "loki", "alloy", "uptime-kuma", "minecraft-exporter":
		return true
	}
	return false
}
func (a App) lifecycle(ctx context.Context, cfg Config, action string, services []string) error {
	if len(services) == 0 {
		services = []string{"proxy", "auth", "lobby", "survival"}
	}
	for _, s := range services {
		if !validService(s) {
			return &ExitError{2, fmt.Errorf("unsupported service %q", s)}
		}
	}
	if err := cfg.ValidateSecrets(); err != nil {
		return &ExitError{3, err}
	}
	if err := cfg.ValidateEnvironmentFile(); err != nil {
		return &ExitError{3, err}
	}
	if action == "start" || action == "restart" {
		if err := prepareRuntimeConfigs(cfg); err != nil {
			return &ExitError{4, err}
		}
	}
	if _, err := a.compose(ctx, cfg, "config", "--quiet"); err != nil {
		return &ExitError{3, err}
	}
	args := []string{action}
	if action == "start" {
		args = []string{"up", "-d"}
	}
	args = append(args, services...)
	_, err := a.compose(ctx, cfg, args...)
	if err != nil {
		return &ExitError{5, err}
	}
	fmt.Fprintf(a.Out, "%s completed: %s\n", action, strings.Join(services, ", "))
	return nil
}

func (a App) status(ctx context.Context, cfg Config) error {
	out, err := a.compose(ctx, cfg, "ps", "--format", "json")
	if err != nil {
		return &ExitError{5, err}
	}
	rows, parseErr := parseComposeRows(out)
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
func (a App) doctor(ctx context.Context, cfg Config) error {
	type check struct{ Name, Status, Detail string }
	checks := []check{}
	add := func(n string, ok bool, d string) {
		s := "OK"
		if !ok {
			s = "FAIL"
		}
		checks = append(checks, check{n, s, d})
	}
	_, err := a.Runner.Run(ctx, cfg.ProjectDir, "docker", "version", "--format", "{{.Server.Version}}")
	add("docker", err == nil, fmt.Sprint(err))
	_, err = a.compose(ctx, cfg, "version")
	add("compose", err == nil, fmt.Sprint(err))
	_, err = os.Stat(cfg.ComposeFile())
	add("compose-file", err == nil, cfg.ComposeFile())
	for _, p := range []string{cfg.PluginManagerManifest(), cfg.PluginManagerLock(), filepath.Join(cfg.ProjectDir, ".env.example")} {
		_, e := os.Stat(p)
		add(filepath.Base(p), e == nil, p)
	}
	if err := cfg.ValidateSecrets(); err != nil {
		add("secrets", false, err.Error())
	} else {
		add("secrets", true, "permissions and contents valid")
	}
	if err := cfg.ValidateEnvironmentFile(); err != nil {
		add(".env", false, err.Error())
	} else {
		add(".env", true, "permissions valid")
	}
	if cfg.Output == "json" {
		return json.NewEncoder(a.Out).Encode(checks)
	}
	for _, v := range checks {
		fmt.Fprintf(a.Out, "%-16s %-5s %s\n", v.Name, v.Status, v.Detail)
	}
	for _, v := range checks {
		if v.Status == "FAIL" {
			return &ExitError{3, errors.New("doctor found failed checks")}
		}
	}
	return nil
}
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
func (a App) update(ctx context.Context, cfg Config, args []string, dry, yes bool) error {
	args, localDry := stripFlag(args, "--dry-run")
	args, localApply := stripFlag(args, "--apply")
	dry = dry || localDry
	yes = yes || localApply
	if len(args) == 1 && args[0] == "check" {
		return a.updateCheck(ctx, cfg)
	}
	if !dry && !yes {
		return &ExitError{2, errors.New("update --apply requires --yes")}
	}
	return a.nativeUpdate(ctx, cfg, dry, yes)
}

type imageUpdateStatus struct {
	Name            string `json:"name"`
	Configured      string `json:"configured"`
	Approved        string `json:"approved"`
	MatchesApproved bool   `json:"matches_approved"`
}
type updateReport struct {
	Images             []imageUpdateStatus `json:"images"`
	PluginCheck        string              `json:"plugin_check"`
	RegistryInspection string              `json:"registry_inspection"`
}

func (a App) updateCheck(ctx context.Context, cfg Config) error {
	b, err := os.ReadFile(filepath.Join(cfg.ProjectDir, ".env"))
	if err != nil {
		return &ExitError{3, err}
	}
	configured := parseEnvValues(string(b))
	exampleBytes, _ := os.ReadFile(filepath.Join(cfg.ProjectDir, ".env.example"))
	approved := parseEnvValues(string(exampleBytes))
	report := updateReport{RegistryInspection: "not performed: registry inspection is intentionally read-only and not implemented"}
	for _, name := range []string{"VELOCITY_IMAGE", "PURPUR_IMAGE", "PURPUR_LOBBY_IMAGE", "NANOLIMBO_IMAGE", "MC_MONITOR_IMAGE"} {
		v := configured[name]
		a := approved[name]
		report.Images = append(report.Images, imageUpdateStatus{Name: name, Configured: v, Approved: a, MatchesApproved: v == a})
	}
	var plugins bytes.Buffer
	pm := pluginmanager.New(cfg.PluginManagerManifest(), cfg.PluginManagerLock())
	if err := pm.UpdateCheck(ctx, &plugins); err != nil {
		return &ExitError{3, err}
	}
	report.PluginCheck = plugins.String()
	if cfg.Output == "json" {
		return json.NewEncoder(a.Out).Encode(report)
	}
	for _, image := range report.Images {
		fmt.Fprintf(a.Out, "%-20s configured=%s approved-match=%t\n", image.Name, image.Configured, image.MatchesApproved)
	}
	fmt.Fprintf(a.Out, "Plugin check:\n%s", report.PluginCheck)
	fmt.Fprintln(a.Out, report.RegistryInspection)
	return nil
}
func (a App) plugins(ctx context.Context, cfg Config, args []string, dry bool) error {
	args, localDry := stripFlag(args, "--dry-run")
	dry = dry || localDry
	if len(args) == 0 {
		return &ExitError{2, errors.New("plugins requires a command")}
	}
	return a.nativePlugin(ctx, cfg, args, dry)
}

func stripFlag(args []string, flag string) ([]string, bool) {
	found := false
	result := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == flag {
			found = true
		} else {
			result = append(result, arg)
		}
	}
	return result, found
}
