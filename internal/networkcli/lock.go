package networkcli

import (
	"github.com/kizio806/minecraft-network/internal/config"
)

// OperationLock handles exclusive file flock locking.
type OperationLock = config.OperationLock

// AcquireLock requests an exclusive flock lock on .network-operation.lock.
var AcquireLock = config.AcquireLock
