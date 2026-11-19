package networkcli

import (
	"context"
	"fmt"
	"strings"

	"github.com/kizio806/minecraft-network/internal/compose"
)

var (
	isMonitoringService = compose.IsMonitoringService
	validService        = compose.ValidService
)

func (a App) lifecycle(ctx context.Context, cfg Config, action string, services []string, withMonitoring ...bool) error {
	monitoring := len(withMonitoring) > 0 && withMonitoring[0]
	var filtered []string
	for _, s := range services {
		if s == "--monitoring" {
			monitoring = true
		} else {
			filtered = append(filtered, s)
		}
	}
	services = filtered

	var profile string
	if monitoring {
		profile = "monitoring"
		if len(services) == 0 {
			services = []string{"proxy", "auth", "lobby", "survival", "node-exporter", "prometheus", "grafana", "uptime-kuma", "minecraft-exporter"}
		}
	} else {
		for _, s := range services {
			if isMonitoringService(s) {
				profile = "monitoring"
				break
			}
		}
		if len(services) == 0 {
			services = []string{"proxy", "auth", "lobby", "survival"}
		}
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
	if _, err := a.composeWithProfile(ctx, cfg, profile, "config", "--quiet"); err != nil {
		return &ExitError{3, err}
	}
	args := []string{action}
	if action == "start" {
		args = []string{"up", "-d"}
	}
	args = append(args, services...)
	_, err := a.composeWithProfile(ctx, cfg, profile, args...)
	if err != nil {
		return &ExitError{5, err}
	}
	fmt.Fprintf(a.Out, "%s completed: %s\n", action, strings.Join(services, ", "))
	return nil
}
