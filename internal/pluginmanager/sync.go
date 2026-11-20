package pluginmanager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kizio806/minecraft-network/internal/fileutil"
	"gopkg.in/yaml.v3"
)

func (m *Manager) Sync(ctx context.Context, w io.Writer, apply bool) error {
	manifest, lock, err := m.load()
	if err != nil {
		return err
	}
	if !apply {
		return m.Plan(w)
	}
	for id, p := range manifest.Plugins {
		if p.Enabled && p.ConfigDir != "" {
			if err := m.validateConfigDir(p.ConfigDir); err != nil {
				return fmt.Errorf("%s config: %w", id, err)
			}
		}
	}
	var targets []string
	for _, p := range manifest.Plugins {
		if p.Enabled {
			targets = append(targets, p.Servers...)
		}
	}
	if err := m.ensureStopped(ctx, targets); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(m.root, ".plugin-manager"), 0700); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Join(m.root, ".plugin-manager"), "stage-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	var changes []change
	rollbackRoot := filepath.Join(m.root, ".plugin-manager", "rollback", time.Now().UTC().Format("20060102T150405Z"))
	if err := os.MkdirAll(rollbackRoot, 0700); err != nil {
		return err
	}
	transactionFailed := true
	defer func() {
		if transactionFailed {
			_ = rollback(changes)
		}
	}()
	for id, p := range manifest.Plugins {
		if !p.Enabled {
			continue
		}
		entry := lock.Plugins[id]
		path := filepath.Join(stage, entry.Filename)
		if err := download(ctx, entry, path); err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
		if err := inspectJar(path, p.Platform); err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
		for _, target := range p.Servers {
			c, err := m.installOne(path, entry, target, rollbackRoot)
			if err != nil {
				return fmt.Errorf("%s/%s: %w", target, id, err)
			}
			changes = append(changes, c)
			if p.ConfigDir != "" {
				configChanges, err := m.installConfigTree(p.ConfigDir, p.ConfigTargetDir, target, rollbackRoot)
				if err != nil {
					return fmt.Errorf("%s/%s config: %w", target, id, err)
				}
				changes = append(changes, configChanges...)
			}
		}
		fmt.Fprintf(w, "SYNCED\t%s\t%s\n", id, entry.Version)
	}
	if err := m.writeInstalled(lock); err != nil {
		return err
	}
	if err := m.writeRollback(changes); err != nil {
		return err
	}
	transactionFailed = false
	return nil
}

func (m *Manager) ensureStopped(ctx context.Context, targets []string) error {
	compose := filepath.Join(m.root, "docker-compose.yml")
	if _, err := os.Stat(compose); err != nil {
		return err
	}
	args := []string{"compose", "-f", compose, "ps", "--status", "running", "--services"}
	dockerBin := "docker"
	if p, lookErr := exec.LookPath("docker"); lookErr == nil {
		dockerBin = p
	}
	out, err := exec.CommandContext(ctx, dockerBin, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot verify stopped services: %w", err)
	}
	running := string(out)
	for _, target := range targets {
		service := target
		if strings.Contains("\n"+running+"\n", "\n"+service+"\n") {
			return fmt.Errorf("target service %s is running; stop it before sync", service)
		}
	}
	return nil
}

func (m *Manager) validateConfigDir(configDir string) error {
	root := filepath.Clean(m.root)
	source := filepath.Clean(filepath.Join(root, configDir))
	rel, err := filepath.Rel(root, source)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("config-dir escapes repository root")
	}
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("config-dir is not a directory")
	}
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("config-dir contains a symlink")
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return errors.New("config-dir contains a non-regular file")
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".yml") || strings.HasSuffix(strings.ToLower(entry.Name()), ".yaml") {
			if _, err := decode[map[string]any](path); err != nil {
				return fmt.Errorf("invalid YAML %s: %w", path, err)
			}
		}
		return nil
	})
}

func (m *Manager) installConfigTree(configDir, targetDir, target, rollbackRoot string) ([]change, error) {
	root := filepath.Clean(m.root)
	source := filepath.Clean(filepath.Join(root, configDir))
	rel, err := filepath.Rel(root, source)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, errors.New("config-dir escapes repository root")
	}
	info, err := os.Stat(source)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("config-dir is not a directory")
	}
	if targetDir == "" {
		targetDir = filepath.Base(source)
	}
	if filepath.Base(targetDir) != targetDir || targetDir == "." || targetDir == ".." {
		return nil, errors.New("unsafe config-target-dir")
	}
	destinationRoot := filepath.Join(root, targetRoots[target], targetDir)
	var changes []change
	err = filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("config-dir contains a symlink")
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		destination := filepath.Join(destinationRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0750)
		}
		if !entry.Type().IsRegular() {
			return errors.New("config-dir contains a non-regular file")
		}
		c := change{destination: destination}
		if existing, err := os.Stat(destination); err == nil {
			if !existing.Mode().IsRegular() {
				return errors.New("config destination is not a regular file")
			}
			c.existed = true
			c.backup = filepath.Join(rollbackRoot, target, "config", relative)
			if err := os.MkdirAll(filepath.Dir(c.backup), 0700); err != nil {
				return err
			}
			if err := fileutil.CopyFile(destination, c.backup, 0600); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0750); err != nil {
			return err
		}
		tmp := filepath.Join(filepath.Dir(destination), ".install-config-") + filepath.Base(destination)
		if err := fileutil.CopyFile(path, tmp, 0640); err != nil {
			return err
		}
		if err := os.Rename(tmp, destination); err != nil {
			_ = os.Remove(tmp)
			return err
		}
		changes = append(changes, c)
		return nil
	})
	return changes, err
}

func (m *Manager) installOne(staged string, entry LockEntry, target, rollbackRoot string) (change, error) {
	dir, ok := targetRoots[target]
	if !ok {
		return change{}, fmt.Errorf("unsupported target %s", target)
	}
	dir = filepath.Join(m.root, dir)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return change{}, err
	}
	destination := filepath.Join(dir, entry.Filename)
	c := change{destination: destination}
	if _, err := os.Stat(destination); err == nil {
		c.existed = true
		c.backup = filepath.Join(rollbackRoot, target+"-"+entry.Filename)
		if err := os.MkdirAll(filepath.Dir(c.backup), 0700); err != nil {
			return c, err
		}
		if err := fileutil.CopyFile(destination, c.backup, 0600); err != nil {
			return c, err
		}
	} else if !os.IsNotExist(err) {
		return c, err
	}
	tmp := filepath.Join(dir, ".install-"+entry.Filename)
	if err := fileutil.CopyFile(staged, tmp, 0640); err != nil {
		return c, err
	}
	defer os.Remove(tmp)
	if err := os.Rename(tmp, destination); err != nil {
		return c, err
	}
	return c, nil
}

func (m *Manager) writeInstalled(lock Lockfile) error {
	dir := filepath.Join(m.root, ".plugin-manager")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	b, err := yaml.Marshal(InstalledState{InstalledAt: time.Now().UTC(), Plugins: lock.Plugins})
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, ".installed.yml.tmp")
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "installed.yml"))
}
