package discord_test

import (
	"context"
	"testing"

	"github.com/kizio806/minecraft-network/internal/discord"
)

func TestStore_PlayerRankSync(t *testing.T) {
	ctx := context.Background()
	store := discord.NewMemoryStore()

	t.Run("unlinked account check", func(t *testing.T) {
		_, err := store.GetLinkedAccount(ctx, "unknown_discord_id")
		if err == nil {
			t.Fatalf("expected error for unlinked account, got nil")
		}
	})

	t.Run("linked account rank resolution", func(t *testing.T) {
		if err := store.LinkAccount(ctx, "discord_admin", "AdminPlayer"); err != nil {
			t.Fatalf("link failed: %v", err)
		}
		store.SetPlayerRank("AdminPlayer", "administrator")

		nick, err := store.GetLinkedAccount(ctx, "discord_admin")
		if err != nil || nick != "AdminPlayer" {
			t.Fatalf("expected AdminPlayer, got %s (err: %v)", nick, err)
		}

		rank, err := store.GetPlayerRank(ctx, nick)
		if err != nil || rank != "administrator" {
			t.Fatalf("expected administrator, got %s (err: %v)", rank, err)
		}
	})

	t.Run("default rank for normal player", func(t *testing.T) {
		if err := store.LinkAccount(ctx, "discord_player", "NormalPlayer"); err != nil {
			t.Fatalf("link failed: %v", err)
		}
		rank, err := store.GetPlayerRank(ctx, "NormalPlayer")
		if err != nil || rank != "default" {
			t.Fatalf("expected default rank, got %s (err: %v)", rank, err)
		}
	})
}
