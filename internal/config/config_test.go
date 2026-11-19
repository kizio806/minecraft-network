package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kizio806/minecraft-network/internal/config"
)

func TestConfigLoadAndPaths(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte("services: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}

	// 1. Valid default load with table output
	cfg, err := config.Load(dir, "table")
	if err != nil {
		t.Fatalf("unexpected Load error: %v", err)
	}

	if cfg.ComposeFile() != composePath {
		t.Errorf("expected compose file %s, got %s", composePath, cfg.ComposeFile())
	}
	if cfg.PluginManagerManifest() != filepath.Join(dir, "configs/plugins/manifest.yml") {
		t.Errorf("unexpected manifest path: %s", cfg.PluginManagerManifest())
	}
	if cfg.PluginManagerLock() != filepath.Join(dir, "configs/plugins/lock.yml") {
		t.Errorf("unexpected lock path: %s", cfg.PluginManagerLock())
	}
	if cfg.ComponentManifest() != filepath.Join(dir, "configs/plugins/components.yml") {
		t.Errorf("unexpected component manifest path: %s", cfg.ComponentManifest())
	}
	if cfg.ComponentLock() != filepath.Join(dir, "configs/plugins/components.lock.yml") {
		t.Errorf("unexpected component lock path: %s", cfg.ComponentLock())
	}

	// 2. Missing docker-compose.yml in directory
	emptyDir := t.TempDir()
	if _, err := config.Load(emptyDir, "table"); err == nil || !strings.Contains(err.Error(), "does not contain docker-compose.yml") {
		t.Fatalf("expected missing docker-compose.yml error, got: %v", err)
	}

	// 3. Output formats: json vs invalid
	cfgJSON, err := config.Load(dir, "json")
	if err != nil || cfgJSON.Output != "json" {
		t.Fatalf("expected json output load, got cfg=%+v, err=%v", cfgJSON, err)
	}
	if _, err := config.Load(dir, "xml"); err == nil || !strings.Contains(err.Error(), "output must be table or json") {
		t.Fatalf("expected error for invalid output format 'xml', got: %v", err)
	}

	// 4. Default output from environment variable and empty output
	t.Setenv("NETWORK_OUTPUT", "json")
	cfgEnvOut, err := config.Load(dir, "")
	if err != nil || cfgEnvOut.Output != "json" {
		t.Fatalf("expected json output from NETWORK_OUTPUT env, got: %v", err)
	}
	t.Setenv("NETWORK_OUTPUT", "")
	cfgDefaultOut, err := config.Load(dir, "")
	if err != nil || cfgDefaultOut.Output != "table" {
		t.Fatalf("expected table output default, got: %v", err)
	}

	// 5. Empty projectDir uses NETWORK_PROJECT_DIR or os.Getwd
	t.Setenv("NETWORK_PROJECT_DIR", dir)
	cfgEnvProj, err := config.Load("", "table")
	if err != nil || cfgEnvProj.ProjectDir != dir {
		t.Fatalf("expected load from NETWORK_PROJECT_DIR, got: %v", err)
	}
	t.Setenv("NETWORK_PROJECT_DIR", "")
}

func TestConfigLoad_EnvironmentOverrides(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	_ = os.WriteFile(composePath, []byte("services: {}\n"), 0600)

	// Create project secrets for age recipient/identity fallback discovery
	_ = os.MkdirAll(filepath.Join(dir, "secrets"), 0700)
	_ = os.WriteFile(filepath.Join(dir, "secrets", "backup.age-recipient"), []byte("rec"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "secrets", "backup.age-identity"), []byte("id"), 0600)

	// .env with relative backup dir and retention
	envPath := filepath.Join(dir, ".env")
	envContent := `BACKUP_DIR=custom-backups
BACKUP_RETENTION=5
`
	_ = os.WriteFile(envPath, []byte(envContent), 0600)

	cfg, err := config.Load(dir, "table")
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.BackupDir != filepath.Join(dir, "custom-backups") {
		t.Errorf("expected backup dir %s, got %s", filepath.Join(dir, "custom-backups"), cfg.BackupDir)
	}
	if cfg.BackupRetention != 5 {
		t.Errorf("expected retention 5, got %d", cfg.BackupRetention)
	}
	if cfg.BackupRecipientFile != filepath.Join(dir, "secrets", "backup.age-recipient") {
		t.Errorf("expected project recipient fallback, got: %s", cfg.BackupRecipientFile)
	}
	if cfg.BackupIdentityFile != filepath.Join(dir, "secrets", "backup.age-identity") {
		t.Errorf("expected project identity fallback, got: %s", cfg.BackupIdentityFile)
	}

	// Environment variable overrides
	t.Setenv("BACKUP_DIR", "/var/backups")
	t.Setenv("BACKUP_AGE_RECIPIENT_FILE", "/etc/keys/rec.key")
	t.Setenv("BACKUP_AGE_IDENTITY_FILE", "/etc/keys/id.key")
	t.Setenv("BACKUP_RETENTION", "10")

	cfgEnv, err := config.Load(dir, "table")
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfgEnv.BackupDir != "/var/backups" {
		t.Errorf("expected /var/backups, got %s", cfgEnv.BackupDir)
	}
	if cfgEnv.BackupRecipientFile != "/etc/keys/rec.key" {
		t.Errorf("expected /etc/keys/rec.key, got %s", cfgEnv.BackupRecipientFile)
	}
	if cfgEnv.BackupIdentityFile != "/etc/keys/id.key" {
		t.Errorf("expected /etc/keys/id.key, got %s", cfgEnv.BackupIdentityFile)
	}
	if cfgEnv.BackupRetention != 10 {
		t.Errorf("expected retention 10, got %d", cfgEnv.BackupRetention)
	}

	// Invalid BACKUP_RETENTION
	t.Setenv("BACKUP_RETENTION", "not-a-number")
	if _, err := config.Load(dir, "table"); err == nil || !strings.Contains(err.Error(), "BACKUP_RETENTION must be a positive integer") {
		t.Fatalf("expected error for invalid retention, got: %v", err)
	}

	t.Setenv("BACKUP_RETENTION", "0")
	if _, err := config.Load(dir, "table"); err == nil || !strings.Contains(err.Error(), "BACKUP_RETENTION must be a positive integer") {
		t.Fatalf("expected error for non-positive retention, got: %v", err)
	}
}

func TestValidateSecrets(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{ProjectDir: dir}

	// 1. Missing secrets dir
	if err := cfg.ValidateSecrets(); err == nil || !strings.Contains(err.Error(), "secret directory must exist") {
		t.Fatalf("expected missing secrets directory error, got: %v", err)
	}

	// 2. Insecure secrets dir permissions (e.g. 0755)
	secretsDir := filepath.Join(dir, "secrets")
	_ = os.MkdirAll(secretsDir, 0755)
	if err := cfg.ValidateSecrets(); err == nil || !strings.Contains(err.Error(), "mode 0700") {
		t.Fatalf("expected mode 0700 error, got: %v", err)
	}
	_ = os.Chmod(secretsDir, 0700)

	// 3. Missing individual secrets
	if err := cfg.ValidateSecrets(); err == nil || !strings.Contains(err.Error(), "postgres_password is missing") {
		t.Fatalf("expected postgres_password missing error, got: %v", err)
	}

	// Create all required secrets with 0600
	required := []string{
		"postgres_password",
		"forwarding_secret",
		"grafana_admin_password",
		"luckperms_password",
		"veloauth_password",
		"ultimateteams_password",
		"libertybans_password",
	}
	for _, s := range required {
		_ = os.WriteFile(filepath.Join(secretsDir, s), []byte("valid-secret"), 0600)
	}

	// 4. Valid secrets
	if err := cfg.ValidateSecrets(); err != nil {
		t.Fatalf("expected valid secrets, got: %v", err)
	}

	// 5. Empty secret file
	_ = os.WriteFile(filepath.Join(secretsDir, "postgres_password"), []byte(""), 0600)
	if err := cfg.ValidateSecrets(); err == nil || !strings.Contains(err.Error(), "empty or has unsafe permissions") {
		t.Fatalf("expected error for empty secret, got: %v", err)
	}
	_ = os.WriteFile(filepath.Join(secretsDir, "postgres_password"), []byte("valid"), 0600)

	// 6. Insecure secret permissions 0644
	_ = os.Chmod(filepath.Join(secretsDir, "forwarding_secret"), 0644)
	if err := cfg.ValidateSecrets(); err == nil || !strings.Contains(err.Error(), "unsafe permissions") {
		t.Fatalf("expected error for 0644 permissions, got: %v", err)
	}
}

func TestValidateEnvironmentFile(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{ProjectDir: dir}

	// 1. Missing .env
	if err := cfg.ValidateEnvironmentFile(); err == nil || !strings.Contains(err.Error(), ".env is missing") {
		t.Fatalf("expected missing .env error, got: %v", err)
	}

	envFile := filepath.Join(dir, ".env")
	// 2. Insecure permissions 0644
	_ = os.WriteFile(envFile, []byte("data"), 0644)
	if err := cfg.ValidateEnvironmentFile(); err == nil || !strings.Contains(err.Error(), "must be a regular file with mode 0600") {
		t.Fatalf("expected mode 0600 error, got: %v", err)
	}
	_ = os.Chmod(envFile, 0600)

	// 3. Missing VELOCITY_VERSION
	_ = os.WriteFile(envFile, []byte("FOO=BAR\n"), 0600)
	if err := cfg.ValidateEnvironmentFile(); err == nil || !strings.Contains(err.Error(), "VELOCITY_VERSION must be 4.3.1") {
		t.Fatalf("expected VELOCITY_VERSION mismatch error, got: %v", err)
	}

	// 4. Unpinned image digest
	unpinnedEnv := `VELOCITY_VERSION=4.3.1
VELOCITY_IMAGE=ghcr.io/papermc/velocity:latest
PURPUR_IMAGE=ghcr.io/purpurmc/purpur@sha256:0000000000000000000000000000000000000000000000000000000000000002
PURPUR_LOBBY_IMAGE=ghcr.io/purpurmc/purpur@sha256:0000000000000000000000000000000000000000000000000000000000000003
NANOLIMBO_IMAGE=ghcr.io/nanolimbo/nanolimbo@sha256:0000000000000000000000000000000000000000000000000000000000000004
MC_MONITOR_IMAGE=ghcr.io/mc-monitor/mc-monitor@sha256:0000000000000000000000000000000000000000000000000000000000000005
`
	_ = os.WriteFile(envFile, []byte(unpinnedEnv), 0600)
	if err := cfg.ValidateEnvironmentFile(); err == nil || !strings.Contains(err.Error(), "must be pinned to a sha256 image digest") {
		t.Fatalf("expected unpinned image digest error, got: %v", err)
	}

	// 5. Valid pinned environment
	validEnv := `VELOCITY_VERSION=4.3.1
VELOCITY_IMAGE=ghcr.io/papermc/velocity@sha256:0000000000000000000000000000000000000000000000000000000000000001
PURPUR_IMAGE=ghcr.io/purpurmc/purpur@sha256:0000000000000000000000000000000000000000000000000000000000000002
PURPUR_LOBBY_IMAGE=ghcr.io/purpurmc/purpur@sha256:0000000000000000000000000000000000000000000000000000000000000003
NANOLIMBO_IMAGE=ghcr.io/nanolimbo/nanolimbo@sha256:0000000000000000000000000000000000000000000000000000000000000004
MC_MONITOR_IMAGE=ghcr.io/mc-monitor/mc-monitor@sha256:0000000000000000000000000000000000000000000000000000000000000005
`
	_ = os.WriteFile(envFile, []byte(validEnv), 0600)
	if err := cfg.ValidateEnvironmentFile(); err != nil {
		t.Fatalf("expected valid environment, got: %v", err)
	}
}

func TestParseEnvValues(t *testing.T) {
	content := `
# Comment line
ENV_PARAM_ALPHA=value1
ENV_PARAM_BETA="quoted value"
ENV_PARAM_GAMMA='single quoted'
ENV_PARAM_DELTA=with=equal=signs
`
	m := config.ParseEnvValues(content)
	if m["ENV_PARAM_ALPHA"] != "value1" {
		t.Errorf("expected value1, got %s", m["ENV_PARAM_ALPHA"])
	}
	if m["ENV_PARAM_BETA"] != "quoted value" {
		t.Errorf("expected 'quoted value', got %s", m["ENV_PARAM_BETA"])
	}
	if m["ENV_PARAM_GAMMA"] != "single quoted" {
		t.Errorf("expected 'single quoted', got %s", m["ENV_PARAM_GAMMA"])
	}
	if m["ENV_PARAM_DELTA"] != "with=equal=signs" {
		t.Errorf("expected 'with=equal=signs', got %s", m["ENV_PARAM_DELTA"])
	}
}

func TestConfigLock(t *testing.T) {
	dir := t.TempDir()
	lock1, err := config.AcquireLock(dir)
	if err != nil {
		t.Fatalf("unexpected lock error: %v", err)
	}
	defer lock1.Close()

	_, err = config.AcquireLock(dir)
	if !errors.Is(err, config.ErrLockContention) {
		t.Fatalf("expected ErrLockContention, got %v", err)
	}

	if err := lock1.Close(); err != nil {
		t.Fatalf("unexpected unlock error: %v", err)
	}

	lock2, err := config.AcquireLock(dir)
	if err != nil {
		t.Fatalf("unexpected re-lock error: %v", err)
	}
	_ = lock2.Close()

	// Nil lock close safe
	var nilLock *config.OperationLock
	if err := nilLock.Close(); err != nil {
		t.Fatalf("expected nil close to succeed, got: %v", err)
	}
}
