package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
)

// FileSHA256 computes and returns the hex-encoded SHA-256 digest of a file.
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	_, err = io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), err
}

// SecureKeyFile verifies a key file exists, is regular, and has 0600 permissions.
func SecureKeyFile(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("backup key file unavailable: %w", err)
	}
	if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return errors.New("backup key file must be regular and mode 0600")
	}
	return nil
}

// EncryptBackup encrypts an input tar.gz file using an age X25519 recipient.
func EncryptBackup(input, output, recipientPath string) error {
	if err := SecureKeyFile(recipientPath); err != nil {
		return err
	}
	b, err := os.ReadFile(recipientPath)
	if err != nil {
		return err
	}
	recipient, err := age.ParseX25519Recipient(strings.TrimSpace(string(b)))
	if err != nil {
		return fmt.Errorf("invalid backup recipient: %w", err)
	}
	in, err := os.Open(input)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	writer, err := age.Encrypt(out, recipient)
	if err != nil {
		_ = out.Close()
		return err
	}
	if _, err = io.Copy(writer, in); err != nil {
		_ = writer.Close()
		_ = out.Close()
		return err
	}
	if err = writer.Close(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// DecryptBackup decrypts an age encrypted file using an age X25519 identity.
func DecryptBackup(input, identityPath string) (string, error) {
	if err := SecureKeyFile(identityPath); err != nil {
		return "", err
	}
	b, err := os.ReadFile(identityPath)
	if err != nil {
		return "", err
	}
	identities, err := age.ParseIdentities(strings.NewReader(string(b)))
	if err != nil || len(identities) == 0 {
		return "", errors.New("invalid backup identity")
	}
	in, err := os.Open(input)
	if err != nil {
		return "", err
	}
	defer in.Close()
	reader, err := age.Decrypt(in, identities...)
	if err != nil {
		return "", errors.New("backup decryption failed")
	}
	out, err := os.CreateTemp("", "minecraft-backup-plaintext-*.tar.gz")
	if err != nil {
		return "", err
	}
	path := out.Name()
	if _, err = io.Copy(out, reader); err != nil {
		_ = out.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err = out.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}
