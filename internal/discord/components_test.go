package discord_test

import (
	"testing"
	"time"

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

func TestBuildRulesEmbedAndActionRow(t *testing.T) {
	cfg := &discord.Config{
		Network: discord.NetworkSection{
			Domain: "play.example.com",
		},
		Messages: map[string]string{
			"discord_rules_title":      "Network Rules",
			"discord_rules_body":       "1. Be respectful\n2. No cheats",
			"discord_btn_accept_rules": "I Agree",
		},
	}

	embed := discord.BuildRulesEmbed(cfg)
	if embed.Title != "Network Rules" || embed.Color != 0x2ecc71 {
		t.Errorf("unexpected rules embed: %+v", embed)
	}

	row := discord.BuildRulesActionRow(cfg)
	if len(row.Components) != 1 {
		t.Fatalf("expected 1 button in rules row, got %d", len(row.Components))
	}
	btn, ok := row.Components[0].(discordgo.Button)
	if !ok || btn.CustomID != discord.IDBtnAcceptRules || btn.Label != "I Agree" {
		t.Errorf("unexpected button: %+v", row.Components[0])
	}
}

func TestBuildReportEmbedAndActionRow(t *testing.T) {
	cfg := &discord.Config{
		Network: discord.NetworkSection{Domain: "play.example.com"},
		Messages: map[string]string{
			"discord_report_title":           "Report",
			"discord_report_status_pending":  "Pending",
			"discord_report_status_resolved": "Resolved",
			"discord_btn_report_resolve":     "Resolve",
			"discord_btn_report_dismiss":     "Dismiss",
			"discord_btn_report_punish":      "Punish",
		},
	}

	rep := &discord.Report{
		ID:           42,
		ReporterID:   "123456",
		ReporterTag:  "Steve#0001",
		TargetPlayer: "BadActor",
		Reason:       "Killaura",
		Status:       "pending",
		CreatedAt:    time.Now(),
	}

	embed := discord.BuildReportEmbed(cfg, rep)
	if embed.Color != 0xff4e50 {
		t.Errorf("expected danger color for pending report, got %x", embed.Color)
	}

	rep.Status = "resolved"
	rep.ResolvedBy = "Admin"
	resolvedEmbed := discord.BuildReportEmbed(cfg, rep)
	if resolvedEmbed.Color != 0x2ecc71 {
		t.Errorf("expected success color for resolved report, got %x", resolvedEmbed.Color)
	}

	row := discord.BuildReportActionRow(cfg, 42)
	if len(row.Components) != 3 {
		t.Fatalf("expected 3 action buttons, got %d", len(row.Components))
	}
}

func TestBuildPunishModal(t *testing.T) {
	cfg := &discord.Config{
		Messages: map[string]string{
			"discord_modal_punish_title":          "Punish Panel",
			"discord_modal_punish_duration_label": "Duration",
			"discord_modal_punish_reason_label":   "Reason",
		},
	}

	modal := discord.BuildPunishModal(cfg, 10, "Griefer")
	if modal.Title != "Punish Panel" {
		t.Errorf("expected title 'Punish Panel', got %s", modal.Title)
	}
	if len(modal.Components) != 2 {
		t.Fatalf("expected 2 form rows, got %d", len(modal.Components))
	}
}
