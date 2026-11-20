package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	dst := filepath.Join(dir, "sub", "destination.txt")

	content := []byte("hello fileutil copy")
	if err := os.WriteFile(src, content, 0600); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(src, dst, 0600); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("expected %q, got %q", string(content), string(got))
	}

	if err := CopyFile(filepath.Join(dir, "missing.txt"), dst, 0600); err == nil {
		t.Fatal("expected error for missing source file")
	}

	if err := CopyFile(src, filepath.Join(src, "invalid", "target.txt"), 0600); err == nil {
		t.Fatal("expected error when destination directory cannot be created")
	}

	if err := CopyFile(src, dir, 0600); err == nil {
		t.Fatal("expected error when destination cannot be opened as a regular file")
	}

	if err := CopyFile(dir, filepath.Join(dir, "cannot_copy_dir.txt"), 0600); err == nil {
		t.Fatal("expected error when attempting to copy directory as regular file")
	}
}

func TestCopyTree(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	dstDir := filepath.Join(dir, "dst")

	if err := os.MkdirAll(filepath.Join(srcDir, "a", "b"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "root.txt"), []byte("root"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "a", "b", "nested.txt"), []byte("nested"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := CopyTree(srcDir, dstDir, 0600); err != nil {
		t.Fatalf("CopyTree failed: %v", err)
	}

	rootGot, err := os.ReadFile(filepath.Join(dstDir, "root.txt"))
	if err != nil || string(rootGot) != "root" {
		t.Fatalf("unexpected root content: %s %v", string(rootGot), err)
	}

	nestedGot, err := os.ReadFile(filepath.Join(dstDir, "a", "b", "nested.txt"))
	if err != nil || string(nestedGot) != "nested" {
		t.Fatalf("unexpected nested content: %s %v", string(nestedGot), err)
	}

	if err := CopyTree(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "out"), 0600); err == nil {
		t.Fatal("expected error when walking non-existent source directory")
	}

	blockedDst := filepath.Join(dir, "blocked_dst")
	_ = os.MkdirAll(blockedDst, 0750)
	_ = os.WriteFile(filepath.Join(blockedDst, "a"), []byte("blocking-file"), 0600)
	if err := CopyTree(srcDir, blockedDst, 0600); err == nil {
		t.Fatal("expected error when target directory collides with existing file")
	}
}
