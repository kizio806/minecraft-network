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

	// The self-referential symlink makes os.Stat fail with ELOOP.
	if err := os.Symlink("configs", filepath.Join(root, "configs")); err != nil {
		t.Fatal(err)
	}

	tmpFile := filepath.Join(temp, "backup.tar.gz.tmp")
	if err := backup.CreateBackupArchive(root, filepath.Join(temp, "stage"), filepath.Join(temp, "backup.tar.gz")); err == nil {
		t.Fatal("expected error for invalid configs path, got nil")
	}
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Fatalf("expected temporary file %s to be cleaned up on error, got err: %v", tmpFile, err)
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

	// The self-referential symlink makes os.Stat fail with ELOOP.
	if err := os.Symlink("postgres.dump", filepath.Join(stage, "postgres.dump")); err != nil {
		t.Fatal(err)
	}

	if err := backup.CreateBackupArchive(root, stage, filepath.Join(temp, "backup.tar.gz")); err == nil {
		t.Fatal("expected error for invalid staged dump path, got nil")
	}
}

func TestCreateBackupArchive_RenameError(t *testing.T) {
	temp := t.TempDir()
	root := filepath.Join(temp, "project")
	stage := filepath.Join(temp, "stage")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(stage, 0700); err != nil {
		t.Fatal(err)
	}

	// Renaming the temporary archive onto an existing directory must fail.
	destination := filepath.Join(temp, "existing-directory")
	if err := os.Mkdir(destination, 0700); err != nil {
		t.Fatal(err)
	}

	if err := backup.CreateBackupArchive(root, stage, destination); err == nil {
		t.Fatal("expected rename error, got nil")
	}
}
