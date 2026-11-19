package backup_test

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/kizio806/minecraft-network/internal/backup"
)

func TestZipSlipRejection(t *testing.T) {
	temp := t.TempDir()
	archivePath := filepath.Join(temp, "slip.tar.gz")

	f, err := os.OpenFile(archivePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	h := &tar.Header{
		Name: "../evil.txt",
		Mode: 0600,
		Size: 4,
	}
	if err := tw.WriteHeader(h); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("evil")); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gz.Close()
	_ = f.Close()

	destDir := filepath.Join(temp, "dest")
	_ = os.MkdirAll(destDir, 0700)

	err = backup.ExtractBackupArchive(archivePath, destDir)
	if err == nil || !strings.Contains(err.Error(), "directory traversal") {
		t.Fatalf("expected directory traversal error, got %v", err)
	}
}

func TestCryptoRoundtrip(t *testing.T) {
	temp := t.TempDir()
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	idPath := filepath.Join(temp, "id.txt")
	recPath := filepath.Join(temp, "rec.txt")

	if err := os.WriteFile(idPath, []byte(id.String()+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(recPath, []byte(id.Recipient().String()+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	plainPath := filepath.Join(temp, "payload.txt")
	encPath := filepath.Join(temp, "payload.age")
	if err := os.WriteFile(plainPath, []byte("super secret data"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := backup.EncryptBackup(plainPath, encPath, recPath); err != nil {
		t.Fatalf("unexpected encrypt error: %v", err)
	}

	decryptedPath, err := backup.DecryptBackup(encPath, idPath)
	if err != nil {
		t.Fatalf("unexpected decrypt error: %v", err)
	}
	defer os.Remove(decryptedPath)

	content, err := os.ReadFile(decryptedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "super secret data" {
		t.Fatalf("unexpected content: %s", string(content))
	}
}
