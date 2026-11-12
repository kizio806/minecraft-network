package networkcli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/kizio806/minecraft-network/internal/pluginmanager"
)

var managedServices = []string{"proxy", "auth", "lobby", "survival"}

func (a App) nativeInstall(ctx context.Context, c Config) error {
	for _, d := range []string{"secrets", "servers/proxy", "servers/lobby", "servers/auth", "servers/survival"} {
		if err := os.MkdirAll(filepath.Join(c.ProjectDir, d), 0750); err != nil {
			return &ExitError{4, err}
		}
	}
	for _, name := range []string{"postgres_password", "forwarding_secret", "grafana_admin_password", "luckperms_password"} {
		p := filepath.Join(c.ProjectDir, "secrets", name)
		if st, statErr := os.Stat(p); errors.Is(statErr, os.ErrNotExist) || (statErr == nil && st.Size() == 0) {
			flags := os.O_WRONLY | os.O_CREATE
			if statErr == nil {
				flags |= os.O_TRUNC
			} else {
				flags |= os.O_EXCL
			}
			f, e := os.OpenFile(p, flags, 0600)
			if e != nil {
				return &ExitError{4, e}
			}
			secret := make([]byte, 32)
			if _, e = rand.Read(secret); e != nil {
				_ = f.Close()
				return &ExitError{4, e}
			}
			value := base64.RawStdEncoding.EncodeToString(secret) + "\n"
			if name == "forwarding_secret" {
				value = hex.EncodeToString(secret) + "\n"
			}
			if _, e = io.WriteString(f, value); e != nil {
				_ = f.Close()
				return &ExitError{4, e}
			}
			_ = f.Close()
		} else if statErr != nil {
			return &ExitError{4, statErr}
		}
		if err := os.Chmod(p, 0600); err != nil {
			return &ExitError{4, err}
		}
	}
	if err := c.ValidateSecrets(); err != nil {
		return &ExitError{3, err}
	}
	if err := c.ValidateEnvironmentFile(); err != nil {
		return &ExitError{3, err}
	}
	if err := os.Chmod(filepath.Join(c.ProjectDir, ".env"), 0600); err != nil {
		return &ExitError{4, err}
	}
	if _, err := a.compose(ctx, c, "config", "--quiet"); err != nil {
		return &ExitError{3, err}
	}
	pm := pluginmanager.New(c.PluginManagerManifest(), c.PluginManagerLock())
	if err := pm.Sync(ctx, a.Out, true); err != nil {
		return &ExitError{4, fmt.Errorf("plugin synchronization: %w", err)}
	}
	if err := pm.SyncComponents(ctx, a.Out, true); err != nil {
		return &ExitError{4, fmt.Errorf("component synchronization: %w", err)}
	}
	if err := prepareRuntimeConfigs(c); err != nil {
		return &ExitError{4, err}
	}
	fmt.Fprintln(a.Out, "Installation prepared. Review secrets and pinned images before starting.")
	return nil
}

func prepareRuntimeConfigs(c Config) error {
	secret, err := os.ReadFile(filepath.Join(c.ProjectDir, "secrets", "forwarding_secret"))
	if err != nil {
		return err
	}
	secretValue := strings.TrimSpace(string(secret))
	if secretValue == "" || strings.ContainsAny(secretValue, "\r\n") {
		return errors.New("forwarding secret is empty")
	}
	luckpermsSecret, err := os.ReadFile(filepath.Join(c.ProjectDir, "secrets", "luckperms_password"))
	if err != nil {
		return err
	}
	luckpermsValue := strings.TrimSpace(string(luckpermsSecret))
	if luckpermsValue == "" || strings.ContainsAny(luckpermsValue, "\r\n") {
		return errors.New("LuckPerms password is empty")
	}
	for _, server := range []string{"proxy", "lobby", "survival"} {
		for _, name := range []string{"luckperms/config.yml", "LuckPerms/config.yml", "luckperms/luckperms.conf", "LuckPerms/luckperms.conf"} {
			luckpermsConfig := filepath.Join(c.ProjectDir, "servers", server, "plugins", name)
			if data, readErr := os.ReadFile(luckpermsConfig); readErr == nil {
				updated := strings.ReplaceAll(string(data), "__LUCKPERMS_PASSWORD__", luckpermsValue)
				if updated != string(data) && os.WriteFile(luckpermsConfig, []byte(updated), 0600) != nil {
					return fmt.Errorf("cannot render LuckPerms runtime configuration for %s at %s", server, luckpermsConfig)
				}
			} else if !os.IsNotExist(readErr) {
				return readErr
			}
		}
	}
	properties, err := os.ReadFile(filepath.Join(c.ProjectDir, "configs/purpur/server.properties"))
	if err != nil {
		return err
	}
	paper, err := os.ReadFile(filepath.Join(c.ProjectDir, "configs/purpur/paper-global.yml"))
	if err != nil {
		return err
	}
	paperValue := strings.ReplaceAll(string(paper), "__FORWARDING_SECRET__", secretValue)
	for _, server := range []string{"lobby", "survival"} {
		dir := filepath.Join(c.ProjectDir, "servers", server)
		if err = os.MkdirAll(filepath.Join(dir, "config"), 0750); err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(dir, "server.properties"), properties, 0640); err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(dir, "config/paper-global.yml"), []byte(paperValue), 0640); err != nil {
			return err
		}
	}
	return nil
}

func (a App) nativeHealth(ctx context.Context, c Config) error {
	if _, err := a.compose(ctx, c, "config", "--quiet"); err != nil {
		return &ExitError{3, err}
	}
	if err := c.ValidateSecrets(); err != nil {
		return &ExitError{3, err}
	}
	out, err := a.compose(ctx, c, "ps", "--format", "json")
	if err != nil {
		return &ExitError{5, err}
	}
	rows, err := parseComposeRows(out)
	if err != nil {
		return &ExitError{5, err}
	}
	state := map[string]composeRow{}
	for _, row := range rows {
		state[row.Service] = row
	}
	failed := false
	for _, service := range managedServices {
		row, ok := state[service]
		if !ok || row.State != "running" || (row.Health != "" && row.Health != "healthy") {
			failed = true
		}
	}
	if c.Output == "json" {
		if err := json.NewEncoder(a.Out).Encode(map[string]any{"schema_version": 1, "services": rows, "healthy": !failed}); err != nil {
			return &ExitError{5, err}
		}
	} else {
		for _, row := range rows {
			fmt.Fprintf(a.Out, "%-16s %-10s %s\n", row.Service, row.State, row.Health)
		}
	}
	if failed {
		return &ExitError{5, errors.New("one or more required services are not healthy")}
	}
	return nil
}

type composeRow struct {
	Service string `json:"Service"`
	State   string `json:"State"`
	Health  string `json:"Health"`
	Status  string `json:"Status"`
	Name    string `json:"Name"`
}

func parseComposeRows(out string) ([]composeRow, error) {
	rows := []composeRow{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var row composeRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("invalid Compose status JSON: %w", err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (a App) nativeBackup(ctx context.Context, c Config) error {
	lock, err := AcquireLock(c.ProjectDir)
	if err != nil {
		return &ExitError{4, err}
	}
	defer lock.Close()
	if err := c.ValidateSecrets(); err != nil {
		return &ExitError{3, err}
	}
	if err := c.ValidateEnvironmentFile(); err != nil {
		return &ExitError{3, err}
	}
	if _, err = a.compose(ctx, c, "config", "--quiet"); err != nil {
		return &ExitError{3, err}
	}
	if err = os.MkdirAll(c.BackupDir, 0700); err != nil {
		return &ExitError{4, err}
	}
	work, err := os.MkdirTemp("", "minecraft-backup-")
	if err != nil {
		return &ExitError{4, err}
	}
	defer os.RemoveAll(work)
	if _, err = a.compose(ctx, c, append([]string{"stop"}, managedServices...)...); err != nil {
		return &ExitError{5, err}
	}
	defer func() { _, _ = a.compose(context.Background(), c, append([]string{"start"}, managedServices...)...) }()
	if err = os.MkdirAll(filepath.Join(work, "repo"), 0700); err != nil {
		return &ExitError{4, err}
	}
	if container, e := a.compose(ctx, c, "ps", "-q", "postgres"); e == nil && strings.TrimSpace(container) != "" {
		dump, dumpErr := a.compose(ctx, c, "exec", "-T", "postgres", "sh", "-c", `PGPASSWORD="$(cat /run/secrets/postgres_password)" pg_dump --format=custom --no-owner --username="$POSTGRES_USER" "$POSTGRES_DB"`)
		if dumpErr != nil {
			return &ExitError{4, fmt.Errorf("postgres dump failed: %w", dumpErr)}
		}
		if e := os.WriteFile(filepath.Join(work, "postgres.dump"), []byte(dump), 0600); e != nil {
			return &ExitError{4, e}
		}
	}
	plain := filepath.Join(work, "payload.tar.gz")
	archive := filepath.Join(c.BackupDir, "minecraft-network-"+time.Now().UTC().Format("20060102T150405Z")+".tar.gz.age")
	if err = createBackupArchive(c.ProjectDir, work, plain); err != nil {
		return &ExitError{4, err}
	}
	if err = encryptBackup(plain, archive+".tmp", c.BackupRecipientFile); err != nil {
		return &ExitError{4, err}
	}
	if err = os.Rename(archive+".tmp", archive); err != nil {
		return &ExitError{4, err}
	}
	hash, err := fileSHA256(archive)
	if err != nil {
		return &ExitError{4, err}
	}
	if err = os.WriteFile(archive+".sha256", []byte(hash+"  "+filepath.Base(archive)+"\n"), 0600); err != nil {
		return &ExitError{4, err}
	}
	if err = cleanupBackups(c.BackupDir, c.BackupRetention); err != nil {
		return &ExitError{4, err}
	}
	fmt.Fprintf(a.Out, "Backup created: %s\nSHA-256: %s\n", archive, hash)
	return nil
}

func createBackupArchive(root, stage, destination string) error {
	f, err := os.OpenFile(destination+".tmp", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	paths := []string{"configs", "docker", "infrastructure", "servers", "secrets", ".env"}
	for _, rel := range paths {
		if err := addTree(tw, root, rel); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := addPath(tw, filepath.Join(stage, "postgres.dump"), "database/postgres.dump"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(destination+".tmp", destination)
}

func addPath(tw *tar.Writer, path, archiveName string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	h, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	h.Name = archiveName
	if err = tw.WriteHeader(h); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(tw, f)
	return err
}

func addTree(tw *tar.Writer, root, rel string) error {
	path := filepath.Join(root, rel)
	_, err := os.Stat(path)
	if err != nil {
		return err
	}
	return filepath.Walk(path, func(p string, i os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		name, _ := filepath.Rel(root, p)
		if i.IsDir() {
			return nil
		}
		if strings.Contains(name, "/logs/") || strings.Contains(name, "/cache/") {
			return nil
		}
		h, e := tar.FileInfoHeader(i, "")
		if e != nil {
			return e
		}
		h.Name = name
		if e = tw.WriteHeader(h); e != nil {
			return e
		}
		f, e := os.Open(p)
		if e != nil {
			return e
		}
		defer f.Close()
		_, e = io.Copy(tw, f)
		return e
	})
}

func fileSHA256(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), e
}

func secureKeyFile(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("backup key file unavailable: %w", err)
	}
	if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return errors.New("backup key file must be regular and mode 0600")
	}
	return nil
}

func encryptBackup(input, output, recipientPath string) error {
	if err := secureKeyFile(recipientPath); err != nil {
		return err
	}
	b, err := os.ReadFile(recipientPath)
	if err != nil {
		return err
	}
	recipient, err := age.ParseX25519Recipient(strings.TrimSpace(string(b)))
	if err != nil {
		return fmt.Errorf("invalid backup recipient: %w", err)
	}
	in, err := os.Open(input)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	writer, err := age.Encrypt(out, recipient)
	if err != nil {
		out.Close()
		return err
	}
	if _, err = io.Copy(writer, in); err != nil {
		_ = writer.Close()
		_ = out.Close()
		return err
	}
	if err = writer.Close(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func decryptBackup(input, identityPath string) (string, error) {
	if err := secureKeyFile(identityPath); err != nil {
		return "", err
	}
	b, err := os.ReadFile(identityPath)
	if err != nil {
		return "", err
	}
	identities, err := age.ParseIdentities(strings.NewReader(string(b)))
	if err != nil || len(identities) == 0 {
		return "", errors.New("invalid backup identity")
	}
	in, err := os.Open(input)
	if err != nil {
		return "", err
	}
	defer in.Close()
	reader, err := age.Decrypt(in, identities...)
	if err != nil {
		return "", errors.New("backup decryption failed")
	}
	out, err := os.CreateTemp("", "minecraft-backup-plaintext-*.tar.gz")
	if err != nil {
		return "", err
	}
	path := out.Name()
	if _, err = io.Copy(out, reader); err != nil {
		out.Close()
		os.Remove(path)
		return "", err
	}
	if err = out.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

func (a App) nativeRestore(ctx context.Context, c Config, archive string, yes bool) error {
	if !yes {
		return &ExitError{2, errors.New("restore requires --yes")}
	}
	lock, err := AcquireLock(c.ProjectDir)
	if err != nil {
		return &ExitError{4, err}
	}
	defer lock.Close()
	if _, err = os.Stat(archive); err != nil {
		return &ExitError{8, err}
	}
	if !strings.HasSuffix(archive, ".tar.gz.age") {
		return &ExitError{8, errors.New("only encrypted .tar.gz.age backups are supported")}
	}
	side, e := os.ReadFile(archive + ".sha256")
	if e != nil {
		return &ExitError{8, errors.New("backup checksum file is missing")}
	}
	fields := strings.Fields(string(side))
	if len(fields) < 1 || len(fields[0]) != 64 {
		return &ExitError{8, errors.New("backup checksum file is invalid")}
	}
	got, e := fileSHA256(archive)
	if e != nil || got != strings.ToLower(fields[0]) {
		return &ExitError{8, errors.New("backup checksum mismatch")}
	}
	plain := archive
	if strings.HasSuffix(archive, ".age") {
		plain, err = decryptBackup(archive, c.BackupIdentityFile)
		if err != nil {
			return &ExitError{8, err}
		}
		defer os.Remove(plain)
	}
	f, err := os.Open(plain)
	if err != nil {
		return &ExitError{8, err}
	}
	gz, e := gzip.NewReader(f)
	if e != nil {
		f.Close()
		return &ExitError{8, e}
	}
	tr := tar.NewReader(gz)
	temp, e := os.MkdirTemp("", "minecraft-restore-")
	if e != nil {
		return &ExitError{8, e}
	}
	defer os.RemoveAll(temp)
	for {
		h, e := tr.Next()
		if errors.Is(e, io.EOF) {
			break
		}
		if e != nil {
			return &ExitError{8, e}
		}
		if filepath.IsAbs(h.Name) || strings.Contains(filepath.Clean(h.Name), "..") {
			return &ExitError{8, errors.New("unsafe archive path")}
		}
		dest := filepath.Join(temp, h.Name)
		if h.FileInfo().IsDir() {
			continue
		}
		if e = os.MkdirAll(filepath.Dir(dest), 0700); e != nil {
			return &ExitError{8, e}
		}
		out, e := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if e != nil {
			return &ExitError{8, e}
		}
		_, e = io.Copy(out, tr)
		out.Close()
		if e != nil {
			return &ExitError{8, e}
		}
	}
	gz.Close()
	f.Close()
	if _, err = a.compose(ctx, c, append([]string{"stop"}, managedServices...)...); err != nil {
		return &ExitError{5, err}
	}
	defer func() { _, _ = a.compose(context.Background(), c, append([]string{"start"}, managedServices...)...) }()
	for _, rel := range []string{"configs", "docker", "infrastructure", "servers", "secrets", ".env"} {
		src := filepath.Join(temp, rel)
		if _, e := os.Stat(src); e != nil {
			continue
		}
		if e = copyTree(src, filepath.Join(c.ProjectDir, rel)); e != nil {
			return &ExitError{8, e}
		}
	}
	if dump, e := os.Open(filepath.Join(temp, "database", "postgres.dump")); e == nil {
		defer dump.Close()
		if _, e = a.compose(ctx, c, "start", "postgres"); e != nil {
			return &ExitError{8, e}
		}
		if _, e = a.Runner.RunInput(ctx, c.ProjectDir, "docker", dump, "compose", "-f", c.ComposeFile(), "exec", "-T", "postgres", "sh", "-c", `PGPASSWORD="$(cat /run/secrets/postgres_password)" pg_restore --clean --if-exists --no-owner --username="$POSTGRES_USER" --dbname="$POSTGRES_DB"`); e != nil {
			return &ExitError{8, e}
		}
	}
	fmt.Fprintln(a.Out, "Restore completed; run network health before accepting traffic.")
	return nil
}
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, i os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if i.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		in, e := os.Open(p)
		if e != nil {
			return e
		}
		defer in.Close()
		out, e := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if e != nil {
			return e
		}
		_, e = io.Copy(out, in)
		out.Close()
		return e
	})
}

func (a App) nativeUpdate(ctx context.Context, c Config, dry, yes bool) error {
	if !yes && !dry {
		return &ExitError{2, errors.New("update requires --yes or --dry-run")}
	}
	if dry {
		if _, e := a.compose(ctx, c, "config", "--quiet"); e != nil {
			return &ExitError{3, e}
		}
		fmt.Fprintln(a.Out, "DRY-RUN update: pinned Compose images would be pulled and services recreated")
		return nil
	}
	if e := a.nativeBackup(ctx, c); e != nil {
		return e
	}
	if _, e := a.compose(ctx, c, "pull"); e != nil {
		return &ExitError{5, e}
	}
	if _, e := a.compose(ctx, c, "up", "-d", "--remove-orphans"); e != nil {
		return &ExitError{5, e}
	}
	return a.nativeHealth(ctx, c)
}

func (a App) nativePlugin(ctx context.Context, c Config, args []string, dry bool) error {
	if len(args) == 0 {
		return &ExitError{2, errors.New("plugins requires a command")}
	}
	pm := pluginmanager.New(c.PluginManagerManifest(), c.PluginManagerLock())
	switch args[0] {
	case "validate":
		return pm.Validate()
	case "list":
		return pm.List(a.Out)
	case "plan":
		return pm.Plan(a.Out)
	case "verify":
		return pm.VerifyInstalled()
	case "update-check":
		return pm.UpdateCheck(ctx, a.Out)
	case "sync":
		lock, e := AcquireLock(c.ProjectDir)
		if e != nil {
			return &ExitError{4, e}
		}
		defer lock.Close()
		apply := len(args) > 1 && args[1] == "--apply"
		return pm.Sync(ctx, a.Out, apply && !dry)
	case "rollback":
		lock, e := AcquireLock(c.ProjectDir)
		if e != nil {
			return &ExitError{4, e}
		}
		defer lock.Close()
		return pm.Rollback()
	default:
		return &ExitError{2, fmt.Errorf("unknown plugin command %q", args[0])}
	}
}
