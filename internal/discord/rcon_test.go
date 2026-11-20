package discord_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kizio806/minecraft-network/internal/discord"
)

func TestDefaultConsoleExecutor(t *testing.T) {
	ctx := context.Background()

	t.Run("valid target servers", func(t *testing.T) {
		exec := &discord.DefaultConsoleExecutor{}
		for _, srv := range []string{"proxy", "lobby", "survival"} {
			out, err := exec.Execute(ctx, srv, "list")
			if err != nil || out == "" {
				t.Fatalf("unexpected failure for server %s: out=%s err=%v", srv, out, err)
			}
		}
	})

	t.Run("invalid target server", func(t *testing.T) {
		exec := &discord.DefaultConsoleExecutor{}
		_, err := exec.Execute(ctx, "nonexistent", "list")
		if err == nil {
			t.Fatalf("expected error for invalid server, got nil")
		}
	})

	t.Run("mock error output", func(t *testing.T) {
		exec := &discord.DefaultConsoleExecutor{
			MockErr: errors.New("connection refused"),
		}
		_, err := exec.Execute(ctx, "lobby", "say hello")
		if err == nil || err.Error() != "connection refused" {
			t.Fatalf("expected connection refused error, got: %v", err)
		}
	})

	t.Run("custom mock output", func(t *testing.T) {
		exec := &discord.DefaultConsoleExecutor{
			MockOutput: "There are 4 players online",
		}
		out, err := exec.Execute(ctx, "proxy", "glist")
		if err != nil || out != "There are 4 players online" {
			t.Fatalf("unexpected mock output: %s (err: %v)", out, err)
		}
	})
}
