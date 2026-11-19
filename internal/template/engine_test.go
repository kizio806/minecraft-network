package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kizio806/minecraft-network/internal/i18n"
)

func TestEngineRenderString(t *testing.T) {
	ctx := Context{
		Network: NetworkConfig{
			Name:       "Test Network",
			ShortName:  "TestNet",
			Domain:     "test.example.com",
			MaxPlayers: 100,
		},
		Theme: i18n.ThemeColors{
			Primary:   "#ff0000",
			Secondary: "#00ff00",
		},
		Locale: "en",
		Msg: map[string]string{
			"motd_welcome": "Welcome to TestNet!",
		},
		Secrets: map[string]string{
			"__FORWARDING_SECRET__": "super-secret-123",
		},
	}

	engine := NewEngine(ctx)

	// Test token replacement
	input1 := "Welcome: __MSG_MOTD_WELCOME__ on __NETWORK_DOMAIN__ with secret __FORWARDING_SECRET__"
	got1, err := engine.RenderString(input1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected1 := "Welcome: Welcome to TestNet! on test.example.com with secret super-secret-123"
	if got1 != expected1 {
		t.Errorf("got %q, want %q", got1, expected1)
	}

	// Test Go template syntax
	input2 := "Hello {{ .Network.ShortName | upper }}! Max: {{ .Network.MaxPlayers }}"
	got2, err := engine.RenderString(input2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected2 := "Hello TESTNET! Max: 100"
	if got2 != expected2 {
		t.Errorf("got %q, want %q", got2, expected2)
	}

	// Test unresolved placeholder rejection
	input3 := "Unresolved: __MSG_UNKNOWN_PLACEHOLDER__"
	if _, err := engine.RenderString(input3); err == nil {
		t.Fatalf("expected error for unresolved placeholder, got nil")
	}
}

func TestEngineRenderFileAndTree(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	if err := os.MkdirAll(srcDir, 0750); err != nil {
		t.Fatal(err)
	}

	srcFile := filepath.Join(srcDir, "test.txt")
	if err := os.WriteFile(srcFile, []byte("Name: {{ .Network.Name }}"), 0640); err != nil {
		t.Fatal(err)
	}

	engine := NewEngine(Context{
		Network: NetworkConfig{Name: "Rendered Name"},
		Theme:   i18n.DefaultThemes["amber-sunset"],
		Msg:     map[string]string{},
	})

	if err := engine.RenderTree(srcDir, dstDir); err != nil {
		t.Fatalf("render tree error: %v", err)
	}

	dstFile := filepath.Join(dstDir, "test.txt")
	b, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read dst error: %v", err)
	}
	if string(b) != "Name: Rendered Name" {
		t.Errorf("got %q, want %q", string(b), "Name: Rendered Name")
	}
}
