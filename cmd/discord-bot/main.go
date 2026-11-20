package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/kizio806/minecraft-network/internal/discord"
)

var (
	newDiscordSession = func(token string) (*discordgo.Session, error) {
		return discordgo.New("Bot " + token)
	}
	openSession = func(s *discordgo.Session) error {
		return s.Open()
	}
)

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("discord-bot", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", "configs/discord/config.yml", "path to discord config file")
	localesDir := fs.String("locales", "locales", "path to locales directory")
	lang := fs.String("lang", "en", "language bundle to load (en, pl)")
	healthCheck := fs.Bool("health", false, "run healthcheck and exit")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *healthCheck {
		return 0
	}

	logger := log.New(stdout, "", log.LstdFlags)

	if envCfg := os.Getenv("DISCORD_CONFIG"); envCfg != "" {
		*configPath = envCfg
	}

	token := strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN"))
	if token == "" {
		if b, err := os.ReadFile("/run/secrets/discord_bot_token"); err == nil {
			token = strings.TrimSpace(string(b))
		} else if b, err := os.ReadFile("secrets/discord_bot_token"); err == nil {
			token = strings.TrimSpace(string(b))
		}
	}

	if token == "" || token == "placeholder_token_change_me" {
		logger.Println("[INFO] DISCORD_BOT_TOKEN is empty or placeholder. Bot is in standby mode. Set a valid token in secrets/discord_bot_token to enable.")
		<-ctx.Done()
		logger.Println("[INFO] Discord bot shutting down from standby.")
		return 0
	}

	cfg, err := discord.LoadConfig(*configPath, *localesDir, *lang)
	if err != nil {
		fmt.Fprintf(stderr, "[ERROR] Failed to load discord config: %v\n", err)
		return 1
	}

	var store discord.Store
	dbURL := strings.TrimSpace(cfg.Database.URL)
	if dbURL == "" {
		dbURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dbURL != "" {
		pgStore, err := discord.NewPostgresStore(dbURL)
		if err != nil {
			logger.Printf("[WARN] Failed to connect to PostgreSQL (%v). Falling back to in-memory store.", err)
			store = discord.NewMemoryStore()
		} else {
			logger.Println("[INFO] Connected to PostgreSQL for Discord persistent storage.")
			store = pgStore
		}
	} else {
		logger.Println("[INFO] No DATABASE_URL specified. Running with in-memory store.")
		store = discord.NewMemoryStore()
	}
	defer store.Close()

	session, err := newDiscordSession(token)
	if err != nil {
		fmt.Fprintf(stderr, "[ERROR] Failed to create Discord session: %v\n", err)
		return 1
	}

	executor := &discord.DockerConsoleExecutor{}
	bot := discord.NewBot(cfg, session, nil, store, executor)
	bot.RegisterHandlers()

	if err := openSession(session); err != nil {
		fmt.Fprintf(stderr, "[ERROR] Failed to open Discord connection: %v\n", err)
		return 1
	}
	defer session.Close()

	logger.Printf("[INFO] Discord bot connected as %s#%s", session.State.User.Username, session.State.User.Discriminator)

	commands := bot.SlashCommands()
	for _, cmd := range commands {
		_, err := session.ApplicationCommandCreate(session.State.User.ID, cfg.Discord.GuildID, cmd)
		if err != nil {
			logger.Printf("[WARN] Failed to register slash command %s: %v", cmd.Name, err)
		}
	}

	loopCtx, cancelLoop := context.WithCancel(ctx)
	defer cancelLoop()

	go bot.StartStatusLoop(loopCtx)

	<-ctx.Done()
	logger.Println("[INFO] Discord bot stopping gracefully...")
	return 0
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
