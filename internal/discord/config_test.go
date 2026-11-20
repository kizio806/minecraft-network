package discord_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kizio806/minecraft-network/internal/discord"
)

func TestLoadConfig_Success(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "config.yml")
	content := `
discord:
  guild-id: "123456789"
  status-channel-id: "987654321"
  status-update-interval: 45
  roles:
    linked-role-id: "111222333"

network:
  name: "Test Network"
  domain: "mc.test.net"
  max-players: 100

servers:
  - name: "Proxy"
    host: "proxy"
    port: 25565
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := discord.LoadConfig(configPath, "", "en")
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.Discord.GuildID != "123456789" {
		t.Errorf("expected guild-id 123456789, got %s", cfg.Discord.GuildID)
	}
	if cfg.Discord.StatusUpdateInterval != 45 {
		t.Errorf("expected interval 45, got %d", cfg.Discord.StatusUpdateInterval)
	}
	if cfg.Network.MaxPlayers != 100 {
		t.Errorf("expected max-players 100, got %d", cfg.Network.MaxPlayers)
	}
	if len(cfg.Servers) != 1 || cfg.Servers[0].Name != "Proxy" {
		t.Errorf("expected 1 server named Proxy, got %+v", cfg.Servers)
	}
}

func TestLoadConfig_DefaultsAndMissingFile(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "minimal.yml")
	content := `
discord:
  guild-id: "123"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := discord.LoadConfig(configPath, "", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Discord.StatusUpdateInterval != 30 {
		t.Errorf("expected default interval 30, got %d", cfg.Discord.StatusUpdateInterval)
	}
	if cfg.Network.MaxPlayers != 50 {
		t.Errorf("expected default max players 50, got %d", cfg.Network.MaxPlayers)
	}

	// Missing file should return error
	if _, err := discord.LoadConfig(filepath.Join(temp, "nonexistent.yml"), "", "en"); err == nil {
		t.Error("expected error for missing config file, got nil")
	}
}

func TestConfig_Msg_Fallback(t *testing.T) {
	cfg := &discord.Config{
		Messages: map[string]string{
			"known_key": "Custom Value",
		},
	}

	if val := cfg.Msg("known_key", "Default"); val != "Custom Value" {
		t.Errorf("expected 'Custom Value', got '%s'", val)
	}
	if val := cfg.Msg("unknown_key", "Default Value"); val != "Default Value" {
		t.Errorf("expected fallback 'Default Value', got '%s'", val)
	}
}
