package pluginmanager

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const maxArtifactSize int64 = 100 << 20

var trustedHosts = map[string]bool{
	"cdn.modrinth.com":                     true,
	"api.modrinth.com":                     true,
	"github.com":                           true,
	"api.github.com":                       true,
	"release-assets.githubusercontent.com": true,
}

func download(ctx context.Context, entry LockEntry, destination string) error {
	u, err := url.Parse(entry.DownloadURL)
	if err != nil || u.Scheme != "https" || !trustedHosts[strings.ToLower(u.Hostname())] {
		return errors.New("download URL is not HTTPS or trusted")
	}
	client := &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		if req.URL.Scheme != "https" || !trustedHosts[strings.ToLower(req.URL.Hostname())] {
			return errors.New("redirect leaves trusted HTTPS hosts")
		}
		return nil
	}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.DownloadURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxArtifactSize {
		return errors.New("artifact exceeds maximum size")
	}
	f, err := os.CreateTemp(filepath.Dir(destination), ".artifact-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = io.Copy(f, io.LimitReader(resp.Body, maxArtifactSize+1)); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = verifyFile(tmp, entry); err != nil {
		return fmt.Errorf("download integrity: %w", err)
	}
	return os.Rename(tmp, destination)
}

func inspectJar(path, platform string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("artifact is not a valid ZIP/JAR: %w", err)
	}
	defer r.Close()
	for _, file := range r.File {
		name := filepath.ToSlash(file.Name)
		if platform == "velocity" && name == "velocity-plugin.json" {
			return nil
		}
		if platform == "paper" && (name == "plugin.yml" || name == "paper-plugin.yml") {
			return nil
		}
	}
	return fmt.Errorf("artifact has no %s plugin descriptor", platform)
}

func (m *Manager) ensureStopped(ctx context.Context, targets []string) error {
	compose := filepath.Join(m.root, "docker-compose.yml")
	if _, err := os.Stat(compose); err != nil {
		return err
	}
	args := []string{"compose", "-f", compose, "ps", "--status", "running", "--services"}
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot verify stopped services: %w", err)
	}
	running := string(out)
	for _, target := range targets {
		service := target
		if target == "proxy" {
			service = "proxy"
		}
		if strings.Contains("\n"+running+"\n", "\n"+service+"\n") {
			return fmt.Errorf("target service %s is running; stop it before sync", service)
		}
	}
	return nil
}

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
			rollback(changes)
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
	if err := m.writeRollback(rollbackRoot, changes); err != nil {
		return err
	}
	transactionFailed = false
	return nil
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
	entry, ok := lock.Components["nanolimbo"]
	if !ok || entry.Source != "github" || entry.Filename != "NanoLimbo.jar" || entry.SHA256 == "" {
		return errors.New("nanolimbo component lock is incomplete")
	}
	if !apply {
		fmt.Fprintf(w, "INSTALL\tnanolimbo\t%s\n", entry.Version)
		return nil
	}
	if err := m.ensureStopped(ctx, []string{"auth"}); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Join(m.root, ".plugin-manager"), "component-stage-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	path = filepath.Join(stage, entry.Filename)
	lockEntry := LockEntry{DownloadURL: entry.DownloadURL, Filename: entry.Filename, SHA256: entry.SHA256}
	if err := download(ctx, lockEntry, path); err != nil {
		return fmt.Errorf("nanolimbo: %w", err)
	}
	destination := filepath.Join(m.root, "servers/auth", entry.Filename)
	backup := filepath.Join(m.root, ".plugin-manager/rollback", "nanolimbo-"+entry.Filename)
	if err := os.MkdirAll(filepath.Dir(destination), 0750); err != nil {
		return err
	}
	if _, err := os.Stat(destination); err == nil {
		if err := os.MkdirAll(filepath.Dir(backup), 0700); err != nil {
			return err
		}
		if err := copyFile(destination, backup); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	tmp := destination + ".component-install"
	if err := copyFile(path, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, destination); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	fmt.Fprintf(w, "SYNCED\tnanolimbo\t%s\n", entry.Version)
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
			if err := copyFile(destination, c.backup); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0750); err != nil {
			return err
		}
		tmp := filepath.Join(filepath.Dir(destination), ".install-config-") + filepath.Base(destination)
		if err := copyFile(path, tmp); err != nil {
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

type change struct {
	destination, backup string
	existed             bool
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
		if err := copyFile(destination, c.backup); err != nil {
			return c, err
		}
	} else if !os.IsNotExist(err) {
		return c, err
	}
	tmp := filepath.Join(dir, ".install-"+entry.Filename)
	if err := copyFile(staged, tmp); err != nil {
		return c, err
	}
	defer os.Remove(tmp)
	if err := os.Rename(tmp, destination); err != nil {
		return c, err
	}
	return c, nil
}

func rollback(changes []change) {
	for i := len(changes) - 1; i >= 0; i-- {
		c := changes[i]
		if c.existed {
			_ = os.Rename(c.backup, c.destination)
		} else {
			_ = os.Remove(c.destination)
		}
	}
}

func (m *Manager) writeRollback(root string, changes []change) error {
	files := make([]RollbackFile, 0, len(changes))
	for _, c := range changes {
		files = append(files, RollbackFile{Destination: c.destination, Backup: c.backup, Existed: c.existed})
	}
	b, err := yaml.Marshal(RollbackState{CreatedAt: time.Now().UTC().Format(time.RFC3339), Files: files})
	if err != nil {
		return err
	}
	tmp := filepath.Join(m.root, ".plugin-manager", ".rollback.yml.tmp")
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(m.root, ".plugin-manager", "rollback.yml"))
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
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

func (m *Manager) Rollback() error {
	state, err := decode[RollbackState](filepath.Join(m.root, ".plugin-manager", "rollback.yml"))
	if err != nil {
		return fmt.Errorf("no rollback snapshot: %w", err)
	}
	if err := m.ensureStopped(context.Background(), []string{"proxy", "lobby", "survival"}); err != nil {
		return err
	}
	for i := len(state.Files) - 1; i >= 0; i-- {
		f := state.Files[i]
		if f.Existed {
			if err := os.Rename(f.Backup, f.Destination); err != nil {
				return err
			}
		} else if err := os.Remove(f.Destination); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
