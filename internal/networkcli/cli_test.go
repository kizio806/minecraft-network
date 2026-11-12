package networkcli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
)

type fakeRunner struct {
	calls        [][]string
	output       map[string]string
	err          error
	failContains string
}

func (f *fakeRunner) Run(_ context.Context, _ string, name string, args ...string) (string, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if f.failContains != "" && strings.Contains(strings.Join(append([]string{name}, args...), " "), f.failContains) {
		return "", errors.New("mock command failure")
	}
	if f.err != nil {
		return "", f.err
	}
	return f.output[strings.Join(append([]string{name}, args...), " ")], nil
}
func (f *fakeRunner) RunInput(_ context.Context, _ string, name string, _ io.Reader, args ...string) (string, error) {
	return f.Run(context.Background(), "", name, args...)
}

func testConfig(t *testing.T) Config {
	t.Helper()
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "docker-compose.yml"), []byte("include: []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return Config{ProjectDir: d, Output: "table", BackupDir: filepath.Join(d, "backups"), BackupRetention: 1}
}

func operationalConfig(t *testing.T) Config {
	t.Helper()
	c := testConfig(t)
	if err := os.Mkdir(filepath.Join(c.ProjectDir, "secrets"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"postgres_password", "forwarding_secret", "grafana_admin_password", "luckperms_password"} {
		if err := os.WriteFile(filepath.Join(c.ProjectDir, "secrets", n), []byte("test-secret\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(c.ProjectDir, ".env"), []byte("VELOCITY_IMAGE=x/y@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\nPURPUR_IMAGE=x/y@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\nPURPUR_LOBBY_IMAGE=x/y@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee\nNANOLIMBO_IMAGE=x/y@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc\nMC_MONITOR_IMAGE=x/y@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(c.ProjectDir, "configs/purpur"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.ProjectDir, "configs/purpur/server.properties"), []byte("online-mode=false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.ProjectDir, "configs/purpur/paper-global.yml"), []byte("secret: __FORWARDING_SECRET__\n"), 0600); err != nil {
		t.Fatal(err)
	}
	id, e := age.GenerateX25519Identity()
	if e != nil {
		t.Fatal(e)
	}
	c.BackupRecipientFile = filepath.Join(c.ProjectDir, "recipient")
	c.BackupIdentityFile = filepath.Join(c.ProjectDir, "identity")
	if err := os.WriteFile(c.BackupRecipientFile, []byte(id.Recipient().String()+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.BackupIdentityFile, []byte(id.String()+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestConfigRejectsInvalidOutput(t *testing.T) {
	c := testConfig(t)
	if _, err := Load(c.ProjectDir, "xml"); err == nil {
		t.Fatal("expected invalid output")
	}
	if _, err := Load(c.ProjectDir, ""); err != nil {
		t.Fatal(err)
	}
}

func TestEnvironmentRequiresPinnedImages(t *testing.T) {
	c := testConfig(t)
	if err := os.WriteFile(filepath.Join(c.ProjectDir, ".env"), []byte("VELOCITY_IMAGE=itzg/mc-proxy:java25\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.ValidateEnvironmentFile(); err == nil {
		t.Fatal("expected unpinned image rejection")
	}
}
func TestOperationLockSerializes(t *testing.T) {
	c := testConfig(t)
	one, e := AcquireLock(c.ProjectDir)
	if e != nil {
		t.Fatal(e)
	}
	defer one.Close()
	if _, e = AcquireLock(c.ProjectDir); e == nil {
		t.Fatal("expected lock contention")
	}
}
func TestParseComposeRows(t *testing.T) {
	rows, err := parseComposeRows(`{"Service":"proxy","State":"running","Health":"healthy"}`)
	if err != nil || len(rows) != 1 || rows[0].Service != "proxy" {
		t.Fatalf("unexpected rows: %#v %v", rows, err)
	}
	if _, err = parseComposeRows("not-json"); err == nil {
		t.Fatal("expected malformed status rejection")
	}
}
func TestHealthRejectsMissingStoppedAndMalformedServices(t *testing.T) {
	c := operationalConfig(t)
	var out bytes.Buffer
	ps := "{\"Service\":\"proxy\",\"State\":\"running\",\"Health\":\"healthy\"}\n"
	key := "docker compose -f " + c.ComposeFile() + " ps --format json"
	f := &fakeRunner{output: map[string]string{key: ps}}
	if err := (App{Out: &out, Runner: f}).nativeHealth(context.Background(), c); err == nil {
		t.Fatal("expected missing service failure")
	}
	f.output[key] = "not-json"
	if err := (App{Out: &out, Runner: f}).nativeHealth(context.Background(), c); err == nil {
		t.Fatal("expected malformed status failure")
	}
}
func TestRestoreRequiresConfirmation(t *testing.T) {
	c := testConfig(t)
	var out bytes.Buffer
	err := (App{Out: &out, Runner: &fakeRunner{}}).nativeRestore(context.Background(), c, "missing.tar.gz", false)
	var x *ExitError
	if !errors.As(err, &x) || x.Code != 2 {
		t.Fatalf("unexpected error: %v", err)
	}
}
func TestLifecycleAllowlist(t *testing.T) {
	c := testConfig(t)
	var out bytes.Buffer
	err := (App{Out: &out, Runner: &fakeRunner{}}).lifecycle(context.Background(), c, "start", []string{"../../etc"})
	var x *ExitError
	if !errors.As(err, &x) || x.Code != 2 {
		t.Fatalf("unexpected error: %v", err)
	}
}
func TestLifecycleCommandOrderAndFailure(t *testing.T) {
	c := operationalConfig(t)
	f := &fakeRunner{output: map[string]string{}}
	var out bytes.Buffer
	if err := (App{Out: &out, Runner: f}).lifecycle(context.Background(), c, "start", []string{"proxy"}); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 2 || f.calls[0][len(f.calls[0])-2] != "config" || f.calls[1][len(f.calls[1])-3] != "up" {
		t.Fatalf("unexpected calls: %#v", f.calls)
	}
	f2 := &fakeRunner{err: errors.New("compose failed")}
	if err := (App{Out: &out, Runner: f2}).lifecycle(context.Background(), c, "stop", []string{"proxy"}); err == nil {
		t.Fatal("expected compose failure")
	}
}
func TestLifecycleUsesMockDockerExecutable(t *testing.T) {
	c := operationalConfig(t)
	bin := t.TempDir()
	log := filepath.Join(bin, "calls.log")
	docker := filepath.Join(bin, "docker")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"" + log + "\"\n"
	if err := os.WriteFile(docker, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	old := os.Getenv("PATH")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+old)
	if err := (App{Out: io.Discard, Runner: ExecRunner{}}).lifecycle(context.Background(), c, "start", []string{"proxy"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "config --quiet") || !strings.Contains(lines[1], "up -d proxy") {
		t.Fatalf("unexpected docker calls: %q", string(b))
	}
}
func TestEncryptedBackupAndRestore(t *testing.T) {
	c := operationalConfig(t)
	f := &fakeRunner{output: map[string]string{}}
	var out bytes.Buffer
	if err := (App{Out: &out, Runner: f}).nativeBackup(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(c.BackupDir, "*.age"))
	if len(matches) != 1 {
		t.Fatalf("expected encrypted archive, got %v", matches)
	}
	if err := (App{Out: &out, Runner: f}).nativeRestore(context.Background(), c, matches[0], true); err != nil {
		t.Fatal(err)
	}
}
func TestEncryptedBackupRejectsWrongOrCorruptKey(t *testing.T) {
	c := operationalConfig(t)
	f := &fakeRunner{output: map[string]string{}}
	var out bytes.Buffer
	if err := (App{Out: &out, Runner: f}).nativeBackup(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(c.BackupDir, "*.age"))
	if len(matches) != 1 {
		t.Fatal(matches)
	}
	wrong, e := age.GenerateX25519Identity()
	if e != nil {
		t.Fatal(e)
	}
	if err := os.WriteFile(c.BackupIdentityFile, []byte(wrong.String()+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := (App{Out: &out, Runner: f}).nativeRestore(context.Background(), c, matches[0], true); err == nil {
		t.Fatal("expected invalid identity failure")
	}
	if err := os.WriteFile(c.BackupIdentityFile, []byte("missing\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(matches[0], []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := (App{Out: &out, Runner: f}).nativeRestore(context.Background(), c, matches[0], true); err == nil {
		t.Fatal("expected checksum failure")
	}
}
func TestUpdateBacksUpBeforePull(t *testing.T) {
	c := operationalConfig(t)
	f := &fakeRunner{output: map[string]string{"docker compose -f " + c.ComposeFile() + " ps --format json": "{\"Service\":\"proxy\",\"State\":\"running\",\"Health\":\"healthy\"}\n{\"Service\":\"auth\",\"State\":\"running\",\"Health\":\"healthy\"}\n{\"Service\":\"lobby\",\"State\":\"running\",\"Health\":\"healthy\"}\n{\"Service\":\"survival\",\"State\":\"running\",\"Health\":\"healthy\"}\n"}}
	var out bytes.Buffer
	if err := (App{Out: &out, Runner: f}).nativeUpdate(context.Background(), c, false, true); err != nil {
		t.Fatal(err)
	}
	seenPull, seenBackup := false, false
	for _, call := range f.calls {
		s := strings.Join(call, " ")
		if strings.Contains(s, " pull") {
			seenPull = true
		}
		if strings.Contains(s, " stop proxy auth lobby survival") {
			seenBackup = true
			if seenPull {
				t.Fatal("pull happened before backup")
			}
		}
	}
	if !seenBackup || !seenPull {
		t.Fatalf("missing orchestration calls: %#v", f.calls)
	}
}
func TestUpdatePropagatesPullFailureAfterBackup(t *testing.T) {
	c := operationalConfig(t)
	f := &fakeRunner{output: map[string]string{}, failContains: " pull"}
	var out bytes.Buffer
	if err := (App{Out: &out, Runner: f}).nativeUpdate(context.Background(), c, false, true); err == nil {
		t.Fatal("expected pull failure")
	}
	matches, _ := filepath.Glob(filepath.Join(c.BackupDir, "*.age"))
	if len(matches) != 1 {
		t.Fatalf("backup was not created before pull failure: %v", matches)
	}
}
