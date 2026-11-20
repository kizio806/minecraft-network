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

	cfg2 := DefaultInitConfig()
	cfg2.NonInteractive = false
	if err := RunWizard(&cfg2); err != nil {
		t.Fatalf("expected non-terminal wizard to return nil, got %v", err)
	}
}

func TestValidators(t *testing.T) {
	t.Run("ValidateName", func(t *testing.T) {
		if err := ValidateName(""); err == nil {
			t.Error("expected error for empty name")
		}
		if err := ValidateName("   "); err == nil {
			t.Error("expected error for whitespace name")
		}
		if err := ValidateName("My Network"); err != nil {
			t.Errorf("expected valid name to succeed, got: %v", err)
		}
	})

	t.Run("ValidateShortName", func(t *testing.T) {
		if err := ValidateShortName(""); err == nil {
			t.Error("expected error for empty short name")
		}
		if err := ValidateShortName("   "); err == nil {
			t.Error("expected error for whitespace short name")
		}
		if err := ValidateShortName("MN"); err != nil {
			t.Errorf("expected valid short name to succeed, got: %v", err)
		}
	})

	t.Run("ValidateDomain", func(t *testing.T) {
		if err := ValidateDomain(""); err == nil {
			t.Error("expected error for empty domain")
		}
		if err := ValidateDomain("   "); err == nil {
			t.Error("expected error for whitespace domain")
		}
		if err := ValidateDomain("mc.example.com"); err != nil {
			t.Errorf("expected valid domain to succeed, got: %v", err)
		}
	})

	t.Run("ValidateMaxPlayers", func(t *testing.T) {
		if err := ValidateMaxPlayers(""); err == nil {
			t.Error("expected error for empty max players")
		}
		if err := ValidateMaxPlayers("abc"); err == nil {
			t.Error("expected error for non-numeric max players")
		}
		if err := ValidateMaxPlayers("0"); err == nil {
			t.Error("expected error for 0 max players")
		}
		if err := ValidateMaxPlayers("-5"); err == nil {
			t.Error("expected error for negative max players")
		}
		if err := ValidateMaxPlayers(" 100 "); err != nil {
			t.Errorf("expected valid max players to succeed, got: %v", err)
		}
	})
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

	cfg.ShortName = "UpdatedTag"
	cfg.MaxPlayers = 250
	if err := WriteNetworkConfig(tmpDir, cfg); err != nil {
		t.Fatalf("second write network config failed: %v", err)
	}

	updated, err := os.ReadFile(filepath.Join(configsDir, "network.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(configsDir) {
		t.Fatal("expected absolute path")
	}
	if len(updated) == 0 {
		t.Fatal("expected non-empty updated content")
	}

	if err := WriteNetworkConfig(filepath.Join(tmpDir, "non_existent_folder"), cfg); err == nil {
		t.Fatal("expected error when writing to non-existent directory without configs subfolder")
	}
}

func TestBuildWizardForm(t *testing.T) {
	cfg := DefaultInitConfig()
	maxPlayers := "150"
	form := BuildWizardForm(&cfg, &maxPlayers)
	if form == nil {
		t.Fatal("expected non-nil form from BuildWizardForm")
	}
}
