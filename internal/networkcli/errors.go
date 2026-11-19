package networkcli

import (
	"errors"

	"github.com/kizio806/minecraft-network/internal/backup"
	"github.com/kizio806/minecraft-network/internal/config"
)

// Sentinel domain errors for network operations.
var (
	ErrChecksumMismatch     = errors.New("backup checksum mismatch")
	ErrUnsafeArchivePath     = backup.ErrUnsafeArchivePath
	ErrArchiveEntryTooLarge  = backup.ErrArchiveEntryTooLarge
	ErrSecretUnsafe         = errors.New("secret is missing, empty or has unsafe permissions")
	ErrServiceUnhealthy      = errors.New("one or more required services are not healthy")
	ErrLockContention       = config.ErrLockContention
	ErrUntrustedDownloadURL = errors.New("download URL is not HTTPS or trusted")
)

// ExitCode represents typed process exit status codes.
type ExitCode = int

const (
	ExitSuccess      ExitCode = 0
	ExitGeneralError ExitCode = 1
	ExitUsageError   ExitCode = 2
	ExitConfigError  ExitCode = 3
	ExitLockError    ExitCode = 4
	ExitDockerError  ExitCode = 5
	ExitRestoreError ExitCode = 8
)
