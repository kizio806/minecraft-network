package discord

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "reporter_1", Username: "ReporterUser"},
				},
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "report",
					Options: []*discordgo.ApplicationCommandInteractionDataOption{
						{Name: "player", Type: discordgo.ApplicationCommandOptionString, Value: "inv@lid!"},
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

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionModalSubmit,
				Data: discordgo.ModalSubmitInteractionData{
					CustomID: IDModalPunishPrefix + "invalid_split",
				},
			},
		})

		bot.handleInteraction(s, &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionModalSubmit,
				Data: discordgo.ModalSubmitInteractionData{
					CustomID: IDModalPunishPrefix + "1_inv@lid!",
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

func TestResolveDockerPath(t *testing.T) {
	origStat := statFunc
	origLookPath := lookPathFunc
	defer func() {
		statFunc = origStat
		lookPathFunc = origLookPath
	}()

	t.Run("found in candidates", func(t *testing.T) {
		statFunc = func(name string) (os.FileInfo, error) {
			if name == "/usr/bin/docker" {
				return nil, nil
			}
			return nil, os.ErrNotExist
		}
		if p := resolveDockerPath(); p != "/usr/bin/docker" {
			t.Fatalf("expected /usr/bin/docker, got %s", p)
		}
	})

	t.Run("found via lookPath absolute", func(t *testing.T) {
		statFunc = func(string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		}
		lookPathFunc = func(string) (string, error) {
			return "/custom/bin/docker", nil
		}
		if p := resolveDockerPath(); p != "/custom/bin/docker" {
			t.Fatalf("expected /custom/bin/docker, got %s", p)
		}
	})

	t.Run("fallback to default", func(t *testing.T) {
		statFunc = func(string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		}
		lookPathFunc = func(string) (string, error) {
			return "relative-docker", nil
		}
		if p := resolveDockerPath(); p != "/usr/bin/docker" {
			t.Fatalf("expected /usr/bin/docker, got %s", p)
		}
	})
}

func TestDockerConsoleExecutor(t *testing.T) {
	ctx := context.Background()
	origCmd := execCommandContext
	defer func() { execCommandContext = origCmd }()

	execRunner := &DockerConsoleExecutor{ComposeFile: "docker-compose.yml"}

	t.Run("invalid server target", func(t *testing.T) {
		_, err := execRunner.Execute(ctx, "invalid_srv", "help")
		if err == nil {
			t.Fatalf("expected error for invalid server, got nil")
		}
	})

	t.Run("successful execution with output", func(t *testing.T) {
		execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			cs := []string{"-test.run=TestHelperProcessSubprocess", "--", "mock_success"}
			//nolint:gosec // Subprocess invocation is strictly restricted to test binary
			return exec.CommandContext(ctx, os.Args[0], cs...)
		}
		res, err := execRunner.Execute(ctx, "proxy", "list")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != "success output" {
			t.Fatalf("expected success output, got %q", res)
		}
	})

	t.Run("successful execution with empty output", func(t *testing.T) {
		execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			cs := []string{"-test.run=TestHelperProcessSubprocess", "--", "mock_empty"}
			//nolint:gosec // Subprocess invocation is strictly restricted to test binary
			return exec.CommandContext(ctx, os.Args[0], cs...)
		}
		res, err := execRunner.Execute(ctx, "lobby", "list")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != "Command executed successfully (no output)." {
			t.Fatalf("unexpected result: %q", res)
		}
	})

	t.Run("execution error with stderr", func(t *testing.T) {
		execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			cs := []string{"-test.run=TestHelperProcessSubprocess", "--", "mock_stderr"}
			//nolint:gosec // Subprocess invocation is strictly restricted to test binary
			return exec.CommandContext(ctx, os.Args[0], cs...)
		}
		_, err := execRunner.Execute(ctx, "survival", "stop")
		if err == nil || err.Error() != "fatal server error" {
			t.Fatalf("expected fatal server error, got: %v", err)
		}
	})

	t.Run("execution error with empty stderr", func(t *testing.T) {
		execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			cs := []string{"-test.run=TestHelperProcessSubprocess", "--", "mock_exit_only"}
			//nolint:gosec // Subprocess invocation is strictly restricted to test binary
			return exec.CommandContext(ctx, os.Args[0], cs...)
		}
		noFileExec := &DockerConsoleExecutor{}
		_, err := noFileExec.Execute(ctx, "survival", "stop")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestHelperProcessSubprocess(_ *testing.T) {
	if len(os.Args) < 3 || os.Args[len(os.Args)-2] != "--" {
		return
	}
	mode := os.Args[len(os.Args)-1]
	switch mode {
	case "mock_success":
		fmt.Print("success output")
		os.Exit(0)
	case "mock_empty":
		os.Exit(0)
	case "mock_stderr":
		fmt.Fprint(os.Stderr, "fatal server error")
		os.Exit(1)
	case "mock_exit_only":
		os.Exit(1)
	}
}
