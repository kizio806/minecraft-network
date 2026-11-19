package pluginmanager

import (
	"archive/zip"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func verifyFile(path string, entry LockEntry) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h256, h512 := sha256.New(), sha512.New()
	n, err := io.Copy(io.MultiWriter(h256, h512), f)
	if err != nil {
		return err
	}
	if entry.Size > 0 && n != entry.Size {
		return fmt.Errorf("size %d does not match lockfile %d", n, entry.Size)
	}
	if entry.SHA256 != "" && hex.EncodeToString(h256.Sum(nil)) != strings.ToLower(entry.SHA256) {
		return errors.New("sha256 mismatch")
	}
	if entry.SHA512 != "" && hex.EncodeToString(h512.Sum(nil)) != strings.ToLower(entry.SHA512) {
		return errors.New("sha512 mismatch")
	}
	return nil
}

func inspectJar(path, platform string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("artifact is not a valid ZIP/JAR: %w", err)
	}
	defer r.Close()
	for _, file := range r.File {
		name := filepath.ToSlash(file.Name)
		if platform == "velocity" && name == "velocity-plugin.json" {
			return nil
		}
		if platform == "paper" && (name == "plugin.yml" || name == "paper-plugin.yml") {
			return nil
		}
	}
	return fmt.Errorf("artifact has no %s plugin descriptor", platform)
}
