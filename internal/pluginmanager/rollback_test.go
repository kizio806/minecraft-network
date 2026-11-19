package pluginmanager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRollback(t *testing.T) {
	root := t.TempDir()
	mgr := &Manager{root: root}

	// Missing rollback snapshot
	if err := mgr.Rollback(); err == nil {
		t.Fatal("expected error on missing rollback snapshot")
	}

	// Setup rollback scenario
	pmDir := filepath.Join(root, ".plugin-manager")
	if err := os.MkdirAll(pmDir, 0700); err != nil {
		t.Fatal(err)
	}

	destFile := filepath.Join(root, "servers/proxy/plugins/installed.jar")
	backupFile := filepath.Join(pmDir, "rollback/installed.jar.bak")
	_ = os.MkdirAll(filepath.Dir(destFile), 0750)
	_ = os.MkdirAll(filepath.Dir(backupFile), 0700)
	_ = os.WriteFile(destFile, []byte("new-version"), 0600)
	_ = os.WriteFile(backupFile, []byte("original-version"), 0600)

	newFile := filepath.Join(root, "servers/proxy/plugins/new.jar")
	_ = os.WriteFile(newFile, []byte("brand-new"), 0600)

	changes := []change{
		{destination: destFile, backup: backupFile, existed: true},
		{destination: newFile, backup: "", existed: false},
	}

	// Test writeRollback
	if err := mgr.writeRollback(changes); err != nil {
		t.Fatalf("writeRollback failed: %v", err)
	}

	// Test rollback helper function
	if err := rollback(changes); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	restored, _ := os.ReadFile(destFile)
	if string(restored) != "original-version" {
		t.Fatalf("expected original-version after rollback, got: %s", string(restored))
	}
	if _, err := os.Stat(newFile); !os.IsNotExist(err) {
		t.Fatal("expected newFile to be removed after rollback")
	}

	// Mock docker-compose.yml so ensureStopped passes
	_ = os.WriteFile(filepath.Join(root, "docker-compose.yml"), []byte("services: {}\n"), 0600)
	// Mock docker executable on PATH
	bin := t.TempDir()
	docker := filepath.Join(bin, "docker")
	_ = os.WriteFile(docker, []byte("#!/bin/sh\nexit 0\n"), 0755)
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+oldPath)

	// Test Rollback() method
	_ = os.WriteFile(destFile, []byte("new-version"), 0600)
	_ = os.WriteFile(backupFile, []byte("original-version"), 0600)
	_ = os.WriteFile(newFile, []byte("brand-new"), 0600)
	if err := mgr.writeRollback(changes); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Rollback(); err != nil {
		t.Fatalf("Rollback() method failed: %v", err)
	}
	restored, _ = os.ReadFile(destFile)
	if string(restored) != "original-version" {
		t.Fatalf("expected original-version after Rollback(), got: %s", string(restored))
	}

	// Test Rollback error when backup file missing for rename
	missingBackupChanges := []change{
		{destination: filepath.Join(root, "dst.txt"), backup: filepath.Join(root, "missing_bak.txt"), existed: true},
	}
	if err := rollback(missingBackupChanges); err == nil {
		t.Fatal("expected error from rollback() when backup file is missing")
	}
	if err := mgr.writeRollback(missingBackupChanges); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Rollback(); err == nil {
		t.Fatal("expected error from mgr.Rollback() when backup file is missing")
	}
}
