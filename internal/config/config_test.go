package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kizio806/minecraft-network/internal/config"
)

func TestConfigLoadAndPaths(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte("services: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(dir, "table")
	if err != nil {
		t.Fatalf("unexpected Load error: %v", err)
	}

	if cfg.ComposeFile() != composePath {
		t.Errorf("expected compose file %s, got %s", composePath, cfg.ComposeFile())
	}
	if cfg.PluginManagerManifest() != filepath.Join(dir, "configs/plugins/manifest.yml") {
		t.Errorf("unexpected manifest path: %s", cfg.PluginManagerManifest())
	}
}

func TestConfigLock(t *testing.T) {
	dir := t.TempDir()
	lock1, err := config.AcquireLock(dir)
	if err != nil {
		t.Fatalf("unexpected lock error: %v", err)
	}
	defer lock1.Close()

	_, err = config.AcquireLock(dir)
	if !errors.Is(err, config.ErrLockContention) {
		t.Fatalf("expected ErrLockContention, got %v", err)
	}

	if err := lock1.Close(); err != nil {
		t.Fatalf("unexpected unlock error: %v", err)
	}

	lock2, err := config.AcquireLock(dir)
	if err != nil {
		t.Fatalf("unexpected re-lock error: %v", err)
	}
	_ = lock2.Close()
}
