package backup_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kizio806/minecraft-network/internal/backup"
)

func TestCreateBackupArchive_AddTreeError(t *testing.T) {
	temp := t.TempDir()
	root := filepath.Join(temp, "project")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	// A self-referential symlink makes os.Stat fail with ELOOP, exercising
	// CreateBackupArchive's AddTree error return path.
	if err := os.Symlink("configs", filepath.Join(root, "configs")); err != nil {
		t.Fatal(err)
	}

	err := backup.CreateBackupArchive(root, filepath.Join(temp, "stage"), filepath.Join(temp, "backup.tar.gz"))
	if err == nil {
		t.Fatal("expected error for invalid configs path, got nil")
	}
}

func TestCreateBackupArchive_AddPathError(t *testing.T) {
	temp := t.TempDir()
	root := filepath.Join(temp, "project")
	stage := filepath.Join(temp, "stage")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(stage, 0700); err != nil {
		t.Fatal(err)
	}
	// A self-referential symlink makes os.Stat fail with ELOOP, exercising
	// CreateBackupArchive's AddPath error return path.
	if err := os.Symlink("postgres.dump", filepath.Join(stage, "postgres.dump")); err != nil {
		t.Fatal(err)
	}

	err := backup.CreateBackupArchive(root, stage, filepath.Join(temp, "backup.tar.gz"))
	if err == nil {
		t.Fatal("expected error for invalid staged dump path, got nil")
	}
}
