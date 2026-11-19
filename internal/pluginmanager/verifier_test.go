package pluginmanager

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyFileDetails(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "artifact.jar")
	data := []byte("jar-content-payload")
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}

	// Missing file
	if err := verifyFile(filepath.Join(dir, "missing.jar"), LockEntry{}); err == nil {
		t.Fatal("expected error on missing file")
	}

	// Size mismatch
	if err := verifyFile(p, LockEntry{Size: 999}); err == nil {
		t.Fatal("expected error on size mismatch")
	}

	// SHA256 match
	sha256Hash := "e359abefc5617bda8ed67f0dcad753d547995fc6e2e87179a15a4631354dbff6"
	if err := verifyFile(p, LockEntry{Size: int64(len(data)), SHA256: sha256Hash}); err != nil {
		t.Fatalf("unexpected verifyFile error on valid sha256: %v", err)
	}

	// SHA256 mismatch
	if err := verifyFile(p, LockEntry{SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}); err == nil {
		t.Fatal("expected error on sha256 mismatch")
	}

	// SHA512 mismatch
	if err := verifyFile(p, LockEntry{SHA512: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}); err == nil {
		t.Fatal("expected error on sha512 mismatch")
	}
}

func TestInspectJarDescriptors(t *testing.T) {
	dir := t.TempDir()

	// Invalid ZIP file
	corruptPath := filepath.Join(dir, "corrupt.jar")
	_ = os.WriteFile(corruptPath, []byte("not a zip"), 0600)
	if err := inspectJar(corruptPath, "velocity"); err == nil {
		t.Fatal("expected error on corrupt ZIP")
	}

	// Valid Velocity JAR
	veloPath := filepath.Join(dir, "velocity-plugin.jar")
	createZipWithFile(t, veloPath, "velocity-plugin.json")
	if err := inspectJar(veloPath, "velocity"); err != nil {
		t.Fatalf("unexpected error on valid velocity descriptor: %v", err)
	}
	if err := inspectJar(veloPath, "paper"); err == nil {
		t.Fatal("expected error when inspecting velocity JAR for paper")
	}

	// Valid Paper JAR (plugin.yml)
	paperPath := filepath.Join(dir, "paper-plugin.jar")
	createZipWithFile(t, paperPath, "plugin.yml")
	if err := inspectJar(paperPath, "paper"); err != nil {
		t.Fatalf("unexpected error on valid paper descriptor: %v", err)
	}

	// Valid Paper JAR (paper-plugin.yml)
	paper2Path := filepath.Join(dir, "paper2-plugin.jar")
	createZipWithFile(t, paper2Path, "paper-plugin.yml")
	if err := inspectJar(paper2Path, "paper"); err != nil {
		t.Fatalf("unexpected error on valid paper-plugin.yml: %v", err)
	}
}

func createZipWithFile(t *testing.T, target, filename string) {
	t.Helper()
	f, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	fw, err := w.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte("mock-descriptor"))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}
