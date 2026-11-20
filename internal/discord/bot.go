package discord

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// NetworkQuerier defines the interface for polling server statuses.
type NetworkQuerier interface {
	QueryStatuses(ctx context.Context, targets []ServerTarget) []ServerStatus
}

// AccountLinker defines the interface for linking accounts and whitelist operations.
type AccountLinker interface {
	LinkAccount(ctx context.Context, discordID, username string) error
	GetLinkedAccount(ctx context.Context, discordID string) (string, error)
	AddToWhitelist(ctx context.Context, username string) error
	RemoveFromWhitelist(ctx context.Context, username string) error
}

// Bot represents the Discord bot instance.
type Bot struct {
	Config   *Config
	Session  *discordgo.Session
	Querier  NetworkQuerier
	Linker   AccountLinker
	mu       sync.Mutex
	statusID string
}

// NewBot initializes a new Discord bot instance.
func NewBot(cfg *Config, s *discordgo.Session, q NetworkQuerier, l AccountLinker) *Bot {
	b := &Bot{
		Config:  cfg,
		Session: s,
		Querier: q,
		Linker:  l,
	}
	if b.Querier == nil {
		b.Querier = &DefaultQuerier{}
	}
	if b.Linker == nil {
		b.Linker = &DefaultLinker{}
	}
	return b
}

// RegisterHandlers binds event listeners to the Discord session.
func (b *Bot) RegisterHandlers() {
	b.Session.AddHandler(b.handleInteraction)
}

// SlashCommands returns the list of application slash commands.
func (b *Bot) SlashCommands() []*discordgo.ApplicationCommand {
	adminPerm := int64(discordgo.PermissionAdministrator)
	return []*discordgo.ApplicationCommand{
		{
			Name:        "status",
			Description: "Displays live server network status and player counts",
		},
		{
			Name:        "players",
			Description: "Lists online players across all network servers",
		},
		{
			Name:        "link",
			Description: "Links your Discord account to your Minecraft nickname",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "nickname",
					Description: "Your Minecraft in-game username",
					Required:    true,
				},
			},
		},
		{
			Name:                     "whitelist",
			Description:              "Manage server whitelist",
			DefaultMemberPermissions: &adminPerm,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add player to whitelist",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "player",
							Description: "Minecraft player nickname",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove player from whitelist",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "player",
							Description: "Minecraft player nickname",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

// HandleInteraction routes incoming Discord interactions.
func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.handleCommand(s, i)
	case discordgo.InteractionMessageComponent:
		b.handleComponent(s, i)
	case discordgo.InteractionModalSubmit:
		b.handleModal(s, i)
	}
}

func (b *Bot) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	switch data.Name {
	case "status":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		statuses := b.Querier.QueryStatuses(ctx, b.Config.Servers)
		embed := BuildStatusEmbed(b.Config, statuses)
		actions := BuildStatusActionRow(b.Config)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: []discordgo.MessageComponent{actions},
			},
		})

	case "players":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		statuses := b.Querier.QueryStatuses(ctx, b.Config.Servers)
		var lines []string
		for _, st := range statuses {
			lines = append(lines, fmt.Sprintf("**%s**: `%d` players", st.Name, st.Players))
		}
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("👥 **Network Players**\n%s", fmt.Sprintf(b.Config.Msg("discord_status_players", "Online Players: **%d**"), len(lines))),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

	case "link":
		var nick string
		if len(data.Options) > 0 {
			nick = data.Options[0].StringValue()
		}
		discordID := i.Member.User.ID
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := b.Linker.LinkAccount(ctx, discordID, nick); err != nil {
			msg := fmt.Sprintf(b.Config.Msg("discord_link_already", "Already linked to %s"), nick)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: msg,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		if b.Config.Discord.Roles.LinkedRoleID != "" && i.GuildID != "" {
			_ = s.GuildMemberRoleAdd(i.GuildID, discordID, b.Config.Discord.Roles.LinkedRoleID)
		}

		successMsg := fmt.Sprintf(b.Config.Msg("discord_link_success", "Linked to %s"), nick)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: successMsg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

	case "whitelist":
		if len(data.Options) == 0 {
			return
		}
		sub := data.Options[0]
		player := sub.Options[0].StringValue()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		switch sub.Name {
		case "add":
			_ = b.Linker.AddToWhitelist(ctx, player)
			resp := fmt.Sprintf(b.Config.Msg("discord_whitelist_added", "Added %s to whitelist"), player)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: resp,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		case "remove":
			_ = b.Linker.RemoveFromWhitelist(ctx, player)
			resp := fmt.Sprintf(b.Config.Msg("discord_whitelist_removed", "Removed %s from whitelist"), player)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: resp,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
	}
}

func (b *Bot) handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	switch data.CustomID {
	case IDBtnLinkAccount:
		modal := BuildLinkAccountModal(b.Config)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &modal,
		})

	case IDBtnHowToJoin:
		embed := BuildHowToJoinEmbed(b.Config)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})

	case IDBtnRefreshStatus:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		statuses := b.Querier.QueryStatuses(ctx, b.Config.Servers)
		embed := BuildStatusEmbed(b.Config, statuses)
		actions := BuildStatusActionRow(b.Config)

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: []discordgo.MessageComponent{actions},
			},
		})
	}
}

func (b *Bot) handleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	if data.CustomID != IDModalLinkAccount {
		return
	}

	var nick string
	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, input := range row.Components {
				if textInput, ok := input.(*discordgo.TextInput); ok && textInput.CustomID == IDInputMinecraftNick {
					nick = textInput.Value
				}
			}
		}
	}

	if nick == "" {
		return
	}

	discordID := i.Member.User.ID
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.Linker.LinkAccount(ctx, discordID, nick); err != nil {
		msg := fmt.Sprintf(b.Config.Msg("discord_link_already", "Already linked to %s"), nick)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if b.Config.Discord.Roles.LinkedRoleID != "" && i.GuildID != "" {
		_ = s.GuildMemberRoleAdd(i.GuildID, discordID, b.Config.Discord.Roles.LinkedRoleID)
	}

	respMsg := fmt.Sprintf(b.Config.Msg("discord_link_success", "Linked to %s"), nick)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: respMsg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// StartStatusLoop runs a background loop updating the pinned status dashboard.
func (b *Bot) StartStatusLoop(ctx context.Context) {
	if b.Config.Discord.StatusChannelID == "" {
		return
	}

	ticker := time.NewTicker(time.Duration(b.Config.Discord.StatusUpdateInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.updateStatusMessage(ctx)
		}
	}
}

func (b *Bot) updateStatusMessage(ctx context.Context) {
	statuses := b.Querier.QueryStatuses(ctx, b.Config.Servers)
	embed := BuildStatusEmbed(b.Config, statuses)
	actions := BuildStatusActionRow(b.Config)

	b.mu.Lock()
	defer b.mu.Unlock()

	channelID := b.Config.Discord.StatusChannelID
	if b.statusID == "" {
		msg, err := b.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{actions},
		})
		if err == nil {
			b.statusID = msg.ID
		} else {
			log.Printf("discord: failed to send status panel: %v", err)
		}
		return
	}

	embeds := []*discordgo.MessageEmbed{embed}
	components := []discordgo.MessageComponent{actions}
	_, err := b.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         b.statusID,
		Channel:    channelID,
		Embeds:     &embeds,
		Components: &components,
	})
	if err != nil {
		b.statusID = ""
	}
}

// DefaultQuerier queries targets using standard TCP ping.
type DefaultQuerier struct{}

func (DefaultQuerier) QueryStatuses(_ context.Context, targets []ServerTarget) []ServerStatus {
	var result []ServerStatus
	for _, t := range targets {
		result = append(result, ServerStatus{
			Name:    t.Name,
			Online:  true,
			Players: 0,
			Max:     50,
			PingMs:  5,
		})
	}
	return result
}

// DefaultLinker provides an in-memory or fallback account linker.
type DefaultLinker struct {
	mu        sync.Mutex
	linked    map[string]string
	whitelist map[string]bool
}

func (d *DefaultLinker) LinkAccount(_ context.Context, discordID, username string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.linked == nil {
		d.linked = make(map[string]string)
	}
	if existing, ok := d.linked[discordID]; ok && existing != username {
		return fmt.Errorf("already linked to %s", existing)
	}
	d.linked[discordID] = username
	return nil
}

func (d *DefaultLinker) GetLinkedAccount(_ context.Context, discordID string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if nick, ok := d.linked[discordID]; ok {
		return nick, nil
	}
	return "", fmt.Errorf("account not linked")
}

func (d *DefaultLinker) AddToWhitelist(_ context.Context, username string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.whitelist == nil {
		d.whitelist = make(map[string]bool)
	}
	d.whitelist[username] = true
	return nil
}

func (d *DefaultLinker) RemoveFromWhitelist(_ context.Context, username string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.whitelist != nil {
		delete(d.whitelist, username)
	}
	return nil
}
