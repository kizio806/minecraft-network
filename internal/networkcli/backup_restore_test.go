package networkcli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/kizio806/minecraft-network/internal/fileutil"
)

func TestBackupRestoreHelpers(t *testing.T) {
	dir := t.TempDir()

	// fileSHA256
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	hash, err := fileSHA256(filePath)
	if err != nil {
		t.Fatalf("fileSHA256 failed: %v", err)
	}
	if hash != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("unexpected hash: %s", hash)
	}

	// secureKeyFile permissions check
	if err := secureKeyFile(filePath); err != nil {
		t.Fatalf("expected 0600 file to be secure, got: %v", err)
	}
	_ = os.Chmod(filePath, 0644)
	if err := secureKeyFile(filePath); err == nil {
		t.Fatal("expected error on 0644 permissions")
	}
	if err := secureKeyFile(filepath.Join(dir, "missing.key")); err == nil {
		t.Fatal("expected error on missing key file")
	}
	_ = os.Chmod(filePath, 0600)

	// copyFile
	destFile := filepath.Join(dir, "copy.txt")
	if err := fileutil.CopyFile(filePath, destFile, 0600); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}
	data, _ := os.ReadFile(destFile)
	if string(data) != "hello" {
		t.Fatalf("unexpected copyFile data: %s", string(data))
	}
	if err := fileutil.CopyFile(filepath.Join(dir, "non-existent"), destFile, 0600); err == nil {
		t.Fatal("expected error on non-existent copyFile source")
	}

	// copyTree
	srcTree := filepath.Join(dir, "src_tree")
	dstTree := filepath.Join(dir, "dst_tree")
	if err := os.MkdirAll(filepath.Join(srcTree, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcTree, "nested", "f.txt"), []byte("nested-data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := fileutil.CopyTree(srcTree, dstTree, 0600); err != nil {
		t.Fatalf("copyTree failed: %v", err)
	}
	nestedData, _ := os.ReadFile(filepath.Join(dstTree, "nested", "f.txt"))
	if string(nestedData) != "nested-data" {
		t.Fatalf("unexpected copyTree data: %s", string(nestedData))
	}

	// addPath
	tarFile := filepath.Join(dir, "test.tar")
	tf, err := os.Create(tarFile)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(tf)
	if err := addPath(tw, filePath, "test.txt"); err != nil {
		t.Fatalf("addPath failed: %v", err)
	}
	if err := addTree(tw, srcTree, "nested"); err != nil {
		t.Fatalf("addTree failed: %v", err)
	}
	_ = tw.Close()
	_ = tf.Close()

	if err := addPath(tw, filepath.Join(dir, "missing"), "m"); err == nil {
		t.Fatal("expected error on missing addPath")
	}
	if err := addTree(tw, dir, "missing_tree"); err == nil {
		t.Fatal("expected error on missing addTree")
	}

	// Encrypt and Decrypt with invalid keys
	badKeyFile := filepath.Join(dir, "bad.key")
	_ = os.WriteFile(badKeyFile, []byte("invalid-key"), 0600)
	if err := encryptBackup(filePath, filepath.Join(dir, "enc.age"), badKeyFile); err == nil {
		t.Fatal("expected error on encrypt with invalid recipient")
	}
	if _, err := decryptBackup(filepath.Join(dir, "enc.age"), badKeyFile); err == nil {
		t.Fatal("expected error on decrypt with invalid identity")
	}

	// Valid age keypair encryption & decryption
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	recKeyFile := filepath.Join(dir, "valid.recipient")
	idKeyFile := filepath.Join(dir, "valid.identity")
	_ = os.WriteFile(recKeyFile, []byte(id.Recipient().String()), 0600)
	_ = os.WriteFile(idKeyFile, []byte(id.String()), 0600)

	encFile := filepath.Join(dir, "enc_valid.age")
	if err := encryptBackup(filePath, encFile, recKeyFile); err != nil {
		t.Fatalf("encryptBackup failed: %v", err)
	}
	decFile, err := decryptBackup(encFile, idKeyFile)
	if err != nil {
		t.Fatalf("decryptBackup failed: %v", err)
	}
	decBytes, _ := os.ReadFile(decFile)
	if string(decBytes) != "hello" {
		t.Fatalf("unexpected decrypted data: %s", string(decBytes))
	}
	_ = os.Remove(decFile)

	// Backup command wrapper
	c := operationalConfig(t)
	app := App{Runner: &fakeRunner{}}
	if err := app.backup(context.Background(), c, []string{"list"}); err != nil {
		t.Fatalf("unexpected backup list error: %v", err)
	}

	// Restore command wrapper
	if err := app.restore(context.Background(), c, []string{}, false); err == nil {
		t.Fatal("expected error on empty restore args")
	}
}

func TestZipSlipRejection(t *testing.T) {
	dir := t.TempDir()
	c := operationalConfig(t)
	c.BackupDir = dir

	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	recKeyFile := filepath.Join(dir, "recipient.key")
	idKeyFile := filepath.Join(dir, "identity.key")
	_ = os.WriteFile(recKeyFile, []byte(id.Recipient().String()), 0600)
	_ = os.WriteFile(idKeyFile, []byte(id.String()), 0600)
	c.BackupRecipientFile = recKeyFile
	c.BackupIdentityFile = idKeyFile

	// Create malicious tarball containing ../evil.txt
	plainTarGz := filepath.Join(dir, "malicious.tar.gz")
	f, err := os.Create(plainTarGz)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	maliciousHeader := &tar.Header{
		Name: "../evil.txt",
		Mode: 0600,
		Size: int64(len("exploit")),
	}
	if err := tw.WriteHeader(maliciousHeader); err != nil {
		t.Fatal(err)
	}
	_, _ = tw.Write([]byte("exploit"))
	_ = tw.Close()
	_ = gw.Close()
	_ = f.Close()

	archivePath := filepath.Join(dir, "minecraft-network-20260101T000000Z.tar.gz.age")
	if err := encryptBackup(plainTarGz, archivePath, recKeyFile); err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	hash, err := fileSHA256(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(archivePath+".sha256", []byte(hash+"  "+filepath.Base(archivePath)+"\n"), 0600)

	app := App{Runner: &fakeRunner{}}
	err = app.nativeRestore(context.Background(), c, archivePath, true)
	if err == nil {
		t.Fatal("expected nativeRestore to reject Zip Slip path traversal")
	}
	if !errors.Is(err, ErrUnsafeArchivePath) && !strings.Contains(err.Error(), "unsafe archive path") {
		t.Fatalf("expected ErrUnsafeArchivePath, got: %v", err)
	}
}

func TestCleanupBackupsRotation(t *testing.T) {
	dir := t.TempDir()

	// Create 5 fake backups with varying modification times
	now := time.Now()
	for i := 1; i <= 5; i++ {
		name := fmt.Sprintf("minecraft-network-2026010%dT000000Z.tar.gz.age", i)
		archive := filepath.Join(dir, name)
		_ = os.WriteFile(archive, []byte(fmt.Sprintf("backup %d", i)), 0600)
		_ = os.WriteFile(archive+".sha256", []byte("hash"), 0600)
		modTime := now.Add(time.Duration(i) * time.Hour)
		_ = os.Chtimes(archive, modTime, modTime)
	}

	// Retention = 2 -> keep 4 and 5, remove 1, 2, 3
	if err := cleanupBackups(dir, 2); err != nil {
		t.Fatalf("cleanupBackups failed: %v", err)
	}

	// Verify only 2 remain
	matches, _ := filepath.Glob(filepath.Join(dir, "*.tar.gz.age"))
	if len(matches) != 2 {
		t.Fatalf("expected 2 backups after retention cleanup, got %d", len(matches))
	}
	// Verify older sidecars removed
	if _, err := os.Stat(filepath.Join(dir, "minecraft-network-20260101T000000Z.tar.gz.age.sha256")); !os.IsNotExist(err) {
		t.Fatal("expected old sha256 sidecar to be removed")
	}

	// Retention < 1 default test
	if err := cleanupBackups(dir, 0); err != nil {
		t.Fatalf("cleanupBackups failed on retention 0: %v", err)
	}
	matches, _ = filepath.Glob(filepath.Join(dir, "*.tar.gz.age"))
	if len(matches) != 1 {
		t.Fatalf("expected 1 backup after retention 0 (default 1), got %d", len(matches))
	}
}

func TestRestoreValidationFailures(t *testing.T) {
	dir := t.TempDir()
	c := operationalConfig(t)
	c.BackupDir = dir
	app := App{Runner: &fakeRunner{}}
	ctx := context.Background()

	// 1. Restore without yes flag
	if err := app.restore(ctx, c, []string{"some-backup.tar.gz.age"}, false); err == nil {
		t.Fatal("expected error on restore without --yes")
	}

	// 2. Restore with invalid arg count
	if err := app.restore(ctx, c, []string{"b1", "b2"}, true); err == nil {
		t.Fatal("expected error on restore with multiple backup args")
	}

	// 3. Native restore without yes
	if err := app.nativeRestore(ctx, c, "some.tar.gz.age", false); err == nil {
		t.Fatal("expected nativeRestore error without yes")
	}

	// 4. Archive does not exist
	missingArchive := filepath.Join(dir, "missing.tar.gz.age")
	if err := app.nativeRestore(ctx, c, missingArchive, true); err == nil {
		t.Fatal("expected error on missing archive")
	}

	// 5. Invalid archive extension
	invalidExt := filepath.Join(dir, "backup.zip")
	_ = os.WriteFile(invalidExt, []byte("data"), 0600)
	if err := app.nativeRestore(ctx, c, invalidExt, true); err == nil {
		t.Fatal("expected error on non-.tar.gz.age file")
	}

	// 6. Missing .sha256 sidecar
	validExt := filepath.Join(dir, "backup.tar.gz.age")
	_ = os.WriteFile(validExt, []byte("data"), 0600)
	if err := app.nativeRestore(ctx, c, validExt, true); err == nil {
		t.Fatal("expected error when .sha256 sidecar is missing")
	}

	// 7. Invalid .sha256 content
	_ = os.WriteFile(validExt+".sha256", []byte("invalid_hash\n"), 0600)
	if err := app.nativeRestore(ctx, c, validExt, true); err == nil {
		t.Fatal("expected error when .sha256 content is not 64 hex characters")
	}

	// 8. Hash mismatch
	dummyHash := strings.Repeat("a", 64)
	_ = os.WriteFile(validExt+".sha256", []byte(dummyHash+"  backup.tar.gz.age\n"), 0600)
	if err := app.nativeRestore(ctx, c, validExt, true); err == nil {
		t.Fatal("expected error on hash mismatch")
	}
}

func TestNativeBackupFailures(t *testing.T) {
	c := operationalConfig(t)
	ctx := context.Background()

	// 1. Secret validation failure
	cBadSecrets := c
	cBadSecrets.ProjectDir = t.TempDir() // has no secrets/
	app := App{Runner: &fakeRunner{}}
	if err := app.nativeBackup(ctx, cBadSecrets); err == nil {
		t.Fatal("expected error on invalid secrets in nativeBackup")
	}

	// 2. Compose config failure
	errRunner := &fakeRunner{
		failContains: "config --quiet",
	}
	appErr := App{Runner: errRunner}
	if err := appErr.nativeBackup(ctx, c); err == nil {
		t.Fatal("expected error when compose config fails")
	}

	// 3. Compose stop failure
	stopErrRunner := &fakeRunner{
		failContains: "stop proxy",
	}
	appStopErr := App{Runner: stopErrRunner}
	if err := appStopErr.nativeBackup(ctx, c); err == nil {
		t.Fatal("expected error when compose stop fails")
	}
}
