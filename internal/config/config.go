package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

// ErrLockContention is returned when an exclusive operation lock cannot be acquired.
var ErrLockContention = errors.New("another network lifecycle or backup operation is in progress")

// ExpectedVelocityVersion defines the required pinned Velocity proxy version.
const ExpectedVelocityVersion = "4.3.1"

// Config represents operational configuration paths and runtime settings.
type Config struct {
	ProjectDir          string
	Compose             string
	BackupDir           string
	BackupRecipientFile string
	BackupIdentityFile  string
	BackupRetention     int
	Output              string
}

// Load resolves and validates environment variables, paths, and configurations.
func Load(projectDir, output string) (Config, error) {
	if projectDir == "" {
		projectDir = os.Getenv("NETWORK_PROJECT_DIR")
	}
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return Config{}, err
		}
	}
	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return Config{}, err
	}
	if st, err := os.Stat(filepath.Join(projectDir, "docker-compose.yml")); err != nil || !st.Mode().IsRegular() {
		return Config{}, fmt.Errorf("project directory %s does not contain docker-compose.yml", projectDir)
	}
	if output == "" {
		output = os.Getenv("NETWORK_OUTPUT")
	}
	if output == "" {
		output = "table"
	}
	if output != "table" && output != "json" {
		return Config{}, fmt.Errorf("output must be table or json")
	}
	envValues := map[string]string{}
	if envData, err := os.ReadFile(filepath.Join(projectDir, ".env")); err == nil {
		envValues = ParseEnvValues(string(envData))
	}
	getVal := func(key string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return envValues[key]
	}

	backup := getVal("BACKUP_DIR")
	if backup == "" {
		backup = filepath.Join(projectDir, "backups")
	}
	if !filepath.IsAbs(backup) {
		backup = filepath.Join(projectDir, backup)
	}

	recipient := getVal("BACKUP_AGE_RECIPIENT_FILE")
	if recipient == "" {
		projectRecipient := filepath.Join(projectDir, "secrets/backup.age-recipient")
		if _, err := os.Stat(projectRecipient); err == nil {
			recipient = projectRecipient
		} else {
			recipient = "/etc/minecraft-network/backup.age-recipient"
		}
	}
	if !filepath.IsAbs(recipient) {
		recipient = filepath.Join(projectDir, recipient)
	}

	identity := getVal("BACKUP_AGE_IDENTITY_FILE")
	if identity == "" {
		projectIdentity := filepath.Join(projectDir, "secrets/backup.age-identity")
		if _, err := os.Stat(projectIdentity); err == nil {
			identity = projectIdentity
		} else {
			identity = "/etc/minecraft-network/backup.age-identity"
		}
	}
	if !filepath.IsAbs(identity) {
		identity = filepath.Join(projectDir, identity)
	}

	retention := 1
	if raw := getVal("BACKUP_RETENTION"); raw != "" {
		parsed, e := strconv.Atoi(raw)
		if e != nil || parsed < 1 {
			return Config{}, fmt.Errorf("BACKUP_RETENTION must be a positive integer")
		}
		retention = parsed
	}
	return Config{
		ProjectDir:          projectDir,
		Compose:             "docker compose",
		BackupDir:           backup,
		BackupRecipientFile: recipient,
		BackupIdentityFile:  identity,
		BackupRetention:     retention,
		Output:              output,
	}, nil
}

// ComposeFile returns the absolute path to docker-compose.yml.
func (c Config) ComposeFile() string {
	return filepath.Join(c.ProjectDir, "docker-compose.yml")
}

// PluginManagerManifest returns path to plugins manifest.yml.
func (c Config) PluginManagerManifest() string {
	return filepath.Join(c.ProjectDir, "configs/plugins/manifest.yml")
}

// PluginManagerLock returns path to plugins lock.yml.
func (c Config) PluginManagerLock() string {
	return filepath.Join(c.ProjectDir, "configs/plugins/lock.yml")
}

// ComponentManifest returns path to components.yml.
func (c Config) ComponentManifest() string {
	return filepath.Join(c.ProjectDir, "configs/plugins/components.yml")
}

// ComponentLock returns path to components.lock.yml.
func (c Config) ComponentLock() string {
	return filepath.Join(c.ProjectDir, "configs/plugins/components.lock.yml")
}

// ValidateSecrets ensures the secrets directory and all required secrets exist with 0600 mode.
func (c Config) ValidateSecrets() error {
	secretDir := filepath.Join(c.ProjectDir, "secrets")
	st, err := os.Stat(secretDir)
	if err != nil || !st.IsDir() || st.Mode().Perm() != 0700 {
		return fmt.Errorf("secret directory must exist with mode 0700")
	}
	for _, name := range []string{"postgres_password", "forwarding_secret", "grafana_admin_password", "luckperms_password", "veloauth_password", "ultimateteams_password", "libertybans_password"} {
		p := filepath.Join(secretDir, name)
		st, err := os.Stat(p)
		permissionsOK := err == nil && st.Mode().Perm() == 0600
		group472 := false
		if err == nil {
			stat, ok := st.Sys().(*syscall.Stat_t)
			if ok {
				group472 = stat.Gid == 472
			}
		}
		if err == nil && name == "grafana_admin_password" && st.Mode().Perm() == 0640 && group472 {
			permissionsOK = true
		}
		if err != nil || !st.Mode().IsRegular() || !permissionsOK || st.Size() == 0 {
			return fmt.Errorf("secret %s is missing, empty or has unsafe permissions", name)
		}
	}
	return nil
}

// ValidateEnvironmentFile verifies the .env file has mode 0600 and pinned digests.
func (c Config) ValidateEnvironmentFile() error {
	p := filepath.Join(c.ProjectDir, ".env")
	st, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf(".env is missing; copy .env.example and pin image digests")
	}
	if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return fmt.Errorf(".env must be a regular file with mode 0600")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	values := ParseEnvValues(string(b))
	if values["VELOCITY_VERSION"] != ExpectedVelocityVersion {
		return fmt.Errorf("VELOCITY_VERSION must be %s", ExpectedVelocityVersion)
	}
	digest := regexp.MustCompile(`^[^\s@]+(?:/[^\s@]+)*@sha256:[0-9a-fA-F]{64}$`)
	for _, key := range []string{"VELOCITY_IMAGE", "PURPUR_IMAGE", "PURPUR_LOBBY_IMAGE", "NANOLIMBO_IMAGE", "MC_MONITOR_IMAGE"} {
		if !digest.MatchString(values[key]) {
			return fmt.Errorf("%s must be pinned to a sha256 image digest", key)
		}
	}
	return nil
}

// ParseEnvValues extracts non-comment key-value pairs from .env content.
func ParseEnvValues(content string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			values[parts[0]] = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		}
	}
	return values
}

// OperationLock manages exclusive flock file locking for operations.
type OperationLock struct{ file *os.File }

// AcquireLock attempts to obtain an exclusive flock lock on .network-operation.lock.
func AcquireLock(dir string) (*OperationLock, error) {
	path := filepath.Join(dir, ".network-operation.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open operation lock: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, ErrLockContention
	}
	return &OperationLock{file: f}, nil
}

// Close releases the lock and closes the file descriptor.
func (l *OperationLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	errUn := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	errClose := l.file.Close()
	if errUn != nil {
		return errUn
	}
	return errClose
}
