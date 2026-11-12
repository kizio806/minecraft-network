package networkcli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Config struct {
	ProjectDir          string
	Compose             string
	BackupDir           string
	BackupRecipientFile string
	BackupIdentityFile  string
	BackupRetention     int
	Output              string
}

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
	backup := os.Getenv("BACKUP_DIR")
	if backup == "" {
		backup = filepath.Join(projectDir, "backups")
	}
	recipient := os.Getenv("BACKUP_AGE_RECIPIENT_FILE")
	if recipient == "" {
		recipient = "/etc/minecraft-network/backup.age-recipient"
	}
	identity := os.Getenv("BACKUP_AGE_IDENTITY_FILE")
	if identity == "" {
		identity = "/etc/minecraft-network/backup.age-identity"
	}
	retention := 1
	if raw := os.Getenv("BACKUP_RETENTION"); raw != "" {
		parsed, e := strconv.Atoi(raw)
		if e != nil || parsed < 1 {
			return Config{}, fmt.Errorf("BACKUP_RETENTION must be a positive integer")
		}
		retention = parsed
	}
	return Config{ProjectDir: projectDir, Compose: "docker compose", BackupDir: backup, BackupRecipientFile: recipient, BackupIdentityFile: identity, BackupRetention: retention, Output: output}, nil
}

func cleanupBackups(dir string, retention int) error {
	if retention < 1 {
		retention = 1
	}
	matches, e := filepath.Glob(filepath.Join(dir, "minecraft-network-*.tar.gz.age"))
	if e != nil {
		return e
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
		if e = os.Remove(p); e != nil {
			return e
		}
		if e = os.Remove(p + ".sha256"); e != nil && !os.IsNotExist(e) {
			return e
		}
	}
	return nil
}

func (c Config) ComposeFile() string {
	return filepath.Join(c.ProjectDir, "docker-compose.yml")
}
func (c Config) PluginManagerManifest() string {
	return filepath.Join(c.ProjectDir, "configs/plugins/manifest.yml")
}
func (c Config) PluginManagerLock() string {
	return filepath.Join(c.ProjectDir, "configs/plugins/lock.yml")
}

func (c Config) ValidateSecrets() error {
	secretDir := filepath.Join(c.ProjectDir, "secrets")
	st, err := os.Stat(secretDir)
	if err != nil || !st.IsDir() || st.Mode().Perm() != 0700 {
		return fmt.Errorf("secret directory must exist with mode 0700")
	}
	for _, name := range []string{"postgres_password", "forwarding_secret", "grafana_admin_password", "luckperms_password"} {
		p := filepath.Join(secretDir, name)
		st, err := os.Stat(p)
		if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || st.Size() == 0 {
			return fmt.Errorf("secret %s is missing, empty or has unsafe permissions", name)
		}
	}
	return nil
}

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
	values := parseEnvValues(string(b))
	digest := regexp.MustCompile(`^[^\s@]+(?:/[^\s@]+)*@sha256:[0-9a-fA-F]{64}$`)
	for _, key := range []string{"VELOCITY_IMAGE", "PURPUR_IMAGE", "PURPUR_LOBBY_IMAGE", "NANOLIMBO_IMAGE", "MC_MONITOR_IMAGE"} {
		if !digest.MatchString(values[key]) {
			return fmt.Errorf("%s must be pinned to a sha256 image digest", key)
		}
	}
	return nil
}

func parseEnvValues(content string) map[string]string {
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
