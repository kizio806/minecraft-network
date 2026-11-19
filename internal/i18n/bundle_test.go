package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLocaleAndFallback(t *testing.T) {
	tmp := t.TempDir()
	localesDir := filepath.Join(tmp, "locales")
	if err := os.MkdirAll(localesDir, 0750); err != nil {
		t.Fatal(err)
	}

	enContent := []byte("locale: en\nmessages:\n  hello: \"Hello\"\n  world: \"World\"\n")
	plContent := []byte("locale: pl\nmessages:\n  hello: \"Cześć\"\n")

	if err := os.WriteFile(filepath.Join(localesDir, "en.yml"), enContent, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localesDir, "pl.yml"), plContent, 0600); err != nil {
		t.Fatal(err)
	}

	// Test English
	enMsgs, err := LoadLocale(tmp, "en")
	if err != nil {
		t.Fatalf("unexpected error loading en: %v", err)
	}
	if enMsgs["hello"] != "Hello" || enMsgs["world"] != "World" {
		t.Errorf("expected Hello and World, got %v", enMsgs)
	}

	// Test empty lang defaults to en
	emptyMsgs, err := LoadLocale(tmp, "")
	if err != nil || emptyMsgs["hello"] != "Hello" {
		t.Fatalf("expected empty lang to load en: %v", err)
	}

	// Test Polish with English fallback for missing key 'world'
	plMsgs, err := LoadLocale(tmp, "pl")
	if err != nil {
		t.Fatalf("unexpected error loading pl: %v", err)
	}
	if plMsgs["hello"] != "Cześć" {
		t.Errorf("expected Cześć, got %s", plMsgs["hello"])
	}
	if plMsgs["world"] != "World" {
		t.Errorf("expected fallback World, got %s", plMsgs["world"])
	}

	// Test missing locale defaults to fallback
	missingMsgs, err := LoadLocale(tmp, "de")
	if err != nil {
		t.Fatalf("unexpected error for missing locale with fallback: %v", err)
	}
	if missingMsgs["hello"] != "Hello" {
		t.Errorf("expected fallback Hello, got %s", missingMsgs["hello"])
	}

	// Test invalid YAML in target locale
	_ = os.WriteFile(filepath.Join(localesDir, "bad.yml"), []byte("invalid: [unclosed"), 0600)
	if _, err := LoadLocale(tmp, "bad"); err == nil || !strings.Contains(err.Error(), "invalid YAML") {
		t.Fatalf("expected invalid YAML error, got: %v", err)
	}

	// Test missing target locale when NO fallback exists
	emptyDir := t.TempDir()
	if _, err := LoadLocale(emptyDir, "fr"); err == nil || !strings.Contains(err.Error(), "could not read locale file") {
		t.Fatalf("expected error when neither target nor fallback exists, got: %v", err)
	}
}

func TestResolveThemeAndVariables(t *testing.T) {
	// Empty preset name defaults to amber-sunset
	defTheme := ResolveTheme("", nil)
	if defTheme.Primary != DefaultThemes["amber-sunset"].Primary {
		t.Errorf("expected amber-sunset default for empty name, got: %s", defTheme.Primary)
	}

	// Standard presets
	for _, name := range []string{"amber-sunset", "cyber-blue", "emerald-forest", "amethyst-purple"} {
		theme := ResolveTheme(name, nil)
		if theme.Primary != DefaultThemes[name].Primary {
			t.Errorf("expected primary for %s: %s, got %s", name, DefaultThemes[name].Primary, theme.Primary)
		}
	}

	// Unknown preset name defaults to amber-sunset
	unknown := ResolveTheme("unknown-preset", nil)
	if unknown.Primary != DefaultThemes["amber-sunset"].Primary {
		t.Errorf("expected amber-sunset for unknown preset, got: %s", unknown.Primary)
	}

	// Custom preset with partial fields completed
	custom := map[string]ThemeColors{
		"my-custom": {
			Primary: "#112233",
		},
		"empty-custom": {
			Primary: "",
		},
	}
	resolvedCustom := ResolveTheme("my-custom", custom)
	if resolvedCustom.Primary != "#112233" {
		t.Errorf("expected #112233, got %s", resolvedCustom.Primary)
	}
	if resolvedCustom.Secondary != DefaultThemes["amber-sunset"].Secondary {
		t.Errorf("expected fallback secondary, got %s", resolvedCustom.Secondary)
	}
	if resolvedCustom.Accent != DefaultThemes["amber-sunset"].Accent {
		t.Errorf("expected fallback accent, got %s", resolvedCustom.Accent)
	}
	if resolvedCustom.Danger != DefaultThemes["amber-sunset"].Danger {
		t.Errorf("expected fallback danger, got %s", resolvedCustom.Danger)
	}
	if resolvedCustom.Success != DefaultThemes["amber-sunset"].Success {
		t.Errorf("expected fallback success, got %s", resolvedCustom.Success)
	}
	if resolvedCustom.Gold != DefaultThemes["amber-sunset"].Gold {
		t.Errorf("expected fallback gold, got %s", resolvedCustom.Gold)
	}
	if resolvedCustom.Muted != DefaultThemes["amber-sunset"].Muted {
		t.Errorf("expected fallback muted, got %s", resolvedCustom.Muted)
	}

	// Custom preset with empty Primary falls back
	emptyCustom := ResolveTheme("empty-custom", custom)
	if emptyCustom.Primary != DefaultThemes["amber-sunset"].Primary {
		t.Errorf("expected default theme when custom Primary is empty, got: %s", emptyCustom.Primary)
	}

	vars := TemplateVariables(defTheme, map[string]string{"test_msg": "Sample"})
	if vars["__THEME_PRIMARY__"] != defTheme.Primary {
		t.Errorf("expected __THEME_PRIMARY__, got %s", vars["__THEME_PRIMARY__"])
	}
	if vars["__THEME_SECONDARY__"] != defTheme.Secondary {
		t.Errorf("expected __THEME_SECONDARY__, got %s", vars["__THEME_SECONDARY__"])
	}
	if vars["__THEME_ACCENT__"] != defTheme.Accent {
		t.Errorf("expected __THEME_ACCENT__, got %s", vars["__THEME_ACCENT__"])
	}
	if vars["__THEME_DANGER__"] != defTheme.Danger {
		t.Errorf("expected __THEME_DANGER__, got %s", vars["__THEME_DANGER__"])
	}
	if vars["__THEME_SUCCESS__"] != defTheme.Success {
		t.Errorf("expected __THEME_SUCCESS__, got %s", vars["__THEME_SUCCESS__"])
	}
	if vars["__THEME_GOLD__"] != defTheme.Gold {
		t.Errorf("expected __THEME_GOLD__, got %s", vars["__THEME_GOLD__"])
	}
	if vars["__THEME_MUTED__"] != defTheme.Muted {
		t.Errorf("expected __THEME_MUTED__, got %s", vars["__THEME_MUTED__"])
	}
	if vars["__MSG_TEST_MSG__"] != "Sample" {
		t.Errorf("expected __MSG_TEST_MSG__, got %s", vars["__MSG_TEST_MSG__"])
	}
}
