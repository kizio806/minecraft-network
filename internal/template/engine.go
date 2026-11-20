package template

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	texttemplate "text/template"

	"github.com/kizio806/minecraft-network/internal/fileutil"
	"github.com/kizio806/minecraft-network/internal/i18n"
)

// NetworkConfig represents network-level branding, identity and limits.
type NetworkConfig struct {
	Name           string `yaml:"name"`
	ShortName      string `yaml:"short-name"`
	Domain         string `yaml:"domain"`
	Discord        string `yaml:"discord"`
	Website        string `yaml:"website"`
	MOTDLine1      string `yaml:"motd-line-1"`
	MOTDLine2      string `yaml:"motd-line-2"`
	PrimaryColor   string `yaml:"primary-color"`
	SecondaryColor string `yaml:"secondary-color"`
	MaxPlayers     int    `yaml:"max-players"`
}

// Context holds all available data for template execution.
type Context struct {
	Network NetworkConfig
	Theme   i18n.ThemeColors
	Locale  string
	Msg     map[string]string
	Secrets map[string]string
}

// Engine compiles and renders templates using text/template and token expansion.
type Engine struct {
	Ctx Context
}

// NewEngine creates a new template engine with the given context.
func NewEngine(ctx Context) *Engine {
	return &Engine{Ctx: ctx}
}

// FuncMap provides custom helper functions to Go templates.
func (e *Engine) FuncMap() texttemplate.FuncMap {
	return texttemplate.FuncMap{
		"lower":   strings.ToLower,
		"upper":   strings.ToUpper,
		"trim":    strings.TrimSpace,
		"replace": strings.ReplaceAll,
		"quote":   strconv.Quote,
		"default": func(def, val string) string {
			if strings.TrimSpace(val) == "" {
				return def
			}
			return val
		},
		"msg": func(key string) string {
			if v, ok := e.Ctx.Msg[strings.ToLower(key)]; ok {
				return v
			}
			return "__MISSING_MSG_" + strings.ToUpper(key) + "__"
		},
	}
}

// TokenMap generates all standard substitution tokens for backward compatibility.
func (e *Engine) TokenMap() map[string]string {
	tokens := map[string]string{
		"__NETWORK_NAME__":            e.Ctx.Network.Name,
		"__NETWORK_SHORT_NAME__":      e.Ctx.Network.ShortName,
		"__NETWORK_DOMAIN__":          e.Ctx.Network.Domain,
		"__NETWORK_DISCORD__":         e.Ctx.Network.Discord,
		"__NETWORK_WEBSITE__":         e.Ctx.Network.Website,
		"__NETWORK_MOTD_LINE_1__":     e.Ctx.Network.MOTDLine1,
		"__NETWORK_MOTD_LINE_2__":     e.Ctx.Network.MOTDLine2,
		"__NETWORK_PRIMARY_COLOR__":   e.Ctx.Theme.Primary,
		"__NETWORK_SECONDARY_COLOR__": e.Ctx.Theme.Secondary,
		"__NETWORK_MAX_PLAYERS__":     strconv.Itoa(e.Ctx.Network.MaxPlayers),
		"__LOCALE__":                  e.Ctx.Locale,
		"__LOBBY_DISPLAY_NAME__":      "Lobby",
		"__AUTH_DISPLAY_NAME__":       "Auth",
	}

	for k, v := range i18n.TemplateVariables(e.Ctx.Theme, e.Ctx.Msg) {
		tokens[k] = v
	}

	for k, v := range e.Ctx.Secrets {
		tokens[k] = v
	}

	return tokens
}

// RenderString renders a template string by expanding token placeholders,
// executing Go text/template syntax, and validating that no unresolved placeholders remain.
func (e *Engine) RenderString(content string) (string, error) {
	tokens := e.TokenMap()
	for pass := 0; pass < 5; pass++ {
		changed := false
		for from, to := range tokens {
			if strings.Contains(content, from) {
				content = strings.ReplaceAll(content, from, to)
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	if strings.Contains(content, "{{") && strings.Contains(content, "}}") {
		tmpl, err := texttemplate.New("config").Funcs(e.FuncMap()).Parse(content)
		if err != nil {
			return "", fmt.Errorf("parse template: %w", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, e.Ctx); err != nil {
			return "", fmt.Errorf("execute template: %w", err)
		}
		content = buf.String()
	}

	for _, prefix := range []string{"__NETWORK_", "__LOBBY_", "__AUTH_", "__FORWARDING_SECRET__", "__THEME_", "__MSG_", "__MISSING_"} {
		if idx := strings.Index(content, prefix); idx != -1 {
			end := strings.Index(content[idx+len(prefix):], "__")
			token := prefix
			if end != -1 {
				token = content[idx : idx+len(prefix)+end+2]
			}
			return "", fmt.Errorf("unresolved configuration placeholder %s", token)
		}
	}

	return content, nil
}

// IsBinaryFile returns true for binary assets that should not be parsed as text.
func IsBinaryFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".ico", ".jar", ".zip", ".gz", ".age", ".tar", ".dat", ".bin":
		return true
	default:
		return false
	}
}

// RenderFile reads source, renders template, and writes destination.
func (e *Engine) RenderFile(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0750); err != nil {
		return err
	}

	if IsBinaryFile(source) {
		return fileutil.CopyFile(source, target, 0640)
	}

	b, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	rendered, err := e.RenderString(string(b))
	if err != nil {
		return fmt.Errorf("render %s: %w", source, err)
	}

	return os.WriteFile(target, []byte(rendered), 0600)
}

// RenderTree walks source directory and renders all files into target directory.
func (e *Engine) RenderTree(source, target string) error {
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, rel)
		return e.RenderFile(path, dest)
	})
}
