package discord

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestMemoryStoreAccountLinking(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	t.Run("link new account", func(t *testing.T) {
		err := store.LinkAccount(ctx, "discord_123", "Steve")
		if err != nil {
			t.Fatalf("unexpected link error: %v", err)
		}

		nick, err := store.GetLinkedAccount(ctx, "discord_123")
		if err != nil || nick != "Steve" {
			t.Fatalf("expected Steve, got %s (err: %v)", nick, err)
		}
	})

	t.Run("re-link same account same nick", func(t *testing.T) {
		err := store.LinkAccount(ctx, "discord_123", "steve")
		if err != nil {
			t.Fatalf("re-linking same account should succeed: %v", err)
		}
	})

	t.Run("conflict with different nick", func(t *testing.T) {
		err := store.LinkAccount(ctx, "discord_123", "Alex")
		if err == nil {
			t.Fatalf("expected conflict error, got nil")
		}
	})

	t.Run("get unlinked account", func(t *testing.T) {
		_, err := store.GetLinkedAccount(ctx, "discord_999")
		if err == nil {
			t.Fatalf("expected error for unlinked account, got nil")
		}
	})
}

func TestMemoryStoreWhitelist(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	t.Run("add and remove whitelist", func(t *testing.T) {
		if err := store.AddToWhitelist(ctx, "Player1"); err != nil {
			t.Fatalf("unexpected add error: %v", err)
		}

		if err := store.RemoveFromWhitelist(ctx, "player1"); err != nil {
			t.Fatalf("unexpected remove error: %v", err)
		}
	})
}

func TestMemoryStoreReports(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	t.Run("save and get report", func(t *testing.T) {
		rep := &Report{
			ReporterID:   "user_1",
			ReporterTag:  "User#0001",
			TargetPlayer: "Cheater",
			Reason:       "Fly hacking and killaura",
		}
		if err := store.SaveReport(ctx, rep); err != nil {
			t.Fatalf("save report failed: %v", err)
		}
		if rep.ID == 0 {
			t.Fatalf("expected non-zero ID")
		}

		fetched, err := store.GetReport(ctx, rep.ID)
		if err != nil {
			t.Fatalf("get report failed: %v", err)
		}
		if fetched.TargetPlayer != "Cheater" || fetched.Status != "pending" {
			t.Fatalf("unexpected report state: %+v", fetched)
		}
	})

	t.Run("update report status", func(t *testing.T) {
		err := store.UpdateReportStatus(ctx, 1, "resolved", "Admin#0001")
		if err != nil {
			t.Fatalf("update status failed: %v", err)
		}

		fetched, err := store.GetReport(ctx, 1)
		if err != nil {
			t.Fatalf("get report failed: %v", err)
		}
		if fetched.Status != "resolved" || fetched.ResolvedBy != "Admin#0001" || fetched.ResolvedAt == nil {
			t.Fatalf("unexpected resolved report state: %+v", fetched)
		}
	})

	t.Run("not found report", func(t *testing.T) {
		_, err := store.GetReport(ctx, 9999)
		if err == nil {
			t.Fatalf("expected error for missing report")
		}
		err = store.UpdateReportStatus(ctx, 9999, "resolved", "Admin")
		if err == nil {
			t.Fatalf("expected error for updating missing report")
		}
	})
}

func TestMemoryStorePlayerRank(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	t.Run("default rank for unknown player", func(t *testing.T) {
		rank, err := store.GetPlayerRank(ctx, "UnknownPlayer")
		if err != nil || rank != "default" {
			t.Fatalf("expected default, got %s (err: %v)", rank, err)
		}
	})

	t.Run("custom rank lookup", func(t *testing.T) {
		store.SetPlayerRank("Kizio", "owner")
		rank, err := store.GetPlayerRank(ctx, "kizio")
		if err != nil || rank != "owner" {
			t.Fatalf("expected owner, got %s (err: %v)", rank, err)
		}
	})
}

func TestMemoryStoreConcurrency(_ *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			user := fmt.Sprintf("user_%d", id)
			_ = store.LinkAccount(ctx, user, user)
			_, _ = store.GetLinkedAccount(ctx, user)
			_ = store.SaveReport(ctx, &Report{
				ReporterID:   user,
				TargetPlayer: fmt.Sprintf("cheater_%d", id),
				Reason:       "speed",
			})
			_ = store.AddToWhitelist(ctx, user)
		}(i)
	}
	wg.Wait()
}
