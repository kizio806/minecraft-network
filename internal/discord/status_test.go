package discord_test

import (
	"strings"
	"testing"

	"github.com/kizio806/minecraft-network/internal/discord"
)

func TestBuildStatusEmbed(t *testing.T) {
	cfg := &discord.Config{
		Network: discord.NetworkSection{
			Name:       "Test SMP",
			Domain:     "play.testsmp.org",
			MaxPlayers: 150,
		},
		Messages: map[string]string{
			"discord_status_title":   "✦ Network Status ✦",
			"discord_status_desc":    "Live server telemetry",
			"discord_status_online":  "ONLINE",
			"discord_status_offline": "OFFLINE",
			"discord_status_players": "Total: **%d / %d**",
		},
	}

	statuses := []discord.ServerStatus{
		{Name: "Proxy", Online: true, Players: 25, Max: 150, PingMs: 4},
		{Name: "Lobby", Online: true, Players: 5, Max: 50, PingMs: 2},
		{Name: "Survival", Online: false, Players: 0, Max: 100, PingMs: 0},
	}

	embed := discord.BuildStatusEmbed(cfg, statuses)
	if embed.Title != "✦ Network Status ✦" {
		t.Errorf("expected title '✦ Network Status ✦', got %s", embed.Title)
	}
	if embed.Description != "Live server telemetry" {
		t.Errorf("expected description 'Live server telemetry', got %s", embed.Description)
	}
	if len(embed.Fields) != 2 {
		t.Fatalf("expected 2 fields in embed, got %d", len(embed.Fields))
	}

	serversField := embed.Fields[1]
	if !strings.Contains(serversField.Value, "**Proxy**: ONLINE") {
		t.Errorf("expected Proxy to be ONLINE, got: %s", serversField.Value)
	}
	if !strings.Contains(serversField.Value, "**Survival**: OFFLINE") {
		t.Errorf("expected Survival to be OFFLINE, got: %s", serversField.Value)
	}
}
