package fileutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies a regular file from src to dst setting permissions to perm.
func CopyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("open destination file: %w", err)
	}

	if _, err = io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copy contents: %w", err)
	}
	return out.Close()
}

// CopyTree recursively copies an entire directory tree from src to dst.
func CopyTree(src, dst string, perm os.FileMode) error {
	return filepath.Walk(src, func(p string, i os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if i.IsDir() {
			return os.MkdirAll(target, 0750)
		}
		return CopyFile(p, target, perm)
	})
}
