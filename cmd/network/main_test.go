package main

import (
	"testing"
)

func TestRun(t *testing.T) {
	// Help flag should exit with code 0
	if code := run([]string{"--help"}); code != 0 {
		t.Fatalf("expected code 0 on --help, got %d", code)
	}

	// Unknown command with valid project dir should exit with code 2
	if code := run([]string{"--project-dir", "../..", "unknown-test-cmd"}); code != 2 {
		t.Fatalf("expected code 2 on unknown command, got %d", code)
	}
}
