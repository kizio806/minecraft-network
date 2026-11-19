package backup_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kizio806/minecraft-network/internal/backup"
)

func TestCreateBackupArchive_SuccessAndFiltering(t *testing.T) {
	temp := t.TempDir()
	root := filepath.Join(temp, "project")
	stage := filepath.Join(temp, "stage")
	_ = os.MkdirAll(filepath.Join(root, "configs"), 0700)
	_ = os.MkdirAll(filepath.Join(root, "servers", "lobby", "logs"), 0700)
	_ = os.MkdirAll(filepath.Join(root, "servers", "lobby", "cache"), 0700)
	_ = os.MkdirAll(filepath.Join(root, "secrets"), 0700)
	_ = os.MkdirAll(stage, 0700)

	// Valid files
	_ = os.WriteFile(filepath.Join(root, "configs", "network.yml"), []byte("name: test"), 0600)
	_ = os.WriteFile(filepath.Join(root, ".env"), []byte("ENV=test"), 0600)
	_ = os.WriteFile(filepath.Join(root, "secrets", "pass.txt"), []byte("secretpass"), 0600)
	_ = os.WriteFile(filepath.Join(stage, "postgres.dump"), []byte("pg_dump_content"), 0600)

	// Files that MUST be filtered out
	_ = os.WriteFile(filepath.Join(root, "servers", "lobby", "logs", "latest.log"), []byte("log data"), 0600)
	_ = os.WriteFile(filepath.Join(root, "servers", "lobby", "cache", "cached.bin"), []byte("cache data"), 0600)

	destArchive := filepath.Join(temp, "backup.tar.gz")
	if err := backup.CreateBackupArchive(root, stage, destArchive); err != nil {
		t.Fatalf("CreateBackupArchive failed: %v", err)
	}

	// Verify entries in destArchive
	f, err := os.Open(destArchive)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var foundNames []string
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		foundNames = append(foundNames, h.Name)
	}

	hasEntry := func(entry string) bool {
		for _, name := range foundNames {
			if name == entry {
				return true
			}
		}
		return false
	}

	if !hasEntry("configs/network.yml") {
		t.Error("expected configs/network.yml in archive")
	}
	if !hasEntry(".env") {
		t.Error("expected .env in archive")
	}
	if !hasEntry("secrets/pass.txt") {
		t.Error("expected secrets/pass.txt in archive")
	}
	if !hasEntry("database/postgres.dump") {
		t.Error("expected database/postgres.dump in archive")
	}
	if hasEntry("servers/lobby/logs/latest.log") {
		t.Error("expected log file to be excluded from archive")
	}
	if hasEntry("servers/lobby/cache/cached.bin") {
		t.Error("expected cache file to be excluded from archive")
	}
}

func TestCreateBackupArchive_InvalidDestination(t *testing.T) {
	temp := t.TempDir()
	invalidDest := filepath.Join(temp, "non-existent-sub", "backup.tar.gz")
	if err := backup.CreateBackupArchive(temp, temp, invalidDest); err == nil {
		t.Fatal("expected error for non-existent destination directory, got nil")
	}
}

func TestAddPath_Errors(t *testing.T) {
	temp := t.TempDir()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	defer tw.Close()

	if err := backup.AddPath(tw, filepath.Join(temp, "missing-file.txt"), "missing.txt"); err == nil {
		t.Fatal("expected error for missing file in AddPath, got nil")
	}
}

func TestAddTree_Errors(t *testing.T) {
	temp := t.TempDir()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	defer tw.Close()

	if err := backup.AddTree(tw, temp, "non-existent-rel"); err == nil {
		t.Fatal("expected error for missing tree in AddTree, got nil")
	}
}

func TestExtractBackupArchive_Extended(t *testing.T) {
	temp := t.TempDir()

	// Missing archive file
	if err := backup.ExtractBackupArchive(filepath.Join(temp, "missing.tar.gz"), temp); err == nil {
		t.Fatal("expected error for missing archive, got nil")
	}

	// Corrupt gzip
	corruptPath := filepath.Join(temp, "corrupt.tar.gz")
	_ = os.WriteFile(corruptPath, []byte("not-gzip-data"), 0600)
	if err := backup.ExtractBackupArchive(corruptPath, temp); err == nil {
		t.Fatal("expected error for corrupt gzip, got nil")
	}

	// Valid archive with directory header and normal file
	normalArchive := filepath.Join(temp, "normal.tar.gz")
	f, _ := os.OpenFile(normalArchive, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	// Write directory header
	dirH := &tar.Header{
		Name:     "subfolder/",
		Typeflag: tar.TypeDir,
		Mode:     0700,
	}
	_ = tw.WriteHeader(dirH)

	// Write regular file
	fileContent := []byte("regular file data")
	fileH := &tar.Header{
		Name:     "subfolder/file.txt",
		Typeflag: tar.TypeReg,
		Mode:     0600,
		Size:     int64(len(fileContent)),
	}
	_ = tw.WriteHeader(fileH)
	_, _ = tw.Write(fileContent)
	_ = tw.Close()
	_ = gz.Close()
	_ = f.Close()

	extractDir := filepath.Join(temp, "extracted")
	if err := backup.ExtractBackupArchive(normalArchive, extractDir); err != nil {
		t.Fatalf("ExtractBackupArchive failed for normal archive: %v", err)
	}

	extractedContent, err := os.ReadFile(filepath.Join(extractDir, "subfolder", "file.txt"))
	if err != nil {
		t.Fatalf("failed reading extracted file: %v", err)
	}
	if string(extractedContent) != "regular file data" {
		t.Fatalf("unexpected content: %s", string(extractedContent))
	}

	// Traversal: Absolute path
	absArchive := filepath.Join(temp, "abs.tar.gz")
	f2, _ := os.OpenFile(absArchive, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	gz2 := gzip.NewWriter(f2)
	tw2 := tar.NewWriter(gz2)
	_ = tw2.WriteHeader(&tar.Header{Name: "/etc/passwd", Size: 4, Mode: 0600})
	_, _ = tw2.Write([]byte("root"))
	_ = tw2.Close()
	_ = gz2.Close()
	_ = f2.Close()
	if err := backup.ExtractBackupArchive(absArchive, extractDir); err == nil || !errors.Is(err, backup.ErrUnsafeArchivePath) {
		t.Fatalf("expected ErrUnsafeArchivePath for absolute path, got: %v", err)
	}

	// Traversal: Backslash traversal
	winArchive := filepath.Join(temp, "win.tar.gz")
	f3, _ := os.OpenFile(winArchive, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	gz3 := gzip.NewWriter(f3)
	tw3 := tar.NewWriter(gz3)
	_ = tw3.WriteHeader(&tar.Header{Name: `..\escape.txt`, Size: 4, Mode: 0600})
	_, _ = tw3.Write([]byte("test"))
	_ = tw3.Close()
	_ = gz3.Close()
	_ = f3.Close()
	if err := backup.ExtractBackupArchive(winArchive, extractDir); err == nil || !errors.Is(err, backup.ErrUnsafeArchivePath) {
		t.Fatalf("expected ErrUnsafeArchivePath for win traversal, got: %v", err)
	}

	// Archive entry exceeds 500 MB limit
	largeArchive := filepath.Join(temp, "large.tar.gz")
	f4, _ := os.OpenFile(largeArchive, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	gz4, _ := gzip.NewWriterLevel(f4, gzip.BestSpeed)
	tw4 := tar.NewWriter(gz4)
	const largeSize int64 = (500 << 20) + 1024
	_ = tw4.WriteHeader(&tar.Header{
		Name: "large_file.bin",
		Size: largeSize,
		Mode: 0600,
	})
	chunk := make([]byte, 8*1024*1024) // 8MB zero chunk
	for written := int64(0); written < largeSize; {
		toWrite := int64(len(chunk))
		if largeSize-written < toWrite {
			toWrite = largeSize - written
		}
		_, _ = tw4.Write(chunk[:toWrite])
		written += toWrite
	}
	_ = tw4.Close()
	_ = gz4.Close()
	_ = f4.Close()

	if err := backup.ExtractBackupArchive(largeArchive, extractDir); err == nil || !errors.Is(err, backup.ErrArchiveEntryTooLarge) {
		t.Fatalf("expected ErrArchiveEntryTooLarge, got: %v", err)
	}
}
