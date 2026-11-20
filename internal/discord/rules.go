package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// HandleRulesCommand publishes the server rules and verification panel to the designated channel.
func HandleRulesCommand(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *Config) {
	embed := BuildRulesEmbed(cfg)
	actions := BuildRulesActionRow(cfg)

	targetChannel := cfg.Discord.RulesChannelID
	if targetChannel == "" {
		targetChannel = i.ChannelID
	}

	_, err := s.ChannelMessageSendComplex(targetChannel, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{actions},
	})
	if err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Failed to publish rules: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(cfg.Msg("discord_rules_published", "Rules published to <#%s>"), targetChannel),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleAcceptRules handles the interactive verification button click, adding the verified role.
func HandleAcceptRules(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *Config) {
	if i.Member == nil || i.GuildID == "" {
		return
	}

	userID := i.Member.User.ID
	verifiedRoleID := cfg.Discord.Roles.VerifiedRoleID

	if verifiedRoleID != "" {
		for _, role := range i.Member.Roles {
			if role == verifiedRoleID {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: cfg.Msg("discord_verify_already", "You are already verified."),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
		}

		_ = s.GuildMemberRoleAdd(i.GuildID, userID, verifiedRoleID)
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: cfg.Msg("discord_verify_success", "You have accepted the rules and are now verified!"),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
