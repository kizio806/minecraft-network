package pluginmanager

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Manager struct {
	manifestPath string
	lockPath     string
	root         string
}

func New(manifestPath, lockPath string) *Manager {
	root := filepath.Clean(filepath.Join(filepath.Dir(manifestPath), "..", ".."))
	return &Manager{manifestPath: manifestPath, lockPath: lockPath, root: root}
}

func (m *Manager) load() (Manifest, Lockfile, error) {
	manifest, err := decode[Manifest](m.manifestPath)
	if err != nil {
		return Manifest{}, Lockfile{}, fmt.Errorf("manifest: %w", err)
	}
	lock, err := decode[Lockfile](m.lockPath)
	if err != nil {
		return Manifest{}, Lockfile{}, fmt.Errorf("lockfile: %w", err)
	}
	if err := Validate(manifest, lock); err != nil {
		return Manifest{}, Lockfile{}, err
	}
	return manifest, lock, nil
}

func decode[T any](path string) (T, error) {
	var value T
	b, err := os.ReadFile(path)
	if err != nil {
		return value, err
	}
	d := yaml.NewDecoder(strings.NewReader(string(b)))
	d.KnownFields(true)
	if err := d.Decode(&value); err != nil {
		return value, err
	}
	return value, nil
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
		if p.Source != "modrinth" && p.Source != "github" {
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

var targetRoots = map[string]string{"proxy": "servers/proxy/plugins", "lobby": "servers/lobby/plugins", "survival": "servers/survival/plugins"}

func validID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if !(r == '-' || r == '_' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
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

func (m *Manager) Validate() error { _, _, err := m.load(); return err }

func (m *Manager) List(w io.Writer) error {
	manifest, lock, err := m.load()
	if err != nil {
		return err
	}
	for id, p := range manifest.Plugins {
		state := "disabled"
		if p.Enabled {
			state = "enabled"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, state, p.Version, lock.Plugins[id].VersionID)
	}
	return nil
}

func (m *Manager) Plan(w io.Writer) error {
	manifest, lock, err := m.load()
	if err != nil {
		return err
	}
	state, _ := m.readInstalled()
	for id, p := range manifest.Plugins {
		if !p.Enabled {
			fmt.Fprintf(w, "REMOVE\t%s\tdisabled\n", id)
			continue
		}
		if old, ok := state.Plugins[id]; !ok {
			fmt.Fprintf(w, "INSTALL\t%s\t%s\n", id, lock.Plugins[id].Version)
		} else if old.SHA256 != lock.Plugins[id].SHA256 || old.SHA512 != lock.Plugins[id].SHA512 {
			fmt.Fprintf(w, "UPDATE\t%s\t%s\n", id, lock.Plugins[id].Version)
		} else {
			fmt.Fprintf(w, "OK\t%s\t%s\n", id, lock.Plugins[id].Version)
		}
	}
	return nil
}

func (m *Manager) VerifyInstalled() error {
	manifest, lock, err := m.load()
	if err != nil {
		return err
	}
	for id, p := range manifest.Plugins {
		if !p.Enabled {
			continue
		}
		for _, target := range p.Servers {
			path := filepath.Join(m.root, targetRoots[target], lock.Plugins[id].Filename)
			if err := verifyFile(path, lock.Plugins[id]); err != nil {
				return fmt.Errorf("%s/%s: %w", target, id, err)
			}
		}
	}
	return nil
}

func verifyFile(path string, entry LockEntry) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h256, h512 := sha256.New(), sha512.New()
	n, err := io.Copy(io.MultiWriter(h256, h512), f)
	if err != nil {
		return err
	}
	if entry.Size > 0 && n != entry.Size {
		return fmt.Errorf("size %d does not match lockfile %d", n, entry.Size)
	}
	if entry.SHA256 != "" && hex.EncodeToString(h256.Sum(nil)) != strings.ToLower(entry.SHA256) {
		return errors.New("sha256 mismatch")
	}
	if entry.SHA512 != "" && hex.EncodeToString(h512.Sum(nil)) != strings.ToLower(entry.SHA512) {
		return errors.New("sha512 mismatch")
	}
	return nil
}

func (m *Manager) readInstalled() (InstalledState, error) {
	return decode[InstalledState](filepath.Join(m.root, ".plugin-manager", "installed.yml"))
}

func (m *Manager) UpdateCheck(ctx context.Context, w io.Writer) error {
	_, lock, err := m.load()
	if err != nil {
		return err
	}
	for id, entry := range lock.Plugins {
		fmt.Fprintf(w, "%s\tlocked=%s\tupdate-check=manual-provider-required\n", id, entry.Version)
	}
	return nil
}
