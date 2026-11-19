package main

import (
	"os"
	"os/exec"
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

func TestMainInvocation(t *testing.T) {
	if os.Getenv("FLAG_TEST_MAIN") == "1" {
		os.Args = []string{"network", "--help"}
		main()
		return
	}
	// #nosec G204 -- test subprocess execution of test binary
	cmd := exec.Command(os.Args[0], "-test.run=TestMainInvocation")
	cmd.Env = append(os.Environ(), "FLAG_TEST_MAIN=1")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("main failed: %v", err)
	}
}
