package discord

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kizio806/minecraft-network/internal/i18n"
	"gopkg.in/yaml.v3"
)

// ServerTarget represents a backend server to query for status.
type ServerTarget struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// RolesConfig holds Discord role IDs.
type RolesConfig struct {
	LinkedRoleID string `yaml:"linked-role-id"`
	StaffRoleID  string `yaml:"staff-role-id"`
}

// DiscordSection holds Discord-specific options.
type DiscordSection struct {
	GuildID              string      `yaml:"guild-id"`
	StatusChannelID      string      `yaml:"status-channel-id"`
	StatusUpdateInterval int         `yaml:"status-update-interval"`
	ChatChannelID        string      `yaml:"chat-channel-id"`
	StaffChannelID       string      `yaml:"staff-channel-id"`
	Roles                RolesConfig `yaml:"roles"`
}

// NetworkSection holds basic network identity.
type NetworkSection struct {
	Name       string `yaml:"name"`
	Domain     string `yaml:"domain"`
	MaxPlayers int    `yaml:"max-players"`
}

// Config represents the complete Discord bot configuration.
type Config struct {
	Discord  DiscordSection `yaml:"discord"`
	Network  NetworkSection `yaml:"network"`
	Servers  []ServerTarget `yaml:"servers"`
	Messages map[string]string
}

// LoadConfig loads and parses the Discord bot configuration from a file.
// It expands environment variables and integrates with the project's i18n bundle.
func LoadConfig(configPath, localesDir, lang string) (*Config, error) {
	b, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read discord config: %w", err)
	}

	expanded := os.ExpandEnv(string(b))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal discord config: %w", err)
	}

	if cfg.Discord.StatusUpdateInterval <= 0 {
		cfg.Discord.StatusUpdateInterval = 30
	}
	if cfg.Network.MaxPlayers <= 0 {
		cfg.Network.MaxPlayers = 50
	}

	if localesDir != "" {
		projectDir := filepath.Dir(localesDir)
		msgs, err := i18n.LoadLocale(projectDir, lang)
		if err == nil {
			cfg.Messages = msgs
		}
	}
	if cfg.Messages == nil {
		cfg.Messages = make(map[string]string)
	}

	return &cfg, nil
}

// Msg returns the localized string for key, falling back to a formatted default.
func (c *Config) Msg(key, def string) string {
	if val, ok := c.Messages[strings.ToLower(key)]; ok && val != "" {
		return val
	}
	return def
}
