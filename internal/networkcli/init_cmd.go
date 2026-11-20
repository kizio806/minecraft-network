package networkcli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kizio806/minecraft-network/internal/fileutil"
	"github.com/kizio806/minecraft-network/internal/tui"
)

func (a App) initCmd(ctx context.Context, c Config, args []string) error {
	cfg := tui.DefaultInitConfig()

	if existing, err := loadNetworkSettings(c.ProjectDir); err == nil {
		if existing.Network.Name != "" {
			cfg.Name = existing.Network.Name
		}
		if existing.Network.ShortName != "" {
			cfg.ShortName = existing.Network.ShortName
		}
		if existing.Network.Domain != "" {
			cfg.Domain = existing.Network.Domain
		}
		if existing.Network.Discord != "" {
			cfg.Discord = existing.Network.Discord
		}
		if existing.Network.Website != "" {
			cfg.Website = existing.Network.Website
		}
		if existing.Network.MaxPlayers > 0 {
			cfg.MaxPlayers = existing.Network.MaxPlayers
		}
		if existing.Locale != "" {
			cfg.Locale = existing.Locale
		}
		if existing.Theme.Preset != "" {
			cfg.ThemePreset = existing.Theme.Preset
		}
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Fprintln(a.Out, "Usage: network init [flags]")
			fmt.Fprintln(a.Out, "")
			fmt.Fprintln(a.Out, "Interactive onboarding wizard to configure branding, theme, locale and limits.")
			fmt.Fprintln(a.Out, "")
			fmt.Fprintln(a.Out, "Flags:")
			fmt.Fprintln(a.Out, "  --non-interactive, -y    Run in headless automated mode without prompting")
			fmt.Fprintln(a.Out, "  --name STRING            Network display name (e.g. \"Aetheria Network\")")
			fmt.Fprintln(a.Out, "  --domain STRING          Domain name (e.g. \"play.example.com\")")
			fmt.Fprintln(a.Out, "  --locale pl|en           Primary language for in-game messages")
			fmt.Fprintln(a.Out, "  --theme PRESET           Theme preset (e.g. classic-emerald, amethyst-realm, cyberpunk-neon)")
			fmt.Fprintln(a.Out, "  --max-players INT        Global maximum player capacity")
			return nil
		case "--non-interactive", "-y", "--yes":
			cfg.NonInteractive = true
		case "--locale":
			if i+1 < len(args) {
				i++
				cfg.Locale = args[i]
			}
		case "--theme":
			if i+1 < len(args) {
				i++
				cfg.ThemePreset = args[i]
			}
		case "--name":
			if i+1 < len(args) {
				i++
				cfg.Name = args[i]
			}
		case "--domain":
			if i+1 < len(args) {
				i++
				cfg.Domain = args[i]
			}
		case "--max-players":
			if i+1 < len(args) {
				i++
				if v, err := strconv.Atoi(args[i]); err == nil && v > 0 {
					cfg.MaxPlayers = v
				}
			}
		}
	}

	envPath := filepath.Join(c.ProjectDir, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		exampleEnv := filepath.Join(c.ProjectDir, ".env.example")
		if _, err := os.Stat(exampleEnv); err == nil {
			if err := fileutil.CopyFile(exampleEnv, envPath, 0600); err != nil {
				return &ExitError{ExitLockError, fmt.Errorf("copy .env.example: %w", err)}
			}
			fmt.Fprintln(a.Out, "Created .env from .env.example")
		}
	}

	if err := tui.RunWizard(&cfg); err != nil {
		return &ExitError{ExitUsageError, fmt.Errorf("wizard error: %w", err)}
	}

	if err := tui.WriteNetworkConfig(c.ProjectDir, cfg); err != nil {
		return &ExitError{ExitLockError, fmt.Errorf("write configs/network.yml: %w", err)}
	}

	fmt.Fprintln(a.Out, "Configuration saved to configs/network.yml")

	lock, e := AcquireLock(c.ProjectDir)
	if e != nil {
		return &ExitError{ExitLockError, e}
	}
	defer lock.Close()

	if err := a.nativeInstall(ctx, c); err != nil {
		return err
	}

	fmt.Fprintln(a.Out, tui.SuccessStyle.Render("✦ Network successfully initialized and ready to start!"))
	fmt.Fprintln(a.Out, "Start the network with: ./bin/network start")
	return nil
}
