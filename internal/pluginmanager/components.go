package pluginmanager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/kizio806/minecraft-network/internal/fileutil"
)

func (m *Manager) componentDestination(name string, entry ComponentLockEntry) (dest string, service string) {
	if name == "nanolimbo" || entry.Filename == "NanoLimbo.jar" {
		return filepath.Join(m.root, "servers/auth", entry.Filename), "auth"
	}
	return filepath.Join(m.root, "servers/survival/world/datapacks", entry.Filename), "survival"
}

func (m *Manager) SyncComponents(ctx context.Context, w io.Writer, apply bool) error {
	path := filepath.Join(m.root, "configs/plugins/components.lock.yml")
	lock, err := decode[ComponentLockfile](path)
	if err != nil {
		return fmt.Errorf("component lockfile: %w", err)
	}
	if lock.SchemaVersion != 1 {
		return errors.New("component lockfile schema-version must be 1")
	}
	if len(lock.Components) == 0 {
		return errors.New("component lockfile has no components defined")
	}

	names := make([]string, 0, len(lock.Components))
	for name := range lock.Components {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		entry := lock.Components[name]
		if entry.Filename == "" || entry.SHA256 == "" || entry.DownloadURL == "" {
			return fmt.Errorf("component %q lock entry is incomplete", name)
		}
	}

	if !apply {
		for _, name := range names {
			entry := lock.Components[name]
			fmt.Fprintf(w, "INSTALL\t%s\t%s\n", name, entry.Version)
		}
		return nil
	}

	servicesToStopMap := map[string]bool{}
	for _, name := range names {
		_, svc := m.componentDestination(name, lock.Components[name])
		if svc != "" {
			servicesToStopMap[svc] = true
		}
	}
	servicesToStop := make([]string, 0, len(servicesToStopMap))
	for svc := range servicesToStopMap {
		servicesToStop = append(servicesToStop, svc)
	}
	sort.Strings(servicesToStop)

	if err := m.ensureStopped(ctx, servicesToStop); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(m.root, ".plugin-manager"), 0700); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Join(m.root, ".plugin-manager"), "component-stage-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)

	for _, name := range names {
		entry := lock.Components[name]
		stagePath := filepath.Join(stage, entry.Filename)
		lockEntry := LockEntry{
			DownloadURL: entry.DownloadURL,
			Filename:    entry.Filename,
			SHA256:      entry.SHA256,
		}
		if err := download(ctx, lockEntry, stagePath); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}

		destination, _ := m.componentDestination(name, entry)
		backup := filepath.Join(m.root, ".plugin-manager/rollback", name+"-"+entry.Filename)
		if err := os.MkdirAll(filepath.Dir(destination), 0750); err != nil {
			return err
		}
		if _, err := os.Stat(destination); err == nil {
			if err := os.MkdirAll(filepath.Dir(backup), 0700); err != nil {
				return err
			}
			if err := fileutil.CopyFile(destination, backup, 0600); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}

		tmp := destination + ".component-install"
		if err := fileutil.CopyFile(stagePath, tmp, 0640); err != nil {
			return err
		}
		if err := os.Rename(tmp, destination); err != nil {
			_ = os.Remove(tmp)
			return err
		}
		fmt.Fprintf(w, "SYNCED\t%s\t%s\n", name, entry.Version)
	}

	return nil
}
