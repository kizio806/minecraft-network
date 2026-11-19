package networkcli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kizio806/minecraft-network/internal/pluginmanager"
)

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

func (a App) nativeUpdate(ctx context.Context, c Config, dry, yes bool) error {
	if !yes && !dry {
		return &ExitError{2, errors.New("update requires --yes or --dry-run")}
	}
	if dry {
		if _, e := a.compose(ctx, c, "config", "--quiet"); e != nil {
			return &ExitError{3, e}
		}
		fmt.Fprintln(a.Out, "DRY-RUN update: pinned Compose images would be pulled and services recreated")
		return nil
	}
	if e := a.nativeBackup(ctx, c); e != nil {
		return e
	}
	if _, e := a.compose(ctx, c, "pull"); e != nil {
		return &ExitError{5, e}
	}
	if _, e := a.compose(ctx, c, "up", "-d", "--remove-orphans"); e != nil {
		return &ExitError{5, e}
	}
	return a.nativeHealth(ctx, c)
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
		appr := approved[name]
		report.Images = append(report.Images, imageUpdateStatus{Name: name, Configured: v, Approved: appr, MatchesApproved: v == appr})
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
