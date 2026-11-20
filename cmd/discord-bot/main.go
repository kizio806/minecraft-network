package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/kizio806/minecraft-network/internal/discord"
)

func main() {
	configPath := flag.String("config", "configs/discord/config.yml", "path to discord config file")
	localesDir := flag.String("locales", "locales", "path to locales directory")
	lang := flag.String("lang", "en", "language bundle to load (en, pl)")
	healthCheck := flag.Bool("health", false, "run healthcheck and exit")
	flag.Parse()

	if *healthCheck {
		os.Exit(0)
	}

	if envCfg := os.Getenv("DISCORD_CONFIG"); envCfg != "" {
		*configPath = envCfg
	}

	token := strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN"))
	if token == "" {
		// Try reading from Docker secret
		if b, err := os.ReadFile("/run/secrets/discord_bot_token"); err == nil {
			token = strings.TrimSpace(string(b))
		} else if b, err := os.ReadFile("secrets/discord_bot_token"); err == nil {
			token = strings.TrimSpace(string(b))
		}
	}

	if token == "" || token == "placeholder_token_change_me" {
		log.Println("[INFO] DISCORD_BOT_TOKEN is empty or placeholder. Bot is in standby mode. Set a valid token in secrets/discord_bot_token to enable.")
		// Wait for signal so container doesn't crash-loop in development
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("[INFO] Discord bot shutting down.")
		return
	}

	cfg, err := discord.LoadConfig(*configPath, *localesDir, *lang)
	if err != nil {
		log.Fatalf("[ERROR] Failed to load discord config: %v", err)
	}

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("[ERROR] Failed to create Discord session: %v", err)
	}

	bot := discord.NewBot(cfg, session, nil, nil)
	bot.RegisterHandlers()

	if err := session.Open(); err != nil {
		log.Fatalf("[ERROR] Failed to open Discord connection: %v", err)
	}
	defer session.Close()

	log.Printf("[INFO] Discord bot connected as %s#%s", session.State.User.Username, session.State.User.Discriminator)

	// Register slash commands
	commands := bot.SlashCommands()
	for _, cmd := range commands {
		_, err := session.ApplicationCommandCreate(session.State.User.ID, cfg.Discord.GuildID, cmd)
		if err != nil {
			log.Printf("[WARN] Failed to register slash command %s: %v", cmd.Name, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go bot.StartStatusLoop(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("[INFO] Discord bot stopping gracefully...")
	cancel()
	fmt.Println("Discord bot stopped.")
}
