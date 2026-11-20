package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Component Custom IDs
const (
	IDBtnLinkAccount     = "btn_link_account"
	IDBtnHowToJoin       = "btn_how_to_join"
	IDBtnRefreshStatus   = "btn_refresh_status"
	IDModalLinkAccount   = "modal_link_account"
	IDInputMinecraftNick = "input_minecraft_nick"
)

// BuildStatusActionRow creates the modern Discord V2 ActionRow with interactive buttons.
func BuildStatusActionRow(cfg *Config) discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    cfg.Msg("discord_btn_link", "Link Account"),
				Style:    discordgo.PrimaryButton,
				CustomID: IDBtnLinkAccount,
				Emoji: &discordgo.ComponentEmoji{
					Name: "🔗",
				},
			},
			discordgo.Button{
				Label:    cfg.Msg("discord_btn_how_to_join", "How to Join"),
				Style:    discordgo.SecondaryButton,
				CustomID: IDBtnHowToJoin,
				Emoji: &discordgo.ComponentEmoji{
					Name: "📖",
				},
			},
			discordgo.Button{
				Label:    cfg.Msg("discord_btn_refresh", "Refresh"),
				Style:    discordgo.SuccessButton,
				CustomID: IDBtnRefreshStatus,
				Emoji: &discordgo.ComponentEmoji{
					Name: "🔄",
				},
			},
		},
	}
}

// BuildLinkAccountModal creates the pop-up modal for entering the player's Minecraft nickname.
func BuildLinkAccountModal(cfg *Config) discordgo.InteractionResponseData {
	title := cfg.Msg("discord_modal_title", "Minecraft Account Verification")
	label := cfg.Msg("discord_modal_nick_label", "Your Minecraft Nickname")
	placeholder := cfg.Msg("discord_modal_nick_placeholder", "e.g. Steve")

	return discordgo.InteractionResponseData{
		CustomID: IDModalLinkAccount,
		Title:    title,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    IDInputMinecraftNick,
						Label:       label,
						Style:       discordgo.TextInputShort,
						Placeholder: placeholder,
						Required:    true,
						MinLength:   3,
						MaxLength:   16,
					},
				},
			},
		},
	}
}

// BuildHowToJoinEmbed creates the modal or ephemeral embed explaining how to join the network.
func BuildHowToJoinEmbed(cfg *Config) *discordgo.MessageEmbed {
	title := cfg.Msg("discord_how_to_join_title", fmt.Sprintf("✦ HOW TO JOIN %s ✦", cfg.Network.Name))
	text := cfg.Msg("discord_how_to_join_text", fmt.Sprintf("Server Address: %s", cfg.Network.Domain))

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: text,
		Color:       0xf6d365, // Brand primary color
		Footer: &discordgo.MessageEmbedFooter{
			Text: cfg.Network.Domain,
		},
	}
}
