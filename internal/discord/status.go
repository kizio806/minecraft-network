package discord

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ServerStatus represents the live status of an individual server.
type ServerStatus struct {
	Name    string
	Online  bool
	Players int
	Max     int
	PingMs  int64
}

// BuildStatusEmbed builds the rich embed representing network status.
func BuildStatusEmbed(cfg *Config, statuses []ServerStatus) *discordgo.MessageEmbed {
	title := cfg.Msg("discord_status_title", fmt.Sprintf("✦ %s • SERVER STATUS ✦", cfg.Network.Name))
	desc := cfg.Msg("discord_status_desc", "Real-time network status and connection dashboard.")

	onlineLabel := cfg.Msg("discord_status_online", "🟢 Online")
	offlineLabel := cfg.Msg("discord_status_offline", "🔴 Offline")

	totalPlayers := 0
	var serverLines []string
	for _, s := range statuses {
		statusIcon := onlineLabel
		if !s.Online {
			statusIcon = offlineLabel
		} else {
			totalPlayers += s.Players
		}
		serverLines = append(serverLines, fmt.Sprintf("**%s**: %s `%d/%d`", s.Name, statusIcon, s.Players, s.Max))
	}

	playersStr := fmt.Sprintf(cfg.Msg("discord_status_players", "Online Players: **%d / %d**"), totalPlayers, cfg.Network.MaxPlayers)

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "🌐 Network",
			Value:  fmt.Sprintf("`%s`\n%s", cfg.Network.Domain, playersStr),
			Inline: false,
		},
		{
			Name:   "🖥️ Servers",
			Value:  strings.Join(serverLines, "\n"),
			Inline: false,
		},
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: desc,
		Color:       0x00d2ff, // Network vibrant blue
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%s • Last updated: %s", cfg.Network.Domain, time.Now().UTC().Format("15:04:05 UTC")),
		},
	}
}
