package backup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kizio806/minecraft-network/internal/compose"
	"github.com/kizio806/minecraft-network/internal/config"
	"github.com/kizio806/minecraft-network/internal/fileutil"
)

// Manager coordinates end-to-end backup creation, rotation, and atomic restore.
type Manager struct {
	Config  config.Config
	Compose *compose.Client
	Runner  compose.Runner
}

// NewManager creates a backup manager instance.
func NewManager(cfg config.Config, client *compose.Client, runner compose.Runner) *Manager {
	if runner == nil {
		runner = compose.ExecRunner{}
	}
	if client == nil {
		client = compose.NewClient(runner, cfg.ProjectDir, cfg.ComposeFile())
	}
	return &Manager{
		Config:  cfg,
		Compose: client,
		Runner:  runner,
	}
}

// List returns a sorted slice of available backup files.
func (m *Manager) List() ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(m.Config.BackupDir, "minecraft-network-*.tar.gz.age"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

// Cleanup retains the latest N backups, removing older ones and their checksums.
func (m *Manager) Cleanup(retention int) error {
	if retention < 1 {
		retention = 1
	}
	matches, err := filepath.Glob(filepath.Join(m.Config.BackupDir, "minecraft-network-*.tar.gz.age"))
	if err != nil {
		return err
	}
	sort.Slice(matches, func(i, j int) bool {
		a, _ := os.Stat(matches[i])
		b, _ := os.Stat(matches[j])
		return a.ModTime().After(b.ModTime())
	})
	if retention > len(matches) {
		retention = len(matches)
	}
	for _, p := range matches[retention:] {
		if err := os.Remove(p); err != nil {
			return err
		}
		if err := os.Remove(p + ".sha256"); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// Create performs an atomic, encrypted backup with database dump and integrity hash.
func (m *Manager) Create(ctx context.Context) (string, string, error) {
	lock, err := config.AcquireLock(m.Config.ProjectDir)
	if err != nil {
		return "", "", err
	}
	defer lock.Close()

	if err := m.Config.ValidateSecrets(); err != nil {
		return "", "", err
	}
	if err := m.Config.ValidateEnvironmentFile(); err != nil {
		return "", "", err
	}
	if _, err := m.Compose.Compose(ctx, "", "config", "--quiet"); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(m.Config.BackupDir, 0700); err != nil {
		return "", "", err
	}
	work, err := os.MkdirTemp("", "minecraft-backup-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(work)

	if _, err := m.Compose.Compose(ctx, "", append([]string{"stop"}, compose.ManagedServices...)...); err != nil {
		return "", "", err
	}
	defer func() {
		_, _ = m.Compose.Compose(context.Background(), "", append([]string{"start"}, compose.ManagedServices...)...)
	}()

	if err := os.MkdirAll(filepath.Join(work, "repo"), 0700); err != nil {
		return "", "", err
	}
	if container, err := m.Compose.Compose(ctx, "", "ps", "-q", "postgres"); err == nil && strings.TrimSpace(container) != "" {
		dumpFile, dumpCreateErr := os.OpenFile(filepath.Join(work, "postgres.dump"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if dumpCreateErr != nil {
			return "", "", dumpCreateErr
		}
		var errBuf bytes.Buffer
		dumpErr := m.Runner.RunStream(ctx, m.Config.ProjectDir, dumpFile, &errBuf, "docker", "compose", "-f", m.Config.ComposeFile(), "exec", "-T", "postgres", "sh", "-c", `PGPASSWORD="$(cat /run/secrets/postgres_password)" pg_dump --format=custom --no-owner --username="$POSTGRES_USER" "$POSTGRES_DB"`)
		_ = dumpFile.Close()
		if dumpErr != nil {
			return "", "", fmt.Errorf("postgres dump failed: %w: %s", dumpErr, strings.TrimSpace(errBuf.String()))
		}
	}
	plain := filepath.Join(work, "payload.tar.gz")
	archive := filepath.Join(m.Config.BackupDir, "minecraft-network-"+time.Now().UTC().Format("20060102T150405Z")+".tar.gz.age")
	if err := CreateBackupArchive(m.Config.ProjectDir, work, plain); err != nil {
		return "", "", err
	}
	if err := EncryptBackup(plain, archive+".tmp", m.Config.BackupRecipientFile); err != nil {
		return "", "", err
	}
	if err := os.Rename(archive+".tmp", archive); err != nil {
		return "", "", err
	}
	hash, err := FileSHA256(archive)
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(archive+".sha256", []byte(hash+"  "+filepath.Base(archive)+"\n"), 0600); err != nil {
		return "", "", err
	}
	if err := m.Cleanup(m.Config.BackupRetention); err != nil {
		return "", "", err
	}
	return archive, hash, nil
}

// Restore decrypts, verifies, and restores files and database state from an archive.
func (m *Manager) Restore(ctx context.Context, archive string) error {
	lock, err := config.AcquireLock(m.Config.ProjectDir)
	if err != nil {
		return err
	}
	defer lock.Close()

	if _, err := os.Stat(archive); err != nil {
		return err
	}
	if !strings.HasSuffix(archive, ".tar.gz.age") {
		return errors.New("only encrypted .tar.gz.age backups are supported")
	}
	side, err := os.ReadFile(archive + ".sha256")
	if err != nil {
		return errors.New("backup checksum file is missing")
	}
	fields := strings.Fields(string(side))
	if len(fields) < 1 || len(fields[0]) != 64 {
		return errors.New("backup checksum file is invalid")
	}
	got, err := FileSHA256(archive)
	if err != nil || got != strings.ToLower(fields[0]) {
		return errors.New("backup checksum mismatch")
	}

	plain, err := DecryptBackup(archive, m.Config.BackupIdentityFile)
	if err != nil {
		return err
	}
	defer os.Remove(plain)

	temp, err := os.MkdirTemp("", "minecraft-restore-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)

	if err := ExtractBackupArchive(plain, temp); err != nil {
		return err
	}

	if _, err := m.Compose.Compose(ctx, "", append([]string{"stop"}, compose.ManagedServices...)...); err != nil {
		return err
	}
	defer func() {
		_, _ = m.Compose.Compose(context.Background(), "", append([]string{"start"}, compose.ManagedServices...)...)
	}()

	for _, rel := range []string{"configs", "docker", "infrastructure", "servers", "secrets", ".env"} {
		src := filepath.Join(temp, rel)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if err := fileutil.CopyTree(src, filepath.Join(m.Config.ProjectDir, rel), 0600); err != nil {
			return err
		}
	}
	if dump, err := os.Open(filepath.Join(temp, "database", "postgres.dump")); err == nil {
		defer dump.Close()
		if _, err := m.Compose.Compose(ctx, "", "start", "postgres"); err != nil {
			return err
		}
		if _, err := m.Runner.RunInput(ctx, m.Config.ProjectDir, "docker", dump, "compose", "-f", m.Config.ComposeFile(), "exec", "-T", "postgres", "sh", "-c", `PGPASSWORD="$(cat /run/secrets/postgres_password)" pg_restore --clean --if-exists --no-owner --username="$POSTGRES_USER" --dbname="$POSTGRES_DB"`); err != nil {
			return err
		}
	}
	return nil
}
