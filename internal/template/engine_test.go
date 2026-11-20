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

	t.Run("token replacement", func(t *testing.T) {
		input := "Welcome: __MSG_MOTD_WELCOME__ on __NETWORK_DOMAIN__ with secret __FORWARDING_SECRET__"
		got, err := engine.RenderString(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "Welcome: Welcome to TestNet! on test.example.com with secret super-secret-123"
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})

	t.Run("template syntax", func(t *testing.T) {
		input := "Hello {{ .Network.ShortName | upper }}! Max: {{ .Network.MaxPlayers }}"
		got, err := engine.RenderString(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "Hello TESTNET! Max: 100"
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})

	t.Run("unresolved placeholder rejection", func(t *testing.T) {
		input := "Unresolved: __MSG_UNKNOWN_PLACEHOLDER__"
		if _, err := engine.RenderString(input); err == nil {
			t.Fatalf("expected error for unresolved placeholder, got nil")
		}
	})

	t.Run("incomplete placeholder prefix", func(t *testing.T) {
		input := "Prefix only: __NETWORK_INCOMPLETE"
		if _, err := engine.RenderString(input); err == nil {
			t.Fatalf("expected error for incomplete placeholder, got nil")
		}
	})

	t.Run("parse error", func(t *testing.T) {
		input := "Bad template syntax: {{ if }}"
		if _, err := engine.RenderString(input); err == nil || !strings.Contains(err.Error(), "parse template") {
			t.Fatalf("expected parse template error, got: %v", err)
		}
	})

	t.Run("execution error", func(t *testing.T) {
		input := "Execution error: {{ .BadField }}"
		if _, err := engine.RenderString(input); err == nil || !strings.Contains(err.Error(), "execute template") {
			t.Fatalf("expected execute template error, got: %v", err)
		}
	})
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

	dstBinFile := filepath.Join(dstDir, "logo.png")
	bBin, err := os.ReadFile(dstBinFile)
	if err != nil {
		t.Fatalf("read dst binary error: %v", err)
	}
	if string(bBin) != string(binContent) {
		t.Error("binary file content changed during copy")
	}

	if err := engine.RenderTree(filepath.Join(tmpDir, "non-existent-src"), dstDir); err != nil {
		t.Fatalf("expected nil for non-existent source dir in RenderTree, got: %v", err)
	}

	if err := engine.RenderFile(filepath.Join(tmpDir, "missing.txt"), filepath.Join(dstDir, "out.txt")); err == nil {
		t.Fatal("expected error for missing source file in RenderFile, got nil")
	}

	badSrc := filepath.Join(srcDir, "bad.txt")
	_ = os.WriteFile(badSrc, []byte("Unresolved: __MSG_MISSING_KEY__"), 0640)
	if err := engine.RenderFile(badSrc, filepath.Join(dstDir, "bad.txt")); err == nil {
		t.Fatal("expected error for unresolved template in RenderFile, got nil")
	}
}
