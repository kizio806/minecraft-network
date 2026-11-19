package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultInitConfig(t *testing.T) {
	cfg := DefaultInitConfig()
	if cfg.Name == "" || cfg.ShortName == "" || cfg.Domain == "" || cfg.MaxPlayers < 1 {
		t.Errorf("invalid default config: %+v", cfg)
	}
	if cfg.Locale != "en" {
		t.Errorf("expected default locale 'en', got %s", cfg.Locale)
	}
}

func TestRunWizardNonInteractive(t *testing.T) {
	cfg := DefaultInitConfig()
	cfg.NonInteractive = true

	if err := RunWizard(&cfg); err != nil {
		t.Fatalf("expected non-interactive wizard to succeed, got %v", err)
	}
}

func TestWriteNetworkConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configsDir := filepath.Join(tmpDir, "configs")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultInitConfig()
	cfg.Name = "Custom Network"
	cfg.Locale = "pl"

	if err := WriteNetworkConfig(tmpDir, cfg); err != nil {
		t.Fatalf("write network config failed: %v", err)
	}

	written, err := os.ReadFile(filepath.Join(configsDir, "network.yml"))
	if err != nil {
		t.Fatal(err)
	}

	if string(written) == "" {
		t.Fatal("expected non-empty network.yml")
	}
}
