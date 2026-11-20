package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ThemeColors represents standard semantic color tokens used across configs.
type ThemeColors struct {
	Primary   string `yaml:"primary"`
	Secondary string `yaml:"secondary"`
	Accent    string `yaml:"accent"`
	Danger    string `yaml:"danger"`
	Success   string `yaml:"success"`
	Gold      string `yaml:"gold"`
	Muted     string `yaml:"muted"`
}

// DefaultThemes provides standard fallback palettes.
var DefaultThemes = map[string]ThemeColors{
	"amber-sunset": {
		Primary:   "#f6d365",
		Secondary: "#fda085",
		Accent:    "#ff7e5f",
		Danger:    "#c9453e",
		Success:   "#6fc93e",
		Gold:      "#FFD700",
		Muted:     "#aaaaaa",
	},
	"cyber-blue": {
		Primary:   "#00d2ff",
		Secondary: "#3a7bd5",
		Accent:    "#00f2fe",
		Danger:    "#ff416c",
		Success:   "#00b09b",
		Gold:      "#f9d423",
		Muted:     "#aaaaaa",
	},
	"emerald-forest": {
		Primary:   "#2ecc71",
		Secondary: "#27ae60",
		Accent:    "#a8ff78",
		Danger:    "#e74c3c",
		Success:   "#2ecc71",
		Gold:      "#f1c40f",
		Muted:     "#aaaaaa",
	},
	"amethyst-purple": {
		Primary:   "#a18cd1",
		Secondary: "#fbc2eb",
		Accent:    "#8e44ad",
		Danger:    "#e74c3c",
		Success:   "#2ecc71",
		Gold:      "#f39c12",
		Muted:     "#aaaaaa",
	},
}

// LocaleFile represents the YAML structure of a language file.
type LocaleFile struct {
	Locale   string            `yaml:"locale"`
	Messages map[string]string `yaml:"messages"`
}

// LoadLocale loads messages for the specified locale, falling back to 'en' for missing values.
func LoadLocale(projectDir, lang string) (map[string]string, error) {
	if lang == "" {
		lang = "en"
	}

	result := make(map[string]string)

	if lang != "en" {
		enPath := filepath.Join(projectDir, "locales", "en.yml")
		if b, err := os.ReadFile(enPath); err == nil {
			var enFile LocaleFile
			if err := yaml.Unmarshal(b, &enFile); err == nil {
				for k, v := range enFile.Messages {
					result[k] = v
				}
			}
		}
	}

	targetPath := filepath.Join(projectDir, "locales", lang+".yml")
	b, err := os.ReadFile(targetPath)
	if err != nil {
		if len(result) > 0 {
			return result, nil
		}
		return nil, fmt.Errorf("could not read locale file %s: %w", targetPath, err)
	}

	var targetFile LocaleFile
	if err := yaml.Unmarshal(b, &targetFile); err != nil {
		return nil, fmt.Errorf("invalid YAML in locale file %s: %w", targetPath, err)
	}

	for k, v := range targetFile.Messages {
		result[k] = v
	}

	return result, nil
}

// ResolveTheme returns the ThemeColors for the requested preset name.
func ResolveTheme(presetName string, customPresets map[string]ThemeColors) ThemeColors {
	if presetName == "" {
		presetName = "amber-sunset"
	}
	if customPresets != nil {
		if t, ok := customPresets[presetName]; ok && t.Primary != "" {
			return completeTheme(t)
		}
	}
	if def, ok := DefaultThemes[presetName]; ok {
		return def
	}
	return DefaultThemes["amber-sunset"]
}

func completeTheme(t ThemeColors) ThemeColors {
	def := DefaultThemes["amber-sunset"]
	if t.Primary == "" {
		t.Primary = def.Primary
	}
	if t.Secondary == "" {
		t.Secondary = def.Secondary
	}
	if t.Accent == "" {
		t.Accent = def.Accent
	}
	if t.Danger == "" {
		t.Danger = def.Danger
	}
	if t.Success == "" {
		t.Success = def.Success
	}
	if t.Gold == "" {
		t.Gold = def.Gold
	}
	if t.Muted == "" {
		t.Muted = def.Muted
	}
	return t
}

// TemplateVariables generates all substitution placeholders for theming and messages.
func TemplateVariables(theme ThemeColors, messages map[string]string) map[string]string {
	vars := map[string]string{
		"__THEME_PRIMARY__":   theme.Primary,
		"__THEME_SECONDARY__": theme.Secondary,
		"__THEME_ACCENT__":    theme.Accent,
		"__THEME_DANGER__":    theme.Danger,
		"__THEME_SUCCESS__":   theme.Success,
		"__THEME_GOLD__":      theme.Gold,
		"__THEME_MUTED__":     theme.Muted,
	}

	for k, v := range messages {
		token := "__MSG_" + strings.ToUpper(k) + "__"
		vars[token] = v
	}

	return vars
}
