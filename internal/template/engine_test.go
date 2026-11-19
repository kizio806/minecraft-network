package template

import (
	"os"
	"path/filepath"
	"strings"
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

	// Test unresolved prefix without closing __
	input3b := "Prefix only: __NETWORK_INCOMPLETE"
	if _, err := engine.RenderString(input3b); err == nil {
		t.Fatalf("expected error for incomplete placeholder, got nil")
	}

	// Test template parse error
	input4 := "Bad template syntax: {{ if }}"
	if _, err := engine.RenderString(input4); err == nil || !strings.Contains(err.Error(), "parse template") {
		t.Fatalf("expected parse template error, got: %v", err)
	}

	// Test template execution error
	input5 := "Execution error: {{ .BadField }}"
	if _, err := engine.RenderString(input5); err == nil || !strings.Contains(err.Error(), "execute template") {
		t.Fatalf("expected execute template error, got: %v", err)
	}
}

func TestEngineFuncMap(t *testing.T) {
	ctx := Context{
		Network: NetworkConfig{
			Name: "DefaultNet",
		},
		Msg: map[string]string{
			"greeting": "Hello Player!",
		},
	}
	engine := NewEngine(ctx)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{"lower", `{{ "HELLO WORLD" | lower }}`, "hello world"},
		{"upper", `{{ "hello world" | upper }}`, "HELLO WORLD"},
		{"trim", `{{ "   spaced string   " | trim }}`, "spaced string"},
		{"replace", `{{ replace "cat sat on mat" "at" "ar" }}`, "car sar on mar"},
		{"quote", `{{ quote "quote-me" }}`, `"quote-me"`},
		{"default_empty", `{{ default "fallback" "" }}`, "fallback"},
		{"default_whitespace", `{{ default "fallback" "   " }}`, "fallback"},
		{"default_present", `{{ default "fallback" "actual" }}`, "actual"},
		{"msg_present", `{{ msg "greeting" }}`, "Hello Player!"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := engine.RenderString(tc.template)
			if err != nil {
				t.Fatalf("unexpected render error for %s: %v", tc.name, err)
			}
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}

	// Test msg function with missing key
	funcs := engine.FuncMap()
	msgFn := funcs["msg"].(func(string) string)
	missing := msgFn("non_existent_key")
	if missing != "__MISSING_MSG_NON_EXISTENT_KEY__" {
		t.Errorf("unexpected missing key token: %s", missing)
	}
}

func TestIsBinaryFile(t *testing.T) {
	binaryExts := []string{
		"server-icon.png", "image.jpg", "photo.jpeg", "anim.gif", "favicon.ico",
		"plugin.jar", "bundle.zip", "archive.tar.gz", "backup.age", "data.tar",
		"world.dat", "binary.bin",
	}
	for _, f := range binaryExts {
		if !IsBinaryFile(f) {
			t.Errorf("expected %s to be recognized as binary", f)
		}
	}

	textExts := []string{
		"config.yml", "settings.yaml", "info.txt", "velocity.toml", "stats.json", "README.md",
	}
	for _, f := range textExts {
		if IsBinaryFile(f) {
			t.Errorf("expected %s NOT to be recognized as binary", f)
		}
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

	// Also put a binary file in srcDir
	binFile := filepath.Join(srcDir, "logo.png")
	binContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if err := os.WriteFile(binFile, binContent, 0640); err != nil {
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

	// Verify binary file was copied as-is
	dstBinFile := filepath.Join(dstDir, "logo.png")
	bBin, err := os.ReadFile(dstBinFile)
	if err != nil {
		t.Fatalf("read dst binary error: %v", err)
	}
	if string(bBin) != string(binContent) {
		t.Error("binary file content changed during copy")
	}

	// Test RenderTree with non-existent source directory
	if err := engine.RenderTree(filepath.Join(tmpDir, "non-existent-src"), dstDir); err != nil {
		t.Fatalf("expected nil for non-existent source dir in RenderTree, got: %v", err)
	}

	// Test RenderFile errors
	if err := engine.RenderFile(filepath.Join(tmpDir, "missing.txt"), filepath.Join(dstDir, "out.txt")); err == nil {
		t.Fatal("expected error for missing source file in RenderFile, got nil")
	}

	// Test RenderFile template error in source file
	badSrc := filepath.Join(srcDir, "bad.txt")
	_ = os.WriteFile(badSrc, []byte("Unresolved: __MSG_MISSING_KEY__"), 0640)
	if err := engine.RenderFile(badSrc, filepath.Join(dstDir, "bad.txt")); err == nil {
		t.Fatal("expected error for unresolved template in RenderFile, got nil")
	}
}
