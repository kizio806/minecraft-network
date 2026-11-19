package backup_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/kizio806/minecraft-network/internal/backup"
	"github.com/kizio806/minecraft-network/internal/compose"
	"github.com/kizio806/minecraft-network/internal/config"
)

type mockRunner struct {
	runFunc       func(ctx context.Context, dir string, name string, args ...string) (string, error)
	runStreamFunc func(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) error
	runInputFunc  func(ctx context.Context, dir string, name string, input io.Reader, args ...string) (string, error)
}

func (m *mockRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, dir, name, args...)
	}
	return "mock-output", nil
}

func (m *mockRunner) RunStream(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) error {
	if m.runStreamFunc != nil {
		return m.runStreamFunc(ctx, dir, stdout, stderr, name, args...)
	}
	return nil
}

func (m *mockRunner) RunInteractive(_ context.Context, _ string, _ string, _ io.Reader, _, _ io.Writer, _ ...string) error {
	return nil
}

func (m *mockRunner) RunInput(ctx context.Context, dir string, name string, input io.Reader, args ...string) (string, error) {
	if m.runInputFunc != nil {
		return m.runInputFunc(ctx, dir, name, input, args...)
	}
	return "mock-input-ok", nil
}

func setupTestEnvironment(t *testing.T) (config.Config, *mockRunner) {
	t.Helper()
	root := t.TempDir()
	secretsDir := filepath.Join(root, "secrets")
	if err := os.MkdirAll(secretsDir, 0700); err != nil {
		t.Fatal(err)
	}

	requiredSecrets := []string{
		"postgres_password",
		"luckperms_password",
		"forwarding_secret",
		"grafana_admin_password",
		"veloauth_password",
		"ultimateteams_password",
		"libertybans_password",
	}
	for _, s := range requiredSecrets {
		if err := os.WriteFile(filepath.Join(secretsDir, s), []byte("testsecretvalue1234567890"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	recPath := filepath.Join(secretsDir, "backup_key.pub")
	idPath := filepath.Join(secretsDir, "backup_key")
	if err := os.WriteFile(recPath, []byte(id.Recipient().String()+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(idPath, []byte(id.String()+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	envContent := `VELOCITY_VERSION=4.3.1
VELOCITY_IMAGE=ghcr.io/papermc/velocity@sha256:0000000000000000000000000000000000000000000000000000000000000001
PURPUR_IMAGE=ghcr.io/purpurmc/purpur@sha256:0000000000000000000000000000000000000000000000000000000000000002
PURPUR_LOBBY_IMAGE=ghcr.io/purpurmc/purpur@sha256:0000000000000000000000000000000000000000000000000000000000000003
NANOLIMBO_IMAGE=ghcr.io/nanolimbo/nanolimbo@sha256:0000000000000000000000000000000000000000000000000000000000000004
MC_MONITOR_IMAGE=ghcr.io/mc-monitor/mc-monitor@sha256:0000000000000000000000000000000000000000000000000000000000000005
`
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte(envContent), 0600); err != nil {
		t.Fatal(err)
	}

	for _, d := range []string{"configs", "servers", "infrastructure"} {
		_ = os.MkdirAll(filepath.Join(root, d), 0700)
	}
	_ = os.WriteFile(filepath.Join(root, "configs", "sample.yml"), []byte("data: 1"), 0600)

	backupDir := filepath.Join(root, "backups")
	_ = os.MkdirAll(backupDir, 0700)

	cfg := config.Config{
		ProjectDir:          root,
		BackupDir:           backupDir,
		BackupRecipientFile: recPath,
		BackupIdentityFile:  idPath,
		BackupRetention:     2,
	}

	runner := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ string, args ...string) (string, error) {
			if len(args) >= 3 && args[1] == "ps" {
				return "postgres-container-id\n", nil
			}
			return "ok", nil
		},
		runStreamFunc: func(_ context.Context, _ string, stdout, _ io.Writer, _ string, _ ...string) error {
			if stdout != nil {
				_, _ = stdout.Write([]byte("mock_postgres_dump_data"))
			}
			return nil
		},
		runInputFunc: func(_ context.Context, _ string, _ string, _ io.Reader, _ ...string) (string, error) {
			return "restore-ok", nil
		},
	}

	return cfg, runner
}

func TestNewManager_Defaults(t *testing.T) {
	temp := t.TempDir()
	cfg := config.Config{ProjectDir: temp}
	mgr := backup.NewManager(cfg, nil, nil)
	if mgr == nil {
		t.Fatal("expected non-nil Manager")
	}
	if mgr.Runner == nil {
		t.Fatal("expected default Runner to be set")
	}
	if mgr.Compose == nil {
		t.Fatal("expected default Compose client to be set")
	}
}

func TestManager_List(t *testing.T) {
	cfg, runner := setupTestEnvironment(t)
	client := compose.NewClient(runner, cfg.ProjectDir, cfg.ComposeFile())
	mgr := backup.NewManager(cfg, client, runner)

	// Empty list
	list, err := mgr.List()
	if err != nil {
		t.Fatalf("unexpected error on empty list: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 backups, got %d", len(list))
	}

	// Create dummy backup files
	f1 := filepath.Join(cfg.BackupDir, "minecraft-network-20260101T000000Z.tar.gz.age")
	f2 := filepath.Join(cfg.BackupDir, "minecraft-network-20260102T000000Z.tar.gz.age")
	_ = os.WriteFile(f1, []byte("b1"), 0600)
	_ = os.WriteFile(f2, []byte("b2"), 0600)

	list, err = mgr.List()
	if err != nil {
		t.Fatalf("unexpected error listing backups: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(list))
	}
	if list[0] != f1 || list[1] != f2 {
		t.Fatalf("unexpected sort order in List: %v", list)
	}
}

func TestManager_Cleanup(t *testing.T) {
	cfg, runner := setupTestEnvironment(t)
	client := compose.NewClient(runner, cfg.ProjectDir, cfg.ComposeFile())
	mgr := backup.NewManager(cfg, client, runner)

	// Create 3 backup files with distinct mod times
	files := []string{
		filepath.Join(cfg.BackupDir, "minecraft-network-20260101T000000Z.tar.gz.age"),
		filepath.Join(cfg.BackupDir, "minecraft-network-20260102T000000Z.tar.gz.age"),
		filepath.Join(cfg.BackupDir, "minecraft-network-20260103T000000Z.tar.gz.age"),
	}
	for i, f := range files {
		_ = os.WriteFile(f, []byte("data"), 0600)
		_ = os.WriteFile(f+".sha256", []byte("hash"), 0600)
		modTime := time.Now().Add(time.Duration(i) * time.Hour)
		_ = os.Chtimes(f, modTime, modTime)
	}

	// Retention of 2 should keep the latest 2 and remove the oldest (files[0])
	if err := mgr.Cleanup(2); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	if _, err := os.Stat(files[0]); !os.IsNotExist(err) {
		t.Fatalf("expected oldest backup %s to be deleted", files[0])
	}
	if _, err := os.Stat(files[0] + ".sha256"); !os.IsNotExist(err) {
		t.Fatalf("expected oldest checksum %s.sha256 to be deleted", files[0])
	}
	if _, err := os.Stat(files[1]); err != nil {
		t.Fatalf("expected backup %s to be preserved", files[1])
	}
	if _, err := os.Stat(files[2]); err != nil {
		t.Fatalf("expected backup %s to be preserved", files[2])
	}

	// Retention < 1 defaults to 1
	if err := mgr.Cleanup(0); err != nil {
		t.Fatalf("Cleanup with retention 0 failed: %v", err)
	}
	// Now only files[2] should remain
	if _, err := os.Stat(files[1]); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be deleted with retention 1", files[1])
	}
	if _, err := os.Stat(files[2]); err != nil {
		t.Fatalf("expected latest backup %s to remain", files[2])
	}
}

func TestManager_CreateAndRestore_EndToEnd(t *testing.T) {
	cfg, runner := setupTestEnvironment(t)
	client := compose.NewClient(runner, cfg.ProjectDir, cfg.ComposeFile())
	mgr := backup.NewManager(cfg, client, runner)

	archive, hash, err := mgr.Create(context.Background())
	if err != nil {
		t.Fatalf("Create backup failed: %v", err)
	}
	if archive == "" || hash == "" {
		t.Fatalf("expected non-empty archive and hash, got archive=%s, hash=%s", archive, hash)
	}
	if _, err := os.Stat(archive); err != nil {
		t.Fatalf("archive file does not exist: %v", err)
	}
	if _, err := os.Stat(archive + ".sha256"); err != nil {
		t.Fatalf("sha256 file does not exist: %v", err)
	}

	// Verify checksum in sha256 file
	shaContent, err := os.ReadFile(archive + ".sha256")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(shaContent, []byte(hash)) {
		t.Fatalf("sha256 file mismatch: expected prefix %s, got %s", hash, string(shaContent))
	}

	// Now Restore the created archive
	if err := mgr.Restore(context.Background(), archive); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
}

func TestManager_Create_Errors(t *testing.T) {
	// 1. Lock Contention
	cfg, runner := setupTestEnvironment(t)
	client := compose.NewClient(runner, cfg.ProjectDir, cfg.ComposeFile())
	mgr := backup.NewManager(cfg, client, runner)

	lock, err := config.AcquireLock(cfg.ProjectDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := mgr.Create(context.Background()); err == nil {
		t.Fatal("expected error on lock contention during Create, got nil")
	}
	_ = lock.Close()

	// 2. Secret Validation Failure (e.g. missing secret)
	cfg2, runner2 := setupTestEnvironment(t)
	_ = os.Remove(filepath.Join(cfg2.ProjectDir, "secrets", "postgres_password"))
	client2 := compose.NewClient(runner2, cfg2.ProjectDir, cfg2.ComposeFile())
	mgr2 := backup.NewManager(cfg2, client2, runner2)
	if _, _, err := mgr2.Create(context.Background()); err == nil {
		t.Fatal("expected error on missing secret during Create, got nil")
	}

	// 3. Compose config error
	cfg3, runner3 := setupTestEnvironment(t)
	runner3.runFunc = func(_ context.Context, _ string, _ string, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "compose" && args[len(args)-1] == "--quiet" {
			return "", os.ErrPermission
		}
		return "ok", nil
	}
	client3 := compose.NewClient(runner3, cfg3.ProjectDir, cfg3.ComposeFile())
	mgr3 := backup.NewManager(cfg3, client3, runner3)
	if _, _, err := mgr3.Create(context.Background()); err == nil {
		t.Fatal("expected error on compose validation failure, got nil")
	}

	// 4. Postgres dump failure
	cfg4, runner4 := setupTestEnvironment(t)
	runner4.runStreamFunc = func(_ context.Context, _ string, _, stderr io.Writer, _ string, _ ...string) error {
		if stderr != nil {
			_, _ = stderr.Write([]byte("fatal: pg_dump connection refused"))
		}
		return os.ErrInvalid
	}
	client4 := compose.NewClient(runner4, cfg4.ProjectDir, cfg4.ComposeFile())
	mgr4 := backup.NewManager(cfg4, client4, runner4)
	if _, _, err := mgr4.Create(context.Background()); err == nil {
		t.Fatal("expected error on postgres dump failure, got nil")
	}

	// 5. Environment validation failure
	cfg5, runner5 := setupTestEnvironment(t)
	_ = os.WriteFile(filepath.Join(cfg5.ProjectDir, ".env"), []byte("VELOCITY_IMAGE=latest\n"), 0600)
	client5 := compose.NewClient(runner5, cfg5.ProjectDir, cfg5.ComposeFile())
	mgr5 := backup.NewManager(cfg5, client5, runner5)
	if _, _, err := mgr5.Create(context.Background()); err == nil {
		t.Fatal("expected error on invalid .env during Create, got nil")
	}

	// 6. Compose stop failure during Create
	cfg6, runner6 := setupTestEnvironment(t)
	runner6.runFunc = func(_ context.Context, _ string, _ string, args ...string) (string, error) {
		for _, a := range args {
			if a == "stop" {
				return "", os.ErrPermission
			}
		}
		return "mock-output", nil
	}
	client6 := compose.NewClient(runner6, cfg6.ProjectDir, cfg6.ComposeFile())
	mgr6 := backup.NewManager(cfg6, client6, runner6)
	if _, _, err := mgr6.Create(context.Background()); err == nil {
		t.Fatal("expected error on compose stop failure during Create, got nil")
	}
}

func TestManager_Restore_Errors(t *testing.T) {
	cfg, runner := setupTestEnvironment(t)
	client := compose.NewClient(runner, cfg.ProjectDir, cfg.ComposeFile())
	mgr := backup.NewManager(cfg, client, runner)

	// 1. Lock contention
	lock, err := config.AcquireLock(cfg.ProjectDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.Restore(context.Background(), "any.tar.gz.age"); err == nil {
		t.Fatal("expected error on lock contention during Restore, got nil")
	}
	_ = lock.Close()

	// 2. Non-existent archive
	if err := mgr.Restore(context.Background(), filepath.Join(cfg.BackupDir, "missing.tar.gz.age")); err == nil {
		t.Fatal("expected error on missing archive, got nil")
	}

	// 3. Archive does not end with .tar.gz.age
	badExt := filepath.Join(cfg.BackupDir, "bad.zip")
	_ = os.WriteFile(badExt, []byte("zipdata"), 0600)
	if err := mgr.Restore(context.Background(), badExt); err == nil {
		t.Fatal("expected error for non .tar.gz.age file, got nil")
	}

	// 4. Missing .sha256 file
	archivePath := filepath.Join(cfg.BackupDir, "minecraft-network-test.tar.gz.age")
	_ = os.WriteFile(archivePath, []byte("age-content"), 0600)
	if err := mgr.Restore(context.Background(), archivePath); err == nil {
		t.Fatal("expected error for missing .sha256 file, got nil")
	}

	// 5. Invalid .sha256 format (not 64 chars)
	_ = os.WriteFile(archivePath+".sha256", []byte("invalid-short-hash\n"), 0600)
	if err := mgr.Restore(context.Background(), archivePath); err == nil {
		t.Fatal("expected error for invalid .sha256 format, got nil")
	}

	// 6. Checksum mismatch
	_ = os.WriteFile(archivePath+".sha256", []byte("0000000000000000000000000000000000000000000000000000000000000000  archive.tar.gz.age\n"), 0600)
	if err := mgr.Restore(context.Background(), archivePath); err == nil {
		t.Fatal("expected error for checksum mismatch, got nil")
	}

	// 7. Compose stop failure during restore
	// Create valid backup first
	validArchive, _, err := mgr.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	runnerFailStop := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ string, args ...string) (string, error) {
			for _, a := range args {
				if a == "stop" {
					return "", os.ErrPermission
				}
			}
			return "mock-output", nil
		},
	}
	clientFailStop := compose.NewClient(runnerFailStop, cfg.ProjectDir, cfg.ComposeFile())
	mgrFailStop := backup.NewManager(cfg, clientFailStop, runnerFailStop)
	if err := mgrFailStop.Restore(context.Background(), validArchive); err == nil {
		t.Fatal("expected error on compose stop failure during restore")
	}

	// 8. Postgres start failure during restore
	runnerFailPgStart := &mockRunner{
		runFunc: func(_ context.Context, _ string, _ string, args ...string) (string, error) {
			hasStart, hasPostgres := false, false
			for _, a := range args {
				if a == "start" {
					hasStart = true
				}
				if a == "postgres" {
					hasPostgres = true
				}
			}
			if hasStart && hasPostgres {
				return "", os.ErrPermission
			}
			return "mock-output", nil
		},
	}
	clientFailPgStart := compose.NewClient(runnerFailPgStart, cfg.ProjectDir, cfg.ComposeFile())
	mgrFailPgStart := backup.NewManager(cfg, clientFailPgStart, runnerFailPgStart)
	if err := mgrFailPgStart.Restore(context.Background(), validArchive); err == nil {
		t.Fatal("expected error on postgres start failure during restore")
	}

	// 9. Postgres restore command failure
	runnerFailPgRestore := &mockRunner{
		runInputFunc: func(_ context.Context, _ string, _ string, _ io.Reader, _ ...string) (string, error) {
			return "", os.ErrInvalid
		},
	}
	clientFailPgRestore := compose.NewClient(runnerFailPgRestore, cfg.ProjectDir, cfg.ComposeFile())
	mgrFailPgRestore := backup.NewManager(cfg, clientFailPgRestore, runnerFailPgRestore)
	if err := mgrFailPgRestore.Restore(context.Background(), validArchive); err == nil {
		t.Fatal("expected error on pg_restore failure during restore")
	}
}
