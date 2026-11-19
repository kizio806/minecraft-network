package networkcli

import (
	"github.com/kizio806/minecraft-network/internal/backup"
	"github.com/kizio806/minecraft-network/internal/config"
)

// Config represents operational paths and runtime parameters.
type Config = config.Config

// Load initializes and validates environment and configuration settings.
var Load = config.Load

// parseEnvValues parses key-value pairs from .env data.
var parseEnvValues = config.ParseEnvValues

func cleanupBackups(dir string, retention int) error {
	mgr := backup.NewManager(config.Config{BackupDir: dir}, nil, nil)
	return mgr.Cleanup(retention)
}
