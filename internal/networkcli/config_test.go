package networkcli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigLoadErrors(t *testing.T) {
	dir := t.TempDir()

	// Missing docker-compose.yml
	if _, err := Load(dir, "table"); err == nil {
		t.Fatal("expected error when docker-compose.yml is missing")
	}

	// Create dummy docker-compose.yml
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("version: '3.8'"), 0600); err != nil {
		t.Fatal(err)
	}

	// Invalid output format
	if _, err := Load(dir, "yaml"); err == nil {
		t.Fatal("expected error for invalid output format")
	}

	// Invalid BACKUP_RETENTION in .env
	envContent := []byte("BACKUP_RETENTION=invalid\n")
	if err := os.WriteFile(filepath.Join(dir, ".env"), envContent, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir, "table"); err == nil {
		t.Fatal("expected error for non-integer BACKUP_RETENTION")
	}

	// Zero BACKUP_RETENTION
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("BACKUP_RETENTION=0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir, "table"); err == nil {
		t.Fatal("expected error for zero BACKUP_RETENTION")
	}
}

func TestConfigLoadSuccess(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("version: '3.8'"), 0600); err != nil {
		t.Fatal(err)
	}

	secretsDir := filepath.Join(dir, "secrets")
	if err := os.MkdirAll(secretsDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secretsDir, "backup.age-recipient"), []byte("recipient"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secretsDir, "backup.age-identity"), []byte("identity"), 0600); err != nil {
		t.Fatal(err)
	}

	envContent := []byte("BACKUP_DIR=custom_backups\nBACKUP_RETENTION=5\n")
	if err := os.WriteFile(filepath.Join(dir, ".env"), envContent, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir, "json")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Output != "json" {
		t.Errorf("expected json output, got %s", cfg.Output)
	}
	if cfg.BackupRetention != 5 {
		t.Errorf("expected retention 5, got %d", cfg.BackupRetention)
	}
	if cfg.BackupDir != filepath.Join(dir, "custom_backups") {
		t.Errorf("expected relative backup dir resolved to %s, got %s", filepath.Join(dir, "custom_backups"), cfg.BackupDir)
	}
	if cfg.BackupRecipientFile != filepath.Join(secretsDir, "backup.age-recipient") {
		t.Errorf("expected local recipient file, got %s", cfg.BackupRecipientFile)
	}
	if cfg.BackupIdentityFile != filepath.Join(secretsDir, "backup.age-identity") {
		t.Errorf("expected local identity file, got %s", cfg.BackupIdentityFile)
	}
	if cfg.ComposeFile() != filepath.Join(dir, "docker-compose.yml") {
		t.Errorf("unexpected compose file: %s", cfg.ComposeFile())
	}
	if cfg.PluginManagerManifest() != filepath.Join(dir, "configs/plugins/manifest.yml") {
		t.Errorf("unexpected manifest path: %s", cfg.PluginManagerManifest())
	}
	if cfg.PluginManagerLock() != filepath.Join(dir, "configs/plugins/lock.yml") {
		t.Errorf("unexpected lock path: %s", cfg.PluginManagerLock())
	}
}

func TestValidateSecrets(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{ProjectDir: dir}

	// Secret dir does not exist
	if err := cfg.ValidateSecrets(); err == nil {
		t.Fatal("expected error when secrets directory is missing")
	}

	secretDir := filepath.Join(dir, "secrets")
	// Secret dir bad permissions
	if err := os.MkdirAll(secretDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := cfg.ValidateSecrets(); err == nil {
		t.Fatal("expected error when secrets directory is not 0700")
	}

	// Fix directory permissions
	if err := os.Chmod(secretDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Missing secret files
	if err := cfg.ValidateSecrets(); err == nil {
		t.Fatal("expected error when secret files are missing")
	}

	requiredSecrets := []string{
		"postgres_password",
		"forwarding_secret",
		"grafana_admin_password",
		"luckperms_password",
		"veloauth_password",
		"ultimateteams_password",
		"libertybans_password",
	}

	// Create empty secrets with 0600
	for _, s := range requiredSecrets {
		if err := os.WriteFile(filepath.Join(secretDir, s), []byte(""), 0600); err != nil {
			t.Fatal(err)
		}
	}

	// Fails because secrets are empty
	if err := cfg.ValidateSecrets(); err == nil {
		t.Fatal("expected error for empty secret file")
	}

	// Populate non-empty content
	for _, s := range requiredSecrets {
		if err := os.WriteFile(filepath.Join(secretDir, s), []byte("secret_value_123"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	// Should pass
	if err := cfg.ValidateSecrets(); err != nil {
		t.Fatalf("ValidateSecrets failed unexpectedly: %v", err)
	}

	// Unsafe permissions on one secret (0644)
	if err := os.Chmod(filepath.Join(secretDir, "postgres_password"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := cfg.ValidateSecrets(); err == nil {
		t.Fatal("expected error for secret with 0644 permissions")
	}
}

func TestValidateEnvironmentFile(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{ProjectDir: dir}

	// Missing .env
	if err := cfg.ValidateEnvironmentFile(); err == nil {
		t.Fatal("expected error when .env is missing")
	}

	envPath := filepath.Join(dir, ".env")
	// Wrong permissions (0644)
	if err := os.WriteFile(envPath, []byte("VELOCITY_VERSION=4.3.1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := cfg.ValidateEnvironmentFile(); err == nil {
		t.Fatal("expected error when .env is not mode 0600")
	}

	// Fix permissions to 0600
	if err := os.Chmod(envPath, 0600); err != nil {
		t.Fatal(err)
	}

	// Invalid velocity version
	invalidVelocity := []byte("VELOCITY_VERSION=3.0.0\n")
	if err := os.WriteFile(envPath, invalidVelocity, 0600); err != nil {
		t.Fatal(err)
	}
	if err := cfg.ValidateEnvironmentFile(); err == nil {
		t.Fatal("expected error for invalid VELOCITY_VERSION")
	}

	// Missing / unpinned images
	validBase := "VELOCITY_VERSION=4.3.1\n" +
		"VELOCITY_IMAGE=ghcr.io/kizio806/velocity:latest\n"
	if err := os.WriteFile(envPath, []byte(validBase), 0600); err != nil {
		t.Fatal(err)
	}
	if err := cfg.ValidateEnvironmentFile(); err == nil {
		t.Fatal("expected error for unpinned VELOCITY_IMAGE")
	}

	// Complete pinned .env
	pinnedEnv := "VELOCITY_VERSION=4.3.1\n" +
		"VELOCITY_IMAGE=ghcr.io/velocity@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n" +
		"PURPUR_IMAGE=ghcr.io/purpur@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n" +
		"PURPUR_LOBBY_IMAGE=ghcr.io/purpur-lobby@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n" +
		"NANOLIMBO_IMAGE=ghcr.io/nanolimbo@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n" +
		"MC_MONITOR_IMAGE=ghcr.io/mc-monitor@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n"

	if err := os.WriteFile(envPath, []byte(pinnedEnv), 0600); err != nil {
		t.Fatal(err)
	}
	if err := cfg.ValidateEnvironmentFile(); err != nil {
		t.Fatalf("ValidateEnvironmentFile failed on pinned config: %v", err)
	}
}

func TestCleanupBackups(t *testing.T) {
	dir := t.TempDir()

	// Create 3 backup files and sidecars with spaced mod times
	b1 := filepath.Join(dir, "minecraft-network-2026-01-01T00-00-00.tar.gz.age")
	b2 := filepath.Join(dir, "minecraft-network-2026-01-02T00-00-00.tar.gz.age")
	b3 := filepath.Join(dir, "minecraft-network-2026-01-03T00-00-00.tar.gz.age")

	now := time.Now()
	for i, f := range []string{b1, b2, b3} {
		if err := os.WriteFile(f, []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f+".sha256", []byte("hash"), 0600); err != nil {
			t.Fatal(err)
		}
		mt := now.Add(time.Duration(i) * time.Hour)
		_ = os.Chtimes(f, mt, mt)
	}

	// Retention 2 -> oldest (b1) should be deleted
	if err := cleanupBackups(dir, 2); err != nil {
		t.Fatalf("cleanupBackups failed: %v", err)
	}

	if _, err := os.Stat(b1); !os.IsNotExist(err) {
		t.Errorf("expected b1 to be deleted")
	}
	if _, err := os.Stat(b1 + ".sha256"); !os.IsNotExist(err) {
		t.Errorf("expected b1.sha256 to be deleted")
	}
	if _, err := os.Stat(b2); err != nil {
		t.Errorf("expected b2 to exist: %v", err)
	}
	if _, err := os.Stat(b3); err != nil {
		t.Errorf("expected b3 to exist: %v", err)
	}
}
