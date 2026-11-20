package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// HandleSyncCommand synchronizes the player's in-game LuckPerms rank with Discord roles.
func HandleSyncCommand(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *Config, store Store) {
	if i.Member == nil || i.GuildID == "" {
		return
	}

	discordID := i.Member.User.ID
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	username, err := store.GetLinkedAccount(ctx, discordID)
	if err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: cfg.Msg("discord_sync_not_linked", "Your Discord account is not linked to any Minecraft account. Use /link first."),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	rank, err := store.GetPlayerRank(ctx, username)
	if err != nil {
		rank = "default"
	}

	var targetRoleID string
	switch strings.ToLower(rank) {
	case "owner":
		targetRoleID = cfg.Discord.Roles.OwnerRoleID
	case "administrator", "admin":
		targetRoleID = cfg.Discord.Roles.AdminRoleID
	case "moderator", "mod":
		targetRoleID = cfg.Discord.Roles.ModRoleID
	case "helper":
		targetRoleID = cfg.Discord.Roles.HelperRoleID
	}

	if targetRoleID != "" {
		_ = s.GuildMemberRoleAdd(i.GuildID, discordID, targetRoleID)
		if cfg.Discord.Roles.StaffRoleID != "" {
			_ = s.GuildMemberRoleAdd(i.GuildID, discordID, cfg.Discord.Roles.StaffRoleID)
		}
	}

	if strings.EqualFold(rank, "default") {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: cfg.Msg("discord_sync_default", "You have the default Player rank. No staff roles assigned."),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	msg := fmt.Sprintf(cfg.Msg("discord_sync_success", "Synchronized rank %s for player %s"), rank, username)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
