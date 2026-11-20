package discord

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

type mockDiscordTransport struct {
	fn func(req *http.Request) (*http.Response, error)
}

func (m *mockDiscordTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.fn != nil {
		return m.fn(req)
	}
	// Return valid JSON response for Discord REST requests
	respBody := `{"id": "mock_id_123", "channel_id": "mock_chan_123"}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(respBody)),
		Header:     make(http.Header),
	}, nil
}

func newMockSession() *discordgo.Session {
	s, _ := discordgo.New("Bot mock_token_xyz")
	s.Client.Transport = &mockDiscordTransport{}
	return s
}

func TestBot_InteractionHandling(t *testing.T) {
	s := newMockSession()
	cfg := &Config{
		Discord: DiscordSection{
			GuildID:         "guild_1",
			StatusChannelID: "status_chan",
			StaffChannelID:  "staff_chan",
			RulesChannelID:  "rules_chan",
			ReportChannelID: "report_chan",
			Roles: RolesConfig{
				VerifiedRoleID: "role_verified",
				LinkedRoleID:   "role_linked",
				StaffRoleID:    "role_staff",
				HelperRoleID:   "role_helper",
				ModRoleID:      "role_mod",
				AdminRoleID:    "role_admin",
				OwnerRoleID:    "role_owner",
			},
		},
		Servers: []ServerTarget{
			{Name: "Proxy", Host: "127.0.0.1", Port: 25565},
		},
		Messages: make(map[string]string),
	}

	store := NewMemoryStore()
	exec := &DefaultConsoleExecutor{}
	bot := NewBot(cfg, s, nil, store, exec)
	bot.RegisterHandlers()

	t.Run("command: status", func(_ *testing.T) {
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "status",
				},
			},
		})
	})

	t.Run("command: players", func(_ *testing.T) {
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "players",
				},
			},
		})
	})

	t.Run("command: link", func(_ *testing.T) {
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:    discordgo.InteractionApplicationCommand,
				GuildID: "guild_1",
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "disc_user_1"},
				},
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "link",
					Options: []*discordgo.ApplicationCommandInteractionDataOption{
						{Type: discordgo.ApplicationCommandOptionString, Value: "PlayerOne"},
					},
				},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:    discordgo.InteractionApplicationCommand,
				GuildID: "guild_1",
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "disc_user_1"},
				},
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "link",
					Options: []*discordgo.ApplicationCommandInteractionDataOption{
						{Type: discordgo.ApplicationCommandOptionString, Value: "PlayerConflict"},
					},
				},
			},
		})
	})

	t.Run("command: whitelist add and remove", func(_ *testing.T) {
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "whitelist",
					Options: []*discordgo.ApplicationCommandInteractionDataOption{
						{
							Name: "add",
							Type: discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandInteractionDataOption{
								{Type: discordgo.ApplicationCommandOptionString, Value: "Steve"},
							},
						},
					},
				},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "whitelist",
					Options: []*discordgo.ApplicationCommandInteractionDataOption{
						{
							Name: "remove",
							Type: discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandInteractionDataOption{
								{Type: discordgo.ApplicationCommandOptionString, Value: "Steve"},
							},
						},
					},
				},
			},
		})
	})

	t.Run("command: sync", func(_ *testing.T) {
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:    discordgo.InteractionApplicationCommand,
				GuildID: "guild_1",
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "disc_unlinked"},
				},
				Data: discordgo.ApplicationCommandInteractionData{Name: "sync"},
			},
		})

		_ = store.LinkAccount(context.Background(), "disc_user_1", "PlayerOne")
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:    discordgo.InteractionApplicationCommand,
				GuildID: "guild_1",
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "disc_user_1"},
				},
				Data: discordgo.ApplicationCommandInteractionData{Name: "sync"},
			},
		})

		store.SetPlayerRank("PlayerOne", "owner")
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:    discordgo.InteractionApplicationCommand,
				GuildID: "guild_1",
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "disc_user_1"},
				},
				Data: discordgo.ApplicationCommandInteractionData{Name: "sync"},
			},
		})

		store.SetPlayerRank("PlayerOne", "moderator")
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:    discordgo.InteractionApplicationCommand,
				GuildID: "guild_1",
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "disc_user_1"},
				},
				Data: discordgo.ApplicationCommandInteractionData{Name: "sync"},
			},
		})

		store.SetPlayerRank("PlayerOne", "helper")
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:    discordgo.InteractionApplicationCommand,
				GuildID: "guild_1",
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "disc_user_1"},
				},
				Data: discordgo.ApplicationCommandInteractionData{Name: "sync"},
			},
		})
	})

	t.Run("command: report", func(_ *testing.T) {
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{Name: "report"},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "reporter_1", Username: "ReporterUser", Discriminator: "1234"},
				},
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "report",
					Options: []*discordgo.ApplicationCommandInteractionDataOption{
						{Name: "player", Type: discordgo.ApplicationCommandOptionString, Value: "TargetCheater"},
						{Name: "reason", Type: discordgo.ApplicationCommandOptionString, Value: "Speed hacking"},
					},
				},
			},
		})
	})

	t.Run("command: rules", func(_ *testing.T) {
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:      discordgo.InteractionApplicationCommand,
				ChannelID: "current_chan",
				Data:      discordgo.ApplicationCommandInteractionData{Name: "rules"},
			},
		})

		cfg.Discord.RulesChannelID = ""
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:      discordgo.InteractionApplicationCommand,
				ChannelID: "fallback_chan",
				Data:      discordgo.ApplicationCommandInteractionData{Name: "rules"},
			},
		})
		cfg.Discord.RulesChannelID = "rules_chan"

		errSession := newMockSession()
		errSession.Client.Transport = &mockDiscordTransport{
			fn: func(_ *http.Request) (*http.Response, error) {
				return nil, errors.New("network error")
			},
		}
		botErr := NewBot(cfg, errSession, nil, store, exec)
		botErr.handleInteraction(errSession, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{Name: "rules"},
			},
		})
	})

	t.Run("command: rcon", func(_ *testing.T) {
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Member: &discordgo.Member{
					Permissions: 0,
				},
				Data: discordgo.ApplicationCommandInteractionData{Name: "rcon"},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Member: &discordgo.Member{
					Permissions: int64(discordgo.PermissionAdministrator),
				},
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "rcon",
					Options: []*discordgo.ApplicationCommandInteractionDataOption{
						{Name: "server", Type: discordgo.ApplicationCommandOptionString, Value: "invalid_srv"},
						{Name: "command", Type: discordgo.ApplicationCommandOptionString, Value: "list"},
					},
				},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Member: &discordgo.Member{
					Permissions: int64(discordgo.PermissionAdministrator),
				},
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "rcon",
					Options: []*discordgo.ApplicationCommandInteractionDataOption{
						{Name: "server", Type: discordgo.ApplicationCommandOptionString, Value: "lobby"},
						{Name: "command", Type: discordgo.ApplicationCommandOptionString, Value: "say hello"},
					},
				},
			},
		})

		bot.Executor = &DefaultConsoleExecutor{MockErr: errors.New("exec error")}
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Member: &discordgo.Member{
					Permissions: int64(discordgo.PermissionAdministrator),
				},
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "rcon",
					Options: []*discordgo.ApplicationCommandInteractionDataOption{
						{Name: "server", Type: discordgo.ApplicationCommandOptionString, Value: "survival"},
						{Name: "command", Type: discordgo.ApplicationCommandOptionString, Value: "stop"},
					},
				},
			},
		})
		bot.Executor = exec
	})

	t.Run("components: buttons", func(_ *testing.T) {
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionMessageComponent,
				Data: discordgo.MessageComponentInteractionData{CustomID: IDBtnLinkAccount},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionMessageComponent,
				Data: discordgo.MessageComponentInteractionData{CustomID: IDBtnHowToJoin},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionMessageComponent,
				Data: discordgo.MessageComponentInteractionData{CustomID: IDBtnRefreshStatus},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:    discordgo.InteractionMessageComponent,
				GuildID: "guild_1",
				Member: &discordgo.Member{
					User:  &discordgo.User{ID: "newbie_1"},
					Roles: []string{},
				},
				Data: discordgo.MessageComponentInteractionData{CustomID: IDBtnAcceptRules},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:    discordgo.InteractionMessageComponent,
				GuildID: "guild_1",
				Member: &discordgo.Member{
					User:  &discordgo.User{ID: "verified_1"},
					Roles: []string{"role_verified"},
				},
				Data: discordgo.MessageComponentInteractionData{CustomID: IDBtnAcceptRules},
			},
		})

		_ = store.SaveReport(context.Background(), &Report{ReporterID: "u1", TargetPlayer: "Bad", Reason: "grief"})
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionMessageComponent,
				Member: &discordgo.Member{
					User: &discordgo.User{Username: "StaffMember"},
				},
				Data: discordgo.MessageComponentInteractionData{
					CustomID: IDBtnReportResolvePrefix + "1",
				},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionMessageComponent,
				Member: &discordgo.Member{
					User: &discordgo.User{Username: "StaffMember"},
				},
				Data: discordgo.MessageComponentInteractionData{
					CustomID: IDBtnReportDismissPrefix + "1",
				},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionMessageComponent,
				Member: &discordgo.Member{
					User: &discordgo.User{Username: "StaffMember"},
				},
				Data: discordgo.MessageComponentInteractionData{
					CustomID: IDBtnReportPunishPrefix + "1",
				},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionMessageComponent,
				Member: &discordgo.Member{
					User: &discordgo.User{Username: "StaffMember"},
				},
				Data: discordgo.MessageComponentInteractionData{
					CustomID: IDBtnReportPunishPrefix + "9999",
				},
			},
		})
	})

	t.Run("modals", func(_ *testing.T) {
		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:    discordgo.InteractionModalSubmit,
				GuildID: "guild_1",
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "disc_modal_user"},
				},
				Data: discordgo.ModalSubmitInteractionData{
					CustomID: IDModalLinkAccount,
					Components: []discordgo.MessageComponent{
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: IDInputMinecraftNick,
									Value:    "ModalSteve",
								},
							},
						},
					},
				},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:    discordgo.InteractionModalSubmit,
				GuildID: "guild_1",
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "disc_modal_user"},
				},
				Data: discordgo.ModalSubmitInteractionData{
					CustomID: IDModalLinkAccount,
					Components: []discordgo.MessageComponent{
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: IDInputMinecraftNick,
									Value:    "",
								},
							},
						},
					},
				},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type:    discordgo.InteractionModalSubmit,
				GuildID: "guild_1",
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "disc_modal_user"},
				},
				Data: discordgo.ModalSubmitInteractionData{
					CustomID: IDModalLinkAccount,
					Components: []discordgo.MessageComponent{
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: IDInputMinecraftNick,
									Value:    "DifferentNick",
								},
							},
						},
					},
				},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionModalSubmit,
				Member: &discordgo.Member{
					User: &discordgo.User{Username: "AdminUser"},
				},
				Data: discordgo.ModalSubmitInteractionData{
					CustomID: IDModalPunishPrefix + "1_Bad",
					Components: []discordgo.MessageComponent{
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: IDInputPunishDuration,
									Value:    "permanent",
								},
							},
						},
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: IDInputPunishReason,
									Value:    "Confirmed severe cheating",
								},
							},
						},
					},
				},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionModalSubmit,
				Member: &discordgo.Member{
					User: &discordgo.User{Username: "AdminUser"},
				},
				Data: discordgo.ModalSubmitInteractionData{
					CustomID: IDModalPunishPrefix + "1_Bad",
					Components: []discordgo.MessageComponent{
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: IDInputPunishDuration,
									Value:    "12h",
								},
							},
						},
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: IDInputPunishReason,
									Value:    "Minor griefing",
								},
							},
						},
					},
				},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionModalSubmit,
				Member: &discordgo.Member{
					User: &discordgo.User{Username: "AdminUser"},
				},
				Data: discordgo.ModalSubmitInteractionData{
					CustomID: IDModalPunishPrefix + "1_Bad",
					Components: []discordgo.MessageComponent{
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: IDInputPunishDuration,
									Value:    "invalid_duration",
								},
							},
						},
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: IDInputPunishReason,
									Value:    "test",
								},
							},
						},
					},
				},
			},
		})
	})

	t.Run("status update loop", func(_ *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		cfg.Discord.StatusUpdateInterval = 1
		bot.updateStatusMessage(ctx)
		bot.updateStatusMessage(ctx)

		bot.StartStatusLoop(ctx)
	})
}

func TestDockerConsoleExecutor(t *testing.T) {
	ctx := context.Background()
	exec := &DockerConsoleExecutor{ComposeFile: "docker-compose.yml"}

	t.Run("invalid server target", func(t *testing.T) {
		_, err := exec.Execute(ctx, "invalid_srv", "help")
		if err == nil {
			t.Fatalf("expected error for invalid server, got nil")
		}
	})

	t.Run("valid servers invocation check", func(_ *testing.T) {
		_, _ = exec.Execute(ctx, "proxy", "help")
		_, _ = exec.Execute(ctx, "lobby", "help")
	})
}
