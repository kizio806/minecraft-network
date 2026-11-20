package discord_test

import (
	"context"
	"testing"

	"github.com/kizio806/minecraft-network/internal/discord"
)

func TestBot_SlashCommands(t *testing.T) {
	cfg := &discord.Config{}
	bot := discord.NewBot(cfg, nil, nil, nil)

	cmds := bot.SlashCommands()
	if len(cmds) != 4 {
		t.Fatalf("expected 4 slash commands, got %d", len(cmds))
	}

	expectedNames := map[string]bool{
		"status":    true,
		"players":   true,
		"link":      true,
		"whitelist": true,
	}

	for _, cmd := range cmds {
		if !expectedNames[cmd.Name] {
			t.Errorf("unexpected command name %s", cmd.Name)
		}
	}
}

func TestDefaultQuerier(t *testing.T) {
	querier := &discord.DefaultQuerier{}
	targets := []discord.ServerTarget{
		{Name: "Proxy", Host: "127.0.0.1", Port: 25565},
		{Name: "Survival", Host: "127.0.0.1", Port: 25566},
	}

	statuses := querier.QueryStatuses(context.Background(), targets)
	if len(statuses) != 2 {
		t.Fatalf("expected 2 server statuses, got %d", len(statuses))
	}

	if statuses[0].Name != "Proxy" || !statuses[0].Online {
		t.Errorf("unexpected status: %+v", statuses[0])
	}
}

func TestDefaultLinker(t *testing.T) {
	linker := &discord.DefaultLinker{}
	ctx := context.Background()

	err := linker.LinkAccount(ctx, "discord_123", "Steve")
	if err != nil {
		t.Fatalf("unexpected link error: %v", err)
	}

	nick, err := linker.GetLinkedAccount(ctx, "discord_123")
	if err != nil || nick != "Steve" {
		t.Errorf("expected Steve, got %s (err: %v)", nick, err)
	}

	if err := linker.LinkAccount(ctx, "discord_123", "Steve"); err != nil {
		t.Errorf("expected idempotent link, got %v", err)
	}

	if err := linker.LinkAccount(ctx, "discord_123", "Alex"); err == nil {
		t.Error("expected error linking to different username, got nil")
	}

	if _, err := linker.GetLinkedAccount(ctx, "discord_999"); err == nil {
		t.Error("expected error for unlinked account, got nil")
	}

	if err := linker.AddToWhitelist(ctx, "Steve"); err != nil {
		t.Errorf("unexpected whitelist add error: %v", err)
	}
	if err := linker.RemoveFromWhitelist(ctx, "Steve"); err != nil {
		t.Errorf("unexpected whitelist remove error: %v", err)
	}
}

func TestBot_StatusLoopNoChannel(_ *testing.T) {
	cfg := &discord.Config{}
	bot := discord.NewBot(cfg, nil, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bot.StartStatusLoop(ctx)
}
