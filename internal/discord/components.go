package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Component Custom IDs
const (
	IDBtnLinkAccount         = "btn_link_account"
	IDBtnHowToJoin           = "btn_how_to_join"
	IDBtnRefreshStatus       = "btn_refresh_status"
	IDBtnAcceptRules         = "btn_accept_rules"
	IDBtnReportResolvePrefix = "btn_report_resolve_"
	IDBtnReportDismissPrefix = "btn_report_dismiss_"
	IDBtnReportPunishPrefix  = "btn_report_punish_"

	IDModalLinkAccount    = "modal_link_account"
	IDModalPunishPrefix   = "modal_punish_"
	IDInputMinecraftNick  = "input_minecraft_nick"
	IDInputPunishDuration = "input_punish_duration"
	IDInputPunishReason   = "input_punish_reason"
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
		Color:       0xf6d365,
		Footer: &discordgo.MessageEmbedFooter{
			Text: cfg.Network.Domain,
		},
	}
}

// BuildRulesEmbed creates the rich embed displaying network rules and verification instructions.
func BuildRulesEmbed(cfg *Config) *discordgo.MessageEmbed {
	title := cfg.Msg("discord_rules_title", "📜 Network Rules & Guidelines")
	body := cfg.Msg("discord_rules_body", "Please read and accept the server rules before playing.")

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: body,
		Color:       0x2ecc71,
		Footer: &discordgo.MessageEmbedFooter{
			Text: cfg.Network.Domain,
		},
	}
}

// BuildRulesActionRow creates the ActionRow button for accepting rules and verifying.
func BuildRulesActionRow(cfg *Config) discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    cfg.Msg("discord_btn_accept_rules", "Accept Rules & Verify"),
				Style:    discordgo.SuccessButton,
				CustomID: IDBtnAcceptRules,
				Emoji: &discordgo.ComponentEmoji{
					Name: "✅",
				},
			},
		},
	}
}

// BuildReportEmbed creates the staff moderation embed representing a player report.
func BuildReportEmbed(cfg *Config, r *Report) *discordgo.MessageEmbed {
	var color int
	var statusText string

	switch r.Status {
	case "resolved":
		color = 0x2ecc71
		statusText = cfg.Msg("discord_report_status_resolved", "✅ Resolved")
	case "dismissed":
		color = 0x95a5a6
		statusText = cfg.Msg("discord_report_status_dismissed", "❌ Dismissed")
	default:
		color = 0xff4e50
		statusText = cfg.Msg("discord_report_status_pending", "⏳ Pending Staff Review")
	}

	reporterDisplay := r.ReporterTag
	if reporterDisplay == "" {
		reporterDisplay = fmt.Sprintf("<@%s>", r.ReporterID)
	} else {
		reporterDisplay = fmt.Sprintf("<@%s> (`%s`)", r.ReporterID, reporterDisplay)
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   cfg.Msg("discord_report_field_target", "Reported Player"),
			Value:  fmt.Sprintf("**%s**", r.TargetPlayer),
			Inline: true,
		},
		{
			Name:   cfg.Msg("discord_report_field_reporter", "Reporter"),
			Value:  reporterDisplay,
			Inline: true,
		},
		{
			Name:   cfg.Msg("discord_report_field_status", "Status"),
			Value:  statusText,
			Inline: true,
		},
		{
			Name:   cfg.Msg("discord_report_field_reason", "Reason"),
			Value:  fmt.Sprintf("```\n%s\n```", r.Reason),
			Inline: false,
		},
	}

	if r.ResolvedBy != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   cfg.Msg("discord_report_field_resolved_by", "Resolved By"),
			Value:  fmt.Sprintf("**%s**", r.ResolvedBy),
			Inline: true,
		})
	}

	return &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("%s #%d", cfg.Msg("discord_report_title", "🚨 In-Game Player Report"), r.ID),
		Color:     color,
		Fields:    fields,
		Timestamp: r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Footer: &discordgo.MessageEmbedFooter{
			Text: cfg.Network.Domain,
		},
	}
}

// BuildReportActionRow creates staff action buttons for resolving, dismissing, or punishing.
func BuildReportActionRow(cfg *Config, reportID int64) discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    cfg.Msg("discord_btn_report_resolve", "Resolve"),
				Style:    discordgo.SuccessButton,
				CustomID: fmt.Sprintf("%s%d", IDBtnReportResolvePrefix, reportID),
				Emoji: &discordgo.ComponentEmoji{
					Name: "✅",
				},
			},
			discordgo.Button{
				Label:    cfg.Msg("discord_btn_report_dismiss", "Dismiss"),
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s%d", IDBtnReportDismissPrefix, reportID),
				Emoji: &discordgo.ComponentEmoji{
					Name: "❌",
				},
			},
			discordgo.Button{
				Label:    cfg.Msg("discord_btn_report_punish", "Punish"),
				Style:    discordgo.DangerButton,
				CustomID: fmt.Sprintf("%s%d", IDBtnReportPunishPrefix, reportID),
				Emoji: &discordgo.ComponentEmoji{
					Name: "🔨",
				},
			},
		},
	}
}

// BuildPunishModal constructs the staff moderation modal for specifying duration and reason.
func BuildPunishModal(cfg *Config, reportID int64, targetPlayer string) discordgo.InteractionResponseData {
	title := cfg.Msg("discord_modal_punish_title", "Staff Moderation • Punish Player")
	durationLabel := cfg.Msg("discord_modal_punish_duration_label", "Duration (e.g. 1h, 12h, 24h, 7d, 30d, perm)")
	reasonLabel := cfg.Msg("discord_modal_punish_reason_label", "Punishment Reason / Evidence")

	return discordgo.InteractionResponseData{
		CustomID: fmt.Sprintf("%s%d_%s", IDModalPunishPrefix, reportID, targetPlayer),
		Title:    title,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    IDInputPunishDuration,
						Label:       durationLabel,
						Style:       discordgo.TextInputShort,
						Placeholder: "12h",
						Required:    true,
						MinLength:   1,
						MaxLength:   16,
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    IDInputPunishReason,
						Label:       reasonLabel,
						Style:       discordgo.TextInputParagraph,
						Placeholder: "Violation of network rules",
						Required:    true,
						MinLength:   3,
						MaxLength:   256,
					},
				},
			},
		},
	}
}
