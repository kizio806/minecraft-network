package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

type mockTransport struct{}

func (m *mockTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{"id":"mock_app_1","channel_id":"chan_1"}`)),
		Header:     make(http.Header),
	}, nil
}

func TestRun_HealthCheck(t *testing.T) {
	ctx := context.Background()
	var stdout, stderr bytes.Buffer

	code := run(ctx, []string{"-health"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected code 0 for -health, got %d", code)
	}
}

func TestRun_InvalidFlag(t *testing.T) {
	ctx := context.Background()
	var stdout, stderr bytes.Buffer

	code := run(ctx, []string{"-invalid-flag-xyz"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected code 2 for invalid flag, got %d", code)
	}
}

func TestRun_StandbyMode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var stdout, stderr bytes.Buffer
	t.Setenv("DISCORD_BOT_TOKEN", "placeholder_token_change_me")

	code := run(ctx, []string{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected code 0 for standby mode, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("standby mode")) {
		t.Errorf("expected standby log in stdout, got %s", stdout.String())
	}
}

func TestRun_InvalidConfigFile(t *testing.T) {
	ctx := context.Background()
	var stdout, stderr bytes.Buffer
	t.Setenv("DISCORD_BOT_TOKEN", "real_dummy_token_12345")

	code := run(ctx, []string{"-config", "non_existent_file.yml"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1 for invalid config, got %d", code)
	}
}

func TestRun_InvalidTokenConnectionFailure(t *testing.T) {
	ctx := context.Background()
	var stdout, stderr bytes.Buffer

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte("discord:\n  guild-id: ''\n"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DISCORD_BOT_TOKEN", "invalid_dummy_token_999")
	t.Setenv("DATABASE_URL", "")

	code := run(ctx, []string{"-config", cfgPath, "-locales", ""}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1 for failed discord connection, got %d", code)
	}
}

func TestRun_DatabaseUrlFallback(t *testing.T) {
	ctx := context.Background()
	var stdout, stderr bytes.Buffer

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte("database:\n  url: 'postgres://badhost:5432/db?sslmode=disable'\n"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DISCORD_BOT_TOKEN", "invalid_dummy_token_999")

	code := run(ctx, []string{"-config", cfgPath, "-locales", ""}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1 after failing to connect to Discord, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Falling back to in-memory store")) {
		t.Errorf("expected fallback log message, got %s", stdout.String())
	}
}

func TestRun_EnvDatabaseUrl(t *testing.T) {
	ctx := context.Background()
	var stdout, stderr bytes.Buffer

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte("discord:\n  guild-id: ''\n"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DISCORD_BOT_TOKEN", "invalid_dummy_token_999")
	t.Setenv("DATABASE_URL", "postgres://badhost:5432/db?sslmode=disable")

	code := run(ctx, []string{"-config", cfgPath, "-locales", ""}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1 after failing to connect to Discord, got %d", code)
	}
}

func TestRun_SessionCreationError(t *testing.T) {
	origNew := newDiscordSession
	defer func() { newDiscordSession = origNew }()

	newDiscordSession = func(_ string) (*discordgo.Session, error) {
		return nil, errors.New("forced session error")
	}

	ctx := context.Background()
	var stdout, stderr bytes.Buffer

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte("discord:\n  guild-id: ''\n"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DISCORD_BOT_TOKEN", "any_token")
	code := run(ctx, []string{"-config", cfgPath, "-locales", ""}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1 for forced session creation error, got %d", code)
	}
}

func TestRun_FullLifecycle(t *testing.T) {
	origNew := newDiscordSession
	origOpen := openSession
	defer func() {
		newDiscordSession = origNew
		openSession = origOpen
	}()

	newDiscordSession = func(token string) (*discordgo.Session, error) {
		s, err := discordgo.New("Bot " + token)
		if err == nil {
			s.Client.Transport = &mockTransport{}
			s.State.User = &discordgo.User{ID: "bot_123", Username: "NetworkBot"}
		}
		return s, err
	}
	openSession = func(_ *discordgo.Session) error {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yml")
	cfgContent := `
discord:
  guild-id: "guild_1"
  status-channel-id: "status_chan"
  status-update-interval: 1
network:
  name: "Test Network"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DISCORD_BOT_TOKEN", "valid_mock_token")
	t.Setenv("DISCORD_CONFIG", cfgPath)
	var stdout, stderr bytes.Buffer

	code := run(ctx, []string{"-locales", ""}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected code 0 for full lifecycle, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRun_WithRconAndSecretFiles(t *testing.T) {
	origNew := newDiscordSession
	origOpen := openSession
	defer func() {
		newDiscordSession = origNew
		openSession = origOpen
	}()

	newDiscordSession = func(token string) (*discordgo.Session, error) {
		s, err := discordgo.New("Bot " + token)
		if err == nil {
			s.Client.Transport = &mockTransport{}
			s.State.User = &discordgo.User{ID: "bot_123", Username: "NetworkBot"}
		}
		return s, err
	}
	openSession = func(_ *discordgo.Session) error {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yml")
	cfgContent := `
discord:
  guild-id: "guild_1"
  status-channel-id: "status_chan"
  status-update-interval: 1
network:
  name: "Test Network"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DISCORD_BOT_TOKEN", "valid_mock_token")
	t.Setenv("DISCORD_CONFIG", cfgPath)
	t.Setenv("RCON_PASSWORD", "mock_rcon_pass")
	t.Setenv("DATABASE_URL", "")

	_ = os.MkdirAll("secrets", 0750)
	defer os.RemoveAll("secrets")
	_ = os.WriteFile("secrets/postgres_password", []byte("secret_pg"), 0600)
	_ = os.WriteFile("secrets/rcon_password", []byte("secret_rcon"), 0600)

	var stdout, stderr bytes.Buffer
	code := run(ctx, []string{"-locales", ""}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Initialized native NetworkRconExecutor")) {
		t.Errorf("expected native RCON log in stdout, got %s", stdout.String())
	}
}

func TestRun_FallbackAndCustomRconHosts(t *testing.T) {
	origNew := newDiscordSession
	origOpen := openSession
	defer func() {
		newDiscordSession = origNew
		openSession = origOpen
	}()

	newDiscordSession = func(token string) (*discordgo.Session, error) {
		s, err := discordgo.New("Bot " + token)
		if err == nil {
			s.Client.Transport = &mockTransport{}
			s.State.User = &discordgo.User{ID: "bot_123", Username: "NetworkBot"}
		}
		return s, err
	}
	openSession = func(_ *discordgo.Session) error {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yml")
	cfgContent := `
discord:
  guild-id: "guild_1"
  status-channel-id: "status_chan"
  status-update-interval: 1
network:
  name: "Test Network"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DISCORD_BOT_TOKEN", "valid_mock_token")
	t.Setenv("DISCORD_CONFIG", cfgPath)
	t.Setenv("RCON_PASSWORD", "")
	t.Setenv("RCON_LOBBY_HOST", "custom_lobby:25575")
	t.Setenv("RCON_SURVIVAL_HOST", "custom_survival:25575")
	t.Setenv("DATABASE_URL", "")

	_ = os.MkdirAll("secrets", 0750)
	defer os.RemoveAll("secrets")
	_ = os.WriteFile("secrets/postgres_password", []byte("secret_pg"), 0600)
	_ = os.Remove("secrets/rcon_password")

	t.Setenv("POSTGRES_USER", "custom_u")
	t.Setenv("POSTGRES_HOST", "custom_h")
	t.Setenv("POSTGRES_DB", "custom_d")

	var stdout, stderr bytes.Buffer
	code := run(ctx, []string{"-locales", ""}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("No RCON_PASSWORD set. Falling back to DockerConsoleExecutor")) {
		t.Errorf("expected DockerConsoleExecutor fallback log in stdout, got %s", stdout.String())
	}
}

func TestRun_TokenFromSecretFile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = os.MkdirAll("secrets", 0750)
	defer os.RemoveAll("secrets")
	_ = os.WriteFile("secrets/discord_bot_token", []byte("placeholder_token_change_me"), 0600)

	t.Setenv("DISCORD_BOT_TOKEN", "")

	var stdout, stderr bytes.Buffer
	code := run(ctx, []string{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("standby mode")) {
		t.Errorf("expected standby mode log, got %s", stdout.String())
	}
}

func TestMainInvocation(t *testing.T) {
	if os.Getenv("FLAG_TEST_DISCORD_MAIN") == "1" {
		os.Args = []string{"discord-bot", "-health"}
		main()
		return
	}
	// #nosec G204 -- test subprocess execution of test binary
	cmd := exec.Command(os.Args[0], "-test.run=TestMainInvocation")
	cmd.Env = append(os.Environ(), "FLAG_TEST_DISCORD_MAIN=1")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("main failed: %v", err)
	}
}
