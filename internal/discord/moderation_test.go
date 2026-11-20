package discord_test

import (
	"testing"

	"github.com/kizio806/minecraft-network/internal/discord"
)

func TestParsePunishDuration(t *testing.T) {
	t.Run("valid permanent inputs", func(t *testing.T) {
		for _, input := range []string{"p", "perm", "permanent", "PERM", "  permanent  "} {
			dur, err := discord.ParsePunishDuration(input)
			if err != nil || dur != "permanent" {
				t.Errorf("expected permanent for %q, got %q (err: %v)", input, dur, err)
			}
		}
	})

	t.Run("valid time units", func(t *testing.T) {
		tests := map[string]string{
			"1h":   "1h",
			"12h":  "12h",
			"24h":  "24h",
			"7d":   "7d",
			"30d":  "30d",
			"5m":   "5m",
			"2w":   "2w",
			"1y":   "1y",
			" 10h ": "10h",
		}
		for input, expected := range tests {
			dur, err := discord.ParsePunishDuration(input)
			if err != nil || dur != expected {
				t.Errorf("expected %s for %q, got %s (err: %v)", expected, input, dur, err)
			}
		}
	})

	t.Run("invalid inputs", func(t *testing.T) {
		invalids := []string{"", "0h", "-5d", "abc", "10x", "invalid", "100"}
		for _, input := range invalids {
			_, err := discord.ParsePunishDuration(input)
			if err == nil {
				t.Errorf("expected error for invalid duration %q, got nil", input)
			}
		}
	})
}
