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
			"1h":    "1h",
			"12h":   "12h",
			"24h":   "24h",
			"7d":    "7d",
			"30d":   "30d",
			"5m":    "5m",
			"2w":    "2w",
			"1y":    "1y",
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

	t.Run("validate minecraft username", func(t *testing.T) {
		validNames := []string{"Steve", "Alex", "kizio123", "notch_99", "Player_1234"}
		for _, name := range validNames {
			if err := discord.ValidateMinecraftUsername(name); err != nil {
				t.Errorf("expected valid for %q, got err: %v", name, err)
			}
		}

		invalidNames := []string{
			"",
			"ab",                            // too short (<3)
			"this_name_is_way_too_long_123", // too long (>16)
			"steve\nop",                     // command injection newline
			"alex;rm -rf",                   // command injection semicolon
			"player 1",                      // space
			"user@domain",                   // special char
		}
		for _, name := range invalidNames {
			if err := discord.ValidateMinecraftUsername(name); err == nil {
				t.Errorf("expected error for invalid username %q, got nil", name)
			}
		}
	})

	t.Run("sanitize punish reason", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"Griefing spawn", "Griefing spawn"},
			{"  Spaces around  ", "Spaces around"},
			{"Multiple   spaces\tin\tmiddle", "Multiple spaces in middle"},
			{"Line1\nLine2\rLine3\000Line4", "Line1 Line2 Line3 Line4"},
			{"Control\x01\x1fchars\x7f", "Controlchars"},
			{"", "Violation of server rules"},
			{"   \t\n   ", "Violation of server rules"},
		}

		for _, tc := range tests {
			res := discord.SanitizePunishReason(tc.input)
			if res != tc.expected {
				t.Errorf("expected %q for input %q, got %q", tc.expected, tc.input, res)
			}
		}

		// Test truncation of long reasons
		longInput := ""
		for i := 0; i < 200; i++ {
			longInput += "a"
		}
		truncated := discord.SanitizePunishReason(longInput)
		if len(truncated) != 128 {
			t.Errorf("expected length 128, got %d", len(truncated))
		}
	})
}
