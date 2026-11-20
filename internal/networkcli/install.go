package networkcli

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kizio806/minecraft-network/internal/fileutil"
	"github.com/kizio806/minecraft-network/internal/i18n"
	"github.com/kizio806/minecraft-network/internal/pluginmanager"
	"gopkg.in/yaml.v3"
)

func (a App) nativeInstall(ctx context.Context, c Config) error {
	for _, d := range []string{"secrets", "servers/proxy", "servers/lobby", "servers/auth", "servers/survival"} {
		if err := os.MkdirAll(filepath.Join(c.ProjectDir, d), 0750); err != nil {
			return &ExitError{4, err}
		}
	}
	for _, name := range []string{"postgres_password", "forwarding_secret", "grafana_admin_password", "luckperms_password", "veloauth_password", "ultimateteams_password", "libertybans_password"} {
		p := filepath.Join(c.ProjectDir, "secrets", name)
		if st, statErr := os.Stat(p); errors.Is(statErr, os.ErrNotExist) || (statErr == nil && st.Size() == 0) {
			flags := os.O_WRONLY | os.O_CREATE
			if statErr == nil {
				flags |= os.O_TRUNC
			} else {
				flags |= os.O_EXCL
			}
			f, e := os.OpenFile(p, flags, 0600)
			if e != nil {
				return &ExitError{4, e}
			}
			secret := make([]byte, 32)
			if _, e = rand.Read(secret); e != nil {
				_ = f.Close()
				return &ExitError{4, e}
			}
			value := base64.RawStdEncoding.EncodeToString(secret) + "\n"
			if name == "forwarding_secret" {
				value = hex.EncodeToString(secret) + "\n"
			}
			if _, e = io.WriteString(f, value); e != nil {
				_ = f.Close()
				return &ExitError{4, e}
			}
			_ = f.Close()
		} else if statErr != nil {
			return &ExitError{4, statErr}
		}
		if err := os.Chmod(p, 0600); err != nil {
			return &ExitError{4, err}
		}
	}
	if err := c.ValidateSecrets(); err != nil {
		return &ExitError{3, err}
	}
	if err := c.ValidateEnvironmentFile(); err != nil {
		return &ExitError{3, err}
	}
	if err := os.Chmod(filepath.Join(c.ProjectDir, ".env"), 0600); err != nil {
		return &ExitError{4, err}
	}
	if _, err := a.compose(ctx, c, "config", "--quiet"); err != nil {
		return &ExitError{3, err}
	}
	pm := pluginmanager.New(c.PluginManagerManifest(), c.PluginManagerLock())
	if err := pm.Sync(ctx, a.Out, true); err != nil {
		return &ExitError{4, fmt.Errorf("plugin synchronization: %w", err)}
	}
	if err := pm.SyncComponents(ctx, a.Out, true); err != nil {
		return &ExitError{4, fmt.Errorf("component synchronization: %w", err)}
	}
	if err := prepareRuntimeConfigs(c); err != nil {
		return &ExitError{4, err}
	}
	fmt.Fprintln(a.Out, "Installation prepared. Review secrets and pinned images before starting.")
	return nil
}

func prepareRuntimeConfigs(c Config) error {
	network, err := loadNetworkSettings(c.ProjectDir)
	if err != nil {
		return err
	}
	secret, err := os.ReadFile(filepath.Join(c.ProjectDir, "secrets", "forwarding_secret"))
	if err != nil {
		return err
	}
	secretValue := strings.TrimSpace(string(secret))
	if secretValue == "" || strings.ContainsAny(secretValue, "\r\n") {
		return errors.New("forwarding secret is empty")
	}
	luckpermsSecret, err := os.ReadFile(filepath.Join(c.ProjectDir, "secrets", "luckperms_password"))
	if err != nil {
		return err
	}
	luckpermsValue := strings.TrimSpace(string(luckpermsSecret))
	if luckpermsValue == "" || strings.ContainsAny(luckpermsValue, "\r\n") {
		return errors.New("LuckPerms password is empty")
	}
	veloauthSecret, err := os.ReadFile(filepath.Join(c.ProjectDir, "secrets", "veloauth_password"))
	if err != nil {
		return err
	}
	veloauthValue := strings.TrimSpace(string(veloauthSecret))
	if veloauthValue == "" || strings.ContainsAny(veloauthValue, "\r\n") {
		return errors.New("VeloAuth password is empty")
	}
	var ultimateteamsValue string
	if ultimateteamsSecret, err := os.ReadFile(filepath.Join(c.ProjectDir, "secrets", "ultimateteams_password")); err == nil {
		ultimateteamsValue = strings.TrimSpace(string(ultimateteamsSecret))
	}
	var libertybansValue string
	if libertybansSecret, err := os.ReadFile(filepath.Join(c.ProjectDir, "secrets", "libertybans_password")); err == nil {
		libertybansValue = strings.TrimSpace(string(libertybansSecret))
	}
	passwords := map[string]string{
		"__VELOAUTH_PASSWORD__":      veloauthValue,
		"__LUCKPERMS_PASSWORD__":     luckpermsValue,
		"__ULTIMATETEAMS_PASSWORD__": ultimateteamsValue,
		"__LIBERTYBANS_PASSWORD__":   libertybansValue,
		"__FORWARDING_SECRET__":      secretValue,
	}

	if err := renderConfigTree(filepath.Join(c.ProjectDir, "configs/proxy/plugins"), filepath.Join(c.ProjectDir, "servers/proxy/plugins"), network, passwords); err != nil {
		return err
	}
	if err := renderConfigTree(filepath.Join(c.ProjectDir, "configs/lobby/plugins"), filepath.Join(c.ProjectDir, "servers/lobby/plugins"), network, passwords); err != nil {
		return err
	}
	if err := renderConfigTree(filepath.Join(c.ProjectDir, "configs/survival/plugins"), filepath.Join(c.ProjectDir, "servers/survival/plugins"), network, passwords); err != nil {
		return err
	}

	manifestBytes, mErr := os.ReadFile(c.PluginManagerManifest())
	if mErr == nil {
		var manifest pluginmanager.Manifest
		if err := yaml.Unmarshal(manifestBytes, &manifest); err == nil {
			for _, p := range manifest.Plugins {
				if !p.Enabled || p.ConfigDir == "" {
					continue
				}
				targetDir := p.ConfigTargetDir
				if targetDir == "" {
					targetDir = filepath.Base(p.ConfigDir)
				}
				sourcePath := filepath.Join(c.ProjectDir, p.ConfigDir)
				for _, target := range p.Servers {
					destPath := filepath.Join(c.ProjectDir, "servers", target, "plugins", targetDir)
					currentSource := sourcePath
					serverSpecificSource := filepath.Join(c.ProjectDir, "configs", target, "plugins", filepath.Base(p.ConfigDir))
					if _, err := os.Stat(serverSpecificSource); err == nil {
						currentSource = serverSpecificSource
					}
					if err := renderConfigTree(currentSource, destPath, network, passwords); err != nil {
						return err
					}
					baseName := filepath.Base(p.ConfigDir)
					if targetDir != baseName {
						staleDir := filepath.Join(c.ProjectDir, "servers", target, "plugins", baseName)
						_ = os.RemoveAll(staleDir)
					}
				}
			}
		}
	}

	velocityConfig := filepath.Join(c.ProjectDir, "servers", "proxy", "velocity.toml")
	if err := renderConfigFile(filepath.Join(c.ProjectDir, "configs/proxy/velocity.toml"), velocityConfig, network, passwords); err != nil {
		return err
	}
	for _, server := range []string{"lobby", "survival"} {
		dir := filepath.Join(c.ProjectDir, "servers", server)
		if err = os.MkdirAll(filepath.Join(dir, "config"), 0750); err != nil {
			return err
		}
		if err = renderConfigFile(filepath.Join(c.ProjectDir, "configs", server, "server.properties"), filepath.Join(dir, "server.properties"), network, passwords); err != nil {
			return err
		}
		if err = renderConfigFile(filepath.Join(c.ProjectDir, "configs", server, "paper-global.yml"), filepath.Join(dir, "config/paper-global.yml"), network, passwords); err != nil {
			return err
		}
		worldDefaults := filepath.Join(c.ProjectDir, "configs", server, "paper-world-defaults.yml")
		if _, err := os.Stat(worldDefaults); err == nil {
			if err = renderConfigFile(worldDefaults, filepath.Join(dir, "config/paper-world-defaults.yml"), network, passwords); err != nil {
				return err
			}
		}
		bukkitFile := filepath.Join(c.ProjectDir, "configs", server, "bukkit.yml")
		if _, err := os.Stat(bukkitFile); err == nil {
			if err = renderConfigFile(bukkitFile, filepath.Join(dir, "bukkit.yml"), network, passwords); err != nil {
				return err
			}
		}
	}

	survivalDatapacksSrc := filepath.Join(c.ProjectDir, "configs", "survival", "datapacks")
	if info, err := os.Stat(survivalDatapacksSrc); err == nil && info.IsDir() {
		survivalDatapacksDst := filepath.Join(c.ProjectDir, "servers", "survival", "world", "datapacks")
		if err := os.MkdirAll(survivalDatapacksDst, 0755); err == nil {
			entries, _ := os.ReadDir(survivalDatapacksSrc)
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".zip") {
					srcPath := filepath.Join(survivalDatapacksSrc, entry.Name())
					dstPath := filepath.Join(survivalDatapacksDst, entry.Name())
					if err := fileutil.CopyFile(srcPath, dstPath, 0640); err != nil {
						return fmt.Errorf("copy datapack %s: %w", entry.Name(), err)
					}
				}
			}
		}
	}
	if err := renderConfigFile(filepath.Join(c.ProjectDir, "configs/auth/settings.yml"), filepath.Join(c.ProjectDir, "servers/auth/settings.yml"), network, passwords); err != nil {
		return err
	}

	for _, target := range []string{"proxy", "lobby", "survival"} {
		pluginsDir := filepath.Join(c.ProjectDir, "servers", target, "plugins")
		if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
			continue
		}
		walkErr := filepath.Walk(pluginsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".yml" && ext != ".yaml" && ext != ".conf" && ext != ".txt" && ext != ".properties" && ext != ".toml" && ext != ".json" {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			content := string(data)
			original := content
			for k, v := range network.variables() {
				content = strings.ReplaceAll(content, k, v)
			}
			for k, v := range passwords {
				content = strings.ReplaceAll(content, k, v)
			}
			if content != original {
				if writeErr := os.WriteFile(path, []byte(content), info.Mode().Perm()); writeErr != nil {
					return writeErr
				}
			}
			return nil
		})
		if walkErr != nil {
			return walkErr
		}
	}
	return nil
}

type networkSettings struct {
	Network struct {
		Name           string `yaml:"name"`
		ShortName      string `yaml:"short-name"`
		Domain         string `yaml:"domain"`
		Discord        string `yaml:"discord"`
		Website        string `yaml:"website"`
		MOTDLine1      string `yaml:"motd-line-1"`
		MOTDLine2      string `yaml:"motd-line-2"`
		PrimaryColor   string `yaml:"primary-color"`
		SecondaryColor string `yaml:"secondary-color"`
		MaxPlayers     int    `yaml:"max-players"`
	} `yaml:"network"`
	Locale string `yaml:"locale"`
	Theme  struct {
		Preset  string                      `yaml:"preset"`
		Presets map[string]i18n.ThemeColors `yaml:"presets"`
	} `yaml:"theme"`
	Lobby struct {
		DisplayName string `yaml:"display-name"`
	} `yaml:"lobby"`
	Auth struct {
		DisplayName string `yaml:"display-name"`
	} `yaml:"auth"`
	resolvedTheme i18n.ThemeColors
	messages      map[string]string
}

func loadNetworkSettings(projectDir string) (networkSettings, error) {
	var s networkSettings
	b, err := os.ReadFile(filepath.Join(projectDir, "configs/network.yml"))
	if err != nil {
		return s, err
	}
	if err := yaml.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("invalid configs/network.yml: %w", err)
	}
	if s.Network.Name == "" || s.Network.ShortName == "" || s.Network.MOTDLine1 == "" || s.Network.MOTDLine2 == "" || s.Network.MaxPlayers < 1 || s.Lobby.DisplayName == "" || s.Auth.DisplayName == "" {
		return s, errors.New("configs/network.yml has empty required values")
	}
	if s.Network.Discord == "" {
		s.Network.Discord = "dc." + s.Network.Domain
	}
	if s.Network.Website == "" {
		s.Network.Website = s.Network.Domain
	}

	s.resolvedTheme = i18n.ResolveTheme(s.Theme.Preset, s.Theme.Presets)
	if s.Network.PrimaryColor != "" {
		s.resolvedTheme.Primary = s.Network.PrimaryColor
	}
	if s.Network.SecondaryColor != "" {
		s.resolvedTheme.Secondary = s.Network.SecondaryColor
	}
	s.Network.PrimaryColor = s.resolvedTheme.Primary
	s.Network.SecondaryColor = s.resolvedTheme.Secondary

	if s.Locale == "" {
		s.Locale = "en"
	}
	msgs, err := i18n.LoadLocale(projectDir, s.Locale)
	if err != nil {
		msgs = map[string]string{}
	}
	s.messages = msgs

	return s, nil
}

func (s networkSettings) variables() map[string]string {
	vars := map[string]string{
		"__NETWORK_NAME__":            s.Network.Name,
		"__NETWORK_SHORT_NAME__":      s.Network.ShortName,
		"__NETWORK_DOMAIN__":          s.Network.Domain,
		"__NETWORK_DISCORD__":         s.Network.Discord,
		"__NETWORK_WEBSITE__":         s.Network.Website,
		"__NETWORK_MOTD_LINE_1__":     s.Network.MOTDLine1,
		"__NETWORK_MOTD_LINE_2__":     s.Network.MOTDLine2,
		"__NETWORK_PRIMARY_COLOR__":   s.Network.PrimaryColor,
		"__NETWORK_SECONDARY_COLOR__": s.Network.SecondaryColor,
		"__NETWORK_MAX_PLAYERS__":     strconv.Itoa(s.Network.MaxPlayers),
		"__LOBBY_DISPLAY_NAME__":      s.Lobby.DisplayName,
		"__AUTH_DISPLAY_NAME__":       s.Auth.DisplayName,
		"__LOCALE__":                  s.Locale,
	}

	for k, v := range i18n.TemplateVariables(s.resolvedTheme, s.messages) {
		vars[k] = v
	}

	return vars
}

func isBinaryFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".ico", ".jar", ".zip", ".gz", ".age", ".tar", ".dat", ".bin":
		return true
	default:
		return false
	}
}

func renderConfigFile(source, target string, settings networkSettings, extra map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0750); err != nil {
		return err
	}

	if isBinaryFile(source) {
		return fileutil.CopyFile(source, target, 0640)
	}

	b, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	value := string(b)
	vars := settings.variables()
	for pass := 0; pass < 5; pass++ {
		changed := false
		for from, to := range vars {
			if strings.Contains(value, from) {
				value = strings.ReplaceAll(value, from, to)
				changed = true
			}
		}
		for from, to := range extra {
			if strings.Contains(value, from) {
				value = strings.ReplaceAll(value, from, to)
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	for _, prefix := range []string{"__NETWORK_", "__LOBBY_", "__AUTH_", "__FORWARDING_SECRET__", "__THEME_", "__MSG_"} {
		if idx := strings.Index(value, prefix); idx != -1 {
			end := strings.Index(value[idx+len(prefix):], "__")
			token := prefix
			if end != -1 {
				token = value[idx : idx+len(prefix)+end+2]
			}
			return fmt.Errorf("unresolved configuration placeholder %s in %s", token, source)
		}
	}
	// #nosec G306 -- runtime configs require group readability for container execution
	return os.WriteFile(target, []byte(value), 0640)
}

func renderConfigTree(source, target string, settings networkSettings, extra map[string]string) error {
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return nil
	}
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0750); err != nil {
			return err
		}
		return renderConfigFile(path, dest, settings, extra)
	})
}
