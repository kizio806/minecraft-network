package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"gopkg.in/yaml.v3"
)

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00d2ff")).
			MarginBottom(1)

	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#2ecc71"))

	HeaderBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3a7bd5")).
			Padding(1, 2).
			MarginBottom(1)
)

// InitConfig holds user choices collected during interactive setup.
type InitConfig struct {
	Name           string
	ShortName      string
	Domain         string
	Discord        string
	Website        string
	MOTDLine1      string
	MOTDLine2      string
	MaxPlayers     int
	Locale         string
	ThemePreset    string
	NonInteractive bool
}

// DefaultInitConfig returns standard sensible defaults.
func DefaultInitConfig() InitConfig {
	return InitConfig{
		Name:        "Minecraft Network",
		ShortName:   "MC Network",
		Domain:      "example.com",
		Discord:     "discord.gg/example",
		Website:     "example.com",
		MOTDLine1:   "✦ MC NETWORK ✦",
		MOTDLine2:   "Modern Survival SMP 1.21.x",
		MaxPlayers:  50,
		Locale:      "en",
		ThemePreset: "amber-sunset",
	}
}

// RunWizard launches an interactive terminal form to configure network parameters.
func RunWizard(cfg *InitConfig) error {
	if cfg.NonInteractive || !term.IsTerminal(os.Stdin.Fd()) {
		return nil
	}

	banner := TitleStyle.Render("✦ MINECRAFT NETWORK — INITIALIZATION WIZARD ✦")
	fmt.Println(HeaderBox.Render(banner + "\n" +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#a8a8a8")).
			Render("Configure your server identity, branding, limits and active locale.")))

	maxPlayersStr := strconv.Itoa(cfg.MaxPlayers)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Network Display Name").
				Description("Public title displayed across menus and titles").
				Value(&cfg.Name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("name cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("Short Name / Tag").
				Description("Short prefix for chat and notifications").
				Value(&cfg.ShortName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("short name cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("Primary Domain").
				Description("Player connection address (e.g. mc.example.com)").
				Value(&cfg.Domain).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("domain cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("Discord Invite Link").
				Value(&cfg.Discord),

			huh.NewInput().
				Title("Website URL").
				Value(&cfg.Website),
		),

		huh.NewGroup(
			huh.NewInput().
				Title("Max Players Limit").
				Description("Network concurrency ceiling").
				Value(&maxPlayersStr).
				Validate(func(s string) error {
					v, err := strconv.Atoi(strings.TrimSpace(s))
					if err != nil || v < 1 {
						return fmt.Errorf("must be a positive number")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("Active Locale (In-Game Language)").
				Description("All player-facing menus, holograms and chat will use this language").
				Options(
					huh.NewOption("English (Default)", "en"),
					huh.NewOption("Polski (Polish)", "pl"),
				).
				Value(&cfg.Locale),

			huh.NewSelect[string]().
				Title("Color Theme Preset").
				Description("Semantic color palette for scoreboard, TAB list and rank tags").
				Options(
					huh.NewOption("Amber Sunset (Warm Amber & Coral)", "amber-sunset"),
					huh.NewOption("Cyber Blue (Neon Cyan & Electric Blue)", "cyber-blue"),
					huh.NewOption("Emerald Forest (Vibrant Emerald & Mint)", "emerald-forest"),
					huh.NewOption("Amethyst Purple (Deep Violet & Orchid)", "amethyst-purple"),
				).
				Value(&cfg.ThemePreset),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if val, err := strconv.Atoi(maxPlayersStr); err == nil && val > 0 {
		cfg.MaxPlayers = val
	}

	return nil
}

// WriteNetworkConfig saves the updated configuration to configs/network.yml.
func WriteNetworkConfig(projectDir string, cfg InitConfig) error {
	configPath := filepath.Join(projectDir, "configs", "network.yml")

	type networkFile struct {
		Network struct {
			Name       string `yaml:"name"`
			ShortName  string `yaml:"short-name"`
			Domain     string `yaml:"domain"`
			Discord    string `yaml:"discord"`
			Website    string `yaml:"website"`
			MOTDLine1  string `yaml:"motd-line-1"`
			MOTDLine2  string `yaml:"motd-line-2"`
			MaxPlayers int    `yaml:"max-players"`
		} `yaml:"network"`
		Locale string `yaml:"locale"`
		Theme  struct {
			Preset string `yaml:"preset"`
		} `yaml:"theme"`
		Lobby struct {
			DisplayName string `yaml:"display-name"`
		} `yaml:"lobby"`
		Auth struct {
			DisplayName string `yaml:"display-name"`
		} `yaml:"auth"`
	}

	var nf networkFile
	if b, err := os.ReadFile(configPath); err == nil {
		_ = yaml.Unmarshal(b, &nf)
	}

	nf.Network.Name = cfg.Name
	nf.Network.ShortName = cfg.ShortName
	nf.Network.Domain = cfg.Domain
	nf.Network.Discord = cfg.Discord
	nf.Network.Website = cfg.Website
	if nf.Network.MOTDLine1 == "" {
		nf.Network.MOTDLine1 = cfg.MOTDLine1
	}
	if nf.Network.MOTDLine2 == "" {
		nf.Network.MOTDLine2 = cfg.MOTDLine2
	}
	nf.Network.MaxPlayers = cfg.MaxPlayers
	nf.Locale = cfg.Locale
	nf.Theme.Preset = cfg.ThemePreset
	if nf.Lobby.DisplayName == "" {
		nf.Lobby.DisplayName = "Lobby"
	}
	if nf.Auth.DisplayName == "" {
		nf.Auth.DisplayName = "Auth"
	}

	out, err := yaml.Marshal(&nf)
	if err != nil {
		return fmt.Errorf("marshal configs/network.yml: %w", err)
	}

	header := []byte("# Public, non-secret network identity, theming, and localization.\n# Rendered into servers/ during './bin/network install'.\n\n")
	return os.WriteFile(configPath, append(header, out...), 0600)
}
