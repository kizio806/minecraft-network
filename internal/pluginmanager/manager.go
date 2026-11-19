package pluginmanager

import (
	"context"
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

func (m *Manager) Validate() error {
	_, _, err := m.load()
	return err
}

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

func (m *Manager) readInstalled() (InstalledState, error) {
	return decode[InstalledState](filepath.Join(m.root, ".plugin-manager", "installed.yml"))
}

func (m *Manager) UpdateCheck(ctx context.Context, w io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, lock, err := m.load()
	if err != nil {
		return err
	}
	for id, entry := range lock.Plugins {
		if err := ctx.Err(); err != nil {
			return err
		}
		fmt.Fprintf(w, "%s\tlocked=%s\tupdate-check=manual-provider-required\n", id, entry.Version)
	}
	return nil
}
