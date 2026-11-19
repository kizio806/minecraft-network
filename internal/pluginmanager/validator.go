package pluginmanager

import (
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var targetRoots = map[string]string{
	"proxy":    "servers/proxy/plugins",
	"lobby":    "servers/lobby/plugins",
	"survival": "servers/survival/plugins",
}

func Validate(manifest Manifest, lock Lockfile) error {
	if manifest.SchemaVersion != 1 || lock.SchemaVersion != 1 {
		return errors.New("schema-version must be 1")
	}
	if manifest.Environment.MinecraftVersion != "1.21.11" {
		return fmt.Errorf("unsupported minecraft-version %q", manifest.Environment.MinecraftVersion)
	}
	if len(manifest.Plugins) == 0 {
		return errors.New("plugins must not be empty")
	}
	for id, p := range manifest.Plugins {
		if !validID(id) {
			return fmt.Errorf("invalid plugin identifier %q", id)
		}
		if p.Enabled && (p.Source == "" || p.Project == "" || p.Version == "" || p.VersionID == "") {
			return fmt.Errorf("plugin %s lacks immutable source/version fields", id)
		}
		if p.Source != "modrinth" && p.Source != "github" && p.Source != "auxilor" {
			return fmt.Errorf("plugin %s uses unsupported provider %q", id, p.Source)
		}
		if strings.Contains(strings.ToLower(p.Version), "latest") || strings.Contains(strings.ToLower(p.VersionID), "latest") {
			return fmt.Errorf("plugin %s uses latest", id)
		}
		if p.UpdatePolicy != "manual" {
			return fmt.Errorf("plugin %s must use manual update-policy", id)
		}
		if p.Platform != "velocity" && p.Platform != "paper" {
			return fmt.Errorf("plugin %s has unsupported platform %q", id, p.Platform)
		}
		for _, target := range p.Servers {
			if _, ok := targetRoots[target]; !ok {
				return fmt.Errorf("plugin %s has unsupported target %q", id, target)
			}
		}
		if p.Enabled {
			entry, ok := lock.Plugins[id]
			if !ok {
				return fmt.Errorf("enabled plugin %s is missing from lockfile", id)
			}
			if entry.Project != p.Project || entry.VersionID != p.VersionID || entry.Version != p.Version || entry.Source != p.Source {
				return fmt.Errorf("lockfile entry for %s does not match manifest", id)
			}
			if entry.SHA256 == "" && entry.SHA512 == "" {
				return fmt.Errorf("lockfile entry for %s has no trusted digest", id)
			}
			if entry.SHA256 != "" && !hexDigest(entry.SHA256, 64) || entry.SHA512 != "" && !hexDigest(entry.SHA512, 128) {
				return fmt.Errorf("lockfile entry for %s has malformed digest", id)
			}
			if !contains(entry.GameVersions, manifest.Environment.MinecraftVersion) {
				return fmt.Errorf("lockfile entry for %s does not support Minecraft %s", id, manifest.Environment.MinecraftVersion)
			}
			if !contains(entry.Loaders, p.Platform) {
				return fmt.Errorf("lockfile entry for %s does not support platform %s", id, p.Platform)
			}
			if !sameStrings(entry.Dependencies, p.Dependencies) {
				return fmt.Errorf("lockfile dependencies for %s do not match manifest", id)
			}
			if !strings.HasPrefix(entry.DownloadURL, "https://") || floatingURL(entry.DownloadURL) {
				return fmt.Errorf("plugin %s has unsafe download URL", id)
			}
			if filepath.Base(entry.Filename) != entry.Filename || entry.Filename == "" {
				return fmt.Errorf("plugin %s has unsafe filename", id)
			}
		}
	}
	for id := range lock.Plugins {
		if _, ok := manifest.Plugins[id]; !ok {
			return fmt.Errorf("lockfile contains undeclared plugin %s", id)
		}
	}
	if err := dependencyOrder(manifest.Plugins); err != nil {
		return err
	}
	return nil
}

func floatingURL(raw string) bool {
	s := strings.ToLower(raw)
	return strings.Contains(s, "/latest/") || strings.Contains(s, "/releases/latest") || strings.HasSuffix(s, "/latest") || strings.Contains(s, "?version=latest")
}

func validID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if !(r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func hexDigest(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func dependencyOrder(plugins map[string]PluginSpec) error {
	state := map[string]uint8{}
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case 1:
			return fmt.Errorf("dependency cycle includes %s", id)
		case 2:
			return nil
		}
		p, ok := plugins[id]
		if !ok {
			return fmt.Errorf("missing dependency %s", id)
		}
		state[id] = 1
		for _, dep := range p.Dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range plugins {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}
