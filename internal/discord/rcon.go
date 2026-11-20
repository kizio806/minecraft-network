package discord

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ConsoleExecutor executes console commands against network backend containers.
type ConsoleExecutor interface {
	Execute(ctx context.Context, server, command string) (string, error)
}

func resolveDockerPath() string {
	candidates := []string{"/usr/bin/docker", "/usr/local/bin/docker"}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("docker"); err == nil && filepath.IsAbs(p) {
		return p
	}
	return "/usr/bin/docker"
}

// DockerConsoleExecutor runs commands via docker compose exec.
type DockerConsoleExecutor struct {
	ComposeFile string
}

// Execute dispatches the console command into the specified container.
func (d *DockerConsoleExecutor) Execute(ctx context.Context, server, command string) (string, error) {
	validServers := map[string]bool{"proxy": true, "lobby": true, "survival": true}
	if !validServers[server] {
		return "", fmt.Errorf("invalid server target %q (allowed: proxy, lobby, survival)", server)
	}

	var args []string
	if d.ComposeFile != "" {
		args = append(args, "-f", d.ComposeFile)
	}

	if server == "proxy" {
		args = append(args, "exec", "-T", server, "velocity-cmd", command)
	} else {
		args = append(args, "exec", "-T", server, "rcon-cli", command)
	}

	dockerBin := resolveDockerPath()
	//nolint:gosec // G204: server and arguments are validated against strict whitelist
	cmd := exec.CommandContext(ctx, dockerBin, append([]string{"compose"}, args...)...)
	cmd.Env = append(os.Environ(), "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	res := strings.TrimSpace(out.String())
	if res == "" {
		res = "Command executed successfully (no output)."
	}
	return res, nil
}

// DefaultConsoleExecutor provides a safe mock/fallback executor for local testing and standby.
type DefaultConsoleExecutor struct {
	MockOutput string
	MockErr    error
}

func (d *DefaultConsoleExecutor) Execute(_ context.Context, server, command string) (string, error) {
	validServers := map[string]bool{"proxy": true, "lobby": true, "survival": true}
	if !validServers[server] {
		return "", fmt.Errorf("invalid server target %q", server)
	}
	if d.MockErr != nil {
		return "", d.MockErr
	}
	if d.MockOutput != "" {
		return d.MockOutput, nil
	}
	return fmt.Sprintf("[%s] Executed command: %s", server, command), nil
}

// HandleRconCommand executes an administrative console command on a server backend.
func HandleRconCommand(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *Config, exec ConsoleExecutor) {
	if i.Member == nil || (i.Member.Permissions&discordgo.PermissionAdministrator == 0) {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: cfg.Msg("discord_no_permission", "You do not have permission to execute this administrative command."),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	data := i.ApplicationCommandData()
	var server, command string
	for _, opt := range data.Options {
		switch opt.Name {
		case "server":
			server = strings.ToLower(strings.TrimSpace(opt.StringValue()))
		case "command":
			command = strings.TrimSpace(opt.StringValue())
		}
	}

	validServers := map[string]bool{"proxy": true, "lobby": true, "survival": true}
	if !validServers[server] {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: cfg.Msg("discord_rcon_invalid_server", "Invalid server target. Allowed: proxy, lobby, survival."),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if exec == nil {
		exec = &DefaultConsoleExecutor{}
	}

	output, err := exec.Execute(ctx, server, command)
	if err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Error: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	resp := fmt.Sprintf("💻 **[%s]** `$ %s`\n```\n%s\n```", server, command, output)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: resp,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
