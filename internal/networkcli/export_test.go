package networkcli

import (
	"github.com/kizio806/minecraft-network/internal/backup"
	"github.com/kizio806/minecraft-network/internal/compose"
)

var (
	addPath          = backup.AddPath
	addTree          = backup.AddTree
	decryptBackup    = backup.DecryptBackup
	fileSHA256       = backup.FileSHA256
	secureKeyFile    = backup.SecureKeyFile
	encryptBackup    = backup.EncryptBackup
	parseComposeRows = compose.ParseComposeRows
)
