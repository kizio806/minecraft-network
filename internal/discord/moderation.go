package discord

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var validMinecraftUsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,16}$`)

// ValidateMinecraftUsername ensures the username strictly conforms to standard Minecraft specifications (3-16 chars, alphanumeric + underscores).
func ValidateMinecraftUsername(username string) error {
	trimmed := strings.TrimSpace(username)
	if !validMinecraftUsernameRegex.MatchString(trimmed) {
		return fmt.Errorf("invalid Minecraft username %q: must be 3-16 characters and contain only letters, numbers, and underscores", username)
	}
	return nil
}

// SanitizePunishReason strips control characters, normalizes whitespace, and truncates to max 128 characters to prevent console injection.
func SanitizePunishReason(reason string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' || r == '\x00' {
			return ' '
		}
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, reason)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "" {
		return "Violation of server rules"
	}
	if len(cleaned) > 128 {
		cleaned = cleaned[:128]
	}
	return cleaned
}

// ParsePunishDuration validates and normalizes a punishment duration string.
func ParsePunishDuration(raw string) (string, error) {
	val := strings.ToLower(strings.TrimSpace(raw))
	switch val {
	case "p", "perm", "permanent":
		return "permanent", nil
	}

	if len(val) < 2 {
		return "", fmt.Errorf("invalid duration format: %q (expected e.g. 1h, 12h, 24h, 7d, 30d, permanent)", raw)
	}

	unit := val[len(val)-1]
	numStr := val[:len(val)-1]
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		return "", fmt.Errorf("invalid duration number: %q", raw)
	}

	switch unit {
	case 'm', 'h', 'd', 'w', 'y':
		return fmt.Sprintf("%d%c", n, unit), nil
	default:
		return "", fmt.Errorf("unknown duration unit %q in %q (expected m, h, d, w, y or permanent)", unit, raw)
	}
}

// HandleReportCommand processes `/report player:<nick> reason:<text>`.
func HandleReportCommand(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *Config, store Store) {
	data := i.ApplicationCommandData()
	var targetPlayer, reason string

	for _, opt := range data.Options {
		switch opt.Name {
		case "player":
			targetPlayer = strings.TrimSpace(opt.StringValue())
		case "reason":
			reason = strings.TrimSpace(opt.StringValue())
		}
	}

	if targetPlayer == "" || reason == "" {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Please provide both player and reason.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if err := ValidateMinecraftUsername(targetPlayer); err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Invalid player name: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	reason = SanitizePunishReason(reason)

	reporterID := i.Member.User.ID
	reporterTag := i.Member.User.Username
	if i.Member.User.Discriminator != "0" && i.Member.User.Discriminator != "" {
		reporterTag = fmt.Sprintf("%s#%s", reporterTag, i.Member.User.Discriminator)
	}

	report := &Report{
		ReporterID:   reporterID,
		ReporterTag:  reporterTag,
		TargetPlayer: targetPlayer,
		Reason:       reason,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := store.SaveReport(ctx, report); err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Failed to submit report: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	reportChannel := cfg.Discord.ReportChannelID
	if reportChannel == "" {
		reportChannel = cfg.Discord.StaffChannelID
	}

	if reportChannel != "" {
		embed := BuildReportEmbed(cfg, report)
		actions := BuildReportActionRow(cfg, report.ID)
		_, _ = s.ChannelMessageSendComplex(reportChannel, &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{actions},
		})
	}

	respMsg := fmt.Sprintf(cfg.Msg("discord_report_submitted", "Report submitted for %s"), targetPlayer)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respMsg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleReportButton handles staff moderation button interactions on report messages.
func HandleReportButton(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *Config, store Store, _ ConsoleExecutor) {
	customID := i.MessageComponentData().CustomID

	switch {
	case strings.HasPrefix(customID, IDBtnReportResolvePrefix):
		idStr := strings.TrimPrefix(customID, IDBtnReportResolvePrefix)
		reportID, _ := strconv.ParseInt(idStr, 10, 64)
		staffUser := i.Member.User.Username

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := store.UpdateReportStatus(ctx, reportID, "resolved", staffUser); err != nil {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Failed to resolve report: %v", err),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		rep, err := store.GetReport(ctx, reportID)
		if err == nil {
			embed := BuildReportEmbed(cfg, rep)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		}

	case strings.HasPrefix(customID, IDBtnReportDismissPrefix):
		idStr := strings.TrimPrefix(customID, IDBtnReportDismissPrefix)
		reportID, _ := strconv.ParseInt(idStr, 10, 64)
		staffUser := i.Member.User.Username

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := store.UpdateReportStatus(ctx, reportID, "dismissed", staffUser); err != nil {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Failed to dismiss report: %v", err),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		rep, err := store.GetReport(ctx, reportID)
		if err == nil {
			embed := BuildReportEmbed(cfg, rep)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		}

	case strings.HasPrefix(customID, IDBtnReportPunishPrefix):
		idStr := strings.TrimPrefix(customID, IDBtnReportPunishPrefix)
		reportID, _ := strconv.ParseInt(idStr, 10, 64)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rep, err := store.GetReport(ctx, reportID)
		if err != nil {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Report not found: %v", err),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		modal := BuildPunishModal(cfg, reportID, rep.TargetPlayer)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &modal,
		})
	}
}

// HandlePunishModalSubmit executes punishment command and marks report resolved.
func HandlePunishModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *Config, store Store, exec ConsoleExecutor) {
	data := i.ModalSubmitData()
	payload := strings.TrimPrefix(data.CustomID, IDModalPunishPrefix)
	parts := strings.SplitN(payload, "_", 2)
	if len(parts) != 2 {
		return
	}

	reportID, _ := strconv.ParseInt(parts[0], 10, 64)
	targetPlayer := parts[1]

	if err := ValidateMinecraftUsername(targetPlayer); err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Invalid target player: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var rawDuration, reason string
	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, input := range row.Components {
				if textInput, ok := input.(*discordgo.TextInput); ok {
					switch textInput.CustomID {
					case IDInputPunishDuration:
						rawDuration = strings.TrimSpace(textInput.Value)
					case IDInputPunishReason:
						reason = strings.TrimSpace(textInput.Value)
					}
				}
			}
		}
	}

	duration, err := ParsePunishDuration(rawDuration)
	if err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Invalid duration: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	reason = SanitizePunishReason(reason)

	var cmd string
	if duration == "permanent" {
		cmd = fmt.Sprintf("ban %s %s", targetPlayer, reason)
	} else {
		cmd = fmt.Sprintf("tempban %s %s %s", targetPlayer, duration, reason)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if exec != nil {
		_, _ = exec.Execute(ctx, "survival", cmd)
	}

	staffUser := i.Member.User.Username
	_ = store.UpdateReportStatus(ctx, reportID, "resolved", staffUser)

	confirm := fmt.Sprintf(cfg.Msg("discord_punish_success", "Player %s punished for %s (%s)"), targetPlayer, duration, reason)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: confirm,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
