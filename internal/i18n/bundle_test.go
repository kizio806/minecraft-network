package i18n

import (
	"os"
	"path/filepath"
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
}

func TestResolveThemeAndVariables(t *testing.T) {
	theme := ResolveTheme("cyber-blue", nil)
	if theme.Primary != "#00d2ff" {
		t.Errorf("expected cyber-blue primary #00d2ff, got %s", theme.Primary)
	}

	custom := map[string]ThemeColors{
		"my-custom": {
			Primary: "#112233",
		},
	}
	resolvedCustom := ResolveTheme("my-custom", custom)
	if resolvedCustom.Primary != "#112233" {
		t.Errorf("expected #112233, got %s", resolvedCustom.Primary)
	}
	// Fallback field
	if resolvedCustom.Secondary == "" {
		t.Errorf("expected non-empty fallback secondary")
	}

	vars := TemplateVariables(theme, map[string]string{"test_msg": "Sample"})
	if vars["__THEME_PRIMARY__"] != "#00d2ff" {
		t.Errorf("expected __THEME_PRIMARY__, got %s", vars["__THEME_PRIMARY__"])
	}
	if vars["__MSG_TEST_MSG__"] != "Sample" {
		t.Errorf("expected __MSG_TEST_MSG__, got %s", vars["__MSG_TEST_MSG__"])
	}
}
