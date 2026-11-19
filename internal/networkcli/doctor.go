package networkcli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

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
