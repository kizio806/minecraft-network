package backup_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/kizio806/minecraft-network/internal/backup"
)

func TestFileSHA256(t *testing.T) {
	temp := t.TempDir()
	filePath := filepath.Join(temp, "sample.txt")
	content := []byte("hello sha256 verification")
	if err := os.WriteFile(filePath, content, 0600); err != nil {
		t.Fatal(err)
	}

	expectedHash := sha256.Sum256(content)
	expectedHex := hex.EncodeToString(expectedHash[:])

	gotHex, err := backup.FileSHA256(filePath)
	if err != nil {
		t.Fatalf("FileSHA256 failed: %v", err)
	}
	if gotHex != expectedHex {
		t.Fatalf("expected hash %s, got %s", expectedHex, gotHex)
	}

	if _, err := backup.FileSHA256(filepath.Join(temp, "non-existent.txt")); err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestSecureKeyFile(t *testing.T) {
	temp := t.TempDir()

	missing := filepath.Join(temp, "missing.key")
	if err := backup.SecureKeyFile(missing); err == nil {
		t.Fatal("expected error for missing key file, got nil")
	}

	dirPath := filepath.Join(temp, "dir.key")
	if err := os.Mkdir(dirPath, 0700); err != nil {
		t.Fatal(err)
	}
	if err := backup.SecureKeyFile(dirPath); err == nil {
		t.Fatal("expected error for directory key, got nil")
	}

	insecureFile := filepath.Join(temp, "insecure.key")
	if err := os.WriteFile(insecureFile, []byte("key"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := backup.SecureKeyFile(insecureFile); err == nil {
		t.Fatal("expected error for 0644 key file, got nil")
	}

	validFile := filepath.Join(temp, "valid.key")
	if err := os.WriteFile(validFile, []byte("key"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := backup.SecureKeyFile(validFile); err != nil {
		t.Fatalf("expected valid 0600 key file, got error: %v", err)
	}
}

func TestEncryptBackup_Errors(t *testing.T) {
	temp := t.TempDir()

	insecureRecipient := filepath.Join(temp, "rec_bad.txt")
	_ = os.WriteFile(insecureRecipient, []byte("key"), 0644)
	inputFile := filepath.Join(temp, "input.txt")
	_ = os.WriteFile(inputFile, []byte("data"), 0600)
	outFile := filepath.Join(temp, "out.age")

	if err := backup.EncryptBackup(inputFile, outFile, insecureRecipient); err == nil {
		t.Fatal("expected error for insecure recipient file, got nil")
	}

	invalidRecipient := filepath.Join(temp, "rec_invalid.txt")
	_ = os.WriteFile(invalidRecipient, []byte("not-an-age-recipient"), 0600)
	if err := backup.EncryptBackup(inputFile, outFile, invalidRecipient); err == nil || !strings.Contains(err.Error(), "invalid backup recipient") {
		t.Fatalf("expected invalid backup recipient error, got: %v", err)
	}

	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	validRecipient := filepath.Join(temp, "rec_valid.txt")
	_ = os.WriteFile(validRecipient, []byte(id.Recipient().String()+"\n"), 0600)

	missingInput := filepath.Join(temp, "non-existent-input.txt")
	if err := backup.EncryptBackup(missingInput, outFile, validRecipient); err == nil {
		t.Fatal("expected error for missing input file, got nil")
	}

	invalidOutput := filepath.Join(temp, "missing_dir", "out.age")
	if err := backup.EncryptBackup(inputFile, invalidOutput, validRecipient); err == nil {
		t.Fatal("expected error for non-existent output directory, got nil")
	}
}

func TestDecryptBackup_Errors(t *testing.T) {
	temp := t.TempDir()

	insecureIdentity := filepath.Join(temp, "id_bad.txt")
	_ = os.WriteFile(insecureIdentity, []byte("key"), 0644)
	encFile := filepath.Join(temp, "test.age")
	_ = os.WriteFile(encFile, []byte("age-encrypted-data"), 0600)

	if _, err := backup.DecryptBackup(encFile, insecureIdentity); err == nil {
		t.Fatal("expected error for insecure identity file, got nil")
	}

	invalidIdentity := filepath.Join(temp, "id_invalid.txt")
	_ = os.WriteFile(invalidIdentity, []byte("invalid-identity-content\n"), 0600)
	if _, err := backup.DecryptBackup(encFile, invalidIdentity); err == nil || !strings.Contains(err.Error(), "invalid backup identity") {
		t.Fatalf("expected invalid backup identity error, got: %v", err)
	}

	id1, _ := age.GenerateX25519Identity()
	validIdentity1 := filepath.Join(temp, "id_valid1.txt")
	_ = os.WriteFile(validIdentity1, []byte(id1.String()+"\n"), 0600)

	missingInput := filepath.Join(temp, "missing.age")
	if _, err := backup.DecryptBackup(missingInput, validIdentity1); err == nil {
		t.Fatal("expected error for missing input file, got nil")
	}

	corruptInput := filepath.Join(temp, "corrupt.age")
	_ = os.WriteFile(corruptInput, []byte("not-real-age-ciphertext"), 0600)
	if _, err := backup.DecryptBackup(corruptInput, validIdentity1); err == nil || !strings.Contains(err.Error(), "backup decryption failed") {
		t.Fatalf("expected backup decryption failed error, got: %v", err)
	}

	id2, _ := age.GenerateX25519Identity()
	rec2Path := filepath.Join(temp, "rec2.txt")
	_ = os.WriteFile(rec2Path, []byte(id2.Recipient().String()+"\n"), 0600)

	plainFile := filepath.Join(temp, "plain.txt")
	_ = os.WriteFile(plainFile, []byte("secret"), 0600)
	outAgeFile := filepath.Join(temp, "encrypted_id2.age")
	if err := backup.EncryptBackup(plainFile, outAgeFile, rec2Path); err != nil {
		t.Fatal(err)
	}

	if _, err := backup.DecryptBackup(outAgeFile, validIdentity1); err == nil || !strings.Contains(err.Error(), "backup decryption failed") {
		t.Fatalf("expected backup decryption failed for wrong key, got: %v", err)
	}
}
