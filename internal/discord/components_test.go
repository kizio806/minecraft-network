package discord_test

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/kizio806/minecraft-network/internal/discord"
)

func TestBuildStatusActionRow(t *testing.T) {
	cfg := &discord.Config{
		Messages: map[string]string{
			"discord_btn_link":        "Link Account Now",
			"discord_btn_how_to_join": "How to Join Us",
			"discord_btn_refresh":     "Refresh Data",
		},
	}

	row := discord.BuildStatusActionRow(cfg)
	if len(row.Components) != 3 {
		t.Fatalf("expected 3 buttons, got %d", len(row.Components))
	}

	btn1, ok := row.Components[0].(discordgo.Button)
	if !ok || btn1.CustomID != discord.IDBtnLinkAccount || btn1.Label != "Link Account Now" {
		t.Errorf("unexpected button 1: %+v", row.Components[0])
	}

	btn2, ok := row.Components[1].(discordgo.Button)
	if !ok || btn2.CustomID != discord.IDBtnHowToJoin || btn2.Label != "How to Join Us" {
		t.Errorf("unexpected button 2: %+v", row.Components[1])
	}

	btn3, ok := row.Components[2].(discordgo.Button)
	if !ok || btn3.CustomID != discord.IDBtnRefreshStatus || btn3.Label != "Refresh Data" {
		t.Errorf("unexpected button 3: %+v", row.Components[2])
	}
}

func TestBuildLinkAccountModal(t *testing.T) {
	cfg := &discord.Config{
		Messages: map[string]string{
			"discord_modal_title":            "Account Verification",
			"discord_modal_nick_label":       "Nickname",
			"discord_modal_nick_placeholder": "Steve",
		},
	}

	modal := discord.BuildLinkAccountModal(cfg)
	if modal.CustomID != discord.IDModalLinkAccount {
		t.Errorf("expected custom ID %s, got %s", discord.IDModalLinkAccount, modal.CustomID)
	}
	if modal.Title != "Account Verification" {
		t.Errorf("expected title 'Account Verification', got %s", modal.Title)
	}
	if len(modal.Components) != 1 {
		t.Fatalf("expected 1 action row in modal, got %d", len(modal.Components))
	}
}

func TestBuildHowToJoinEmbed(t *testing.T) {
	cfg := &discord.Config{
		Network: discord.NetworkSection{
			Name:   "Minecraft Network",
			Domain: "mc.example.com",
		},
		Messages: map[string]string{
			"discord_how_to_join_title": "How To Play",
			"discord_how_to_join_text":  "Connect to mc.example.com",
		},
	}

	embed := discord.BuildHowToJoinEmbed(cfg)
	if embed.Title != "How To Play" {
		t.Errorf("expected title 'How To Play', got %s", embed.Title)
	}
	if embed.Description != "Connect to mc.example.com" {
		t.Errorf("expected description 'Connect to mc.example.com', got %s", embed.Description)
	}
	if embed.Footer == nil || embed.Footer.Text != "mc.example.com" {
		t.Errorf("unexpected footer: %+v", embed.Footer)
	}
}
