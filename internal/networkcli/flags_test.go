package networkcli

import (
	"testing"
)

func TestParseGlobal(t *testing.T) {
	// Full flags
	args := []string{
		"--project-dir", "/tmp/project",
		"--output", "json",
		"--yes",
		"--dry-run",
		"--follow",
		"--monitoring",
		"--lines", "50",
		"status",
	}
	p, out, yes, dry, follow, mon, lines, rest, err := parseGlobal(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != "/tmp/project" || out != "json" || !yes || !dry || !follow || !mon || lines != 50 {
		t.Fatalf("unexpected flag values: p=%s out=%s yes=%v dry=%v follow=%v mon=%v lines=%d", p, out, yes, dry, follow, mon, lines)
	}
	if len(rest) != 1 || rest[0] != "status" {
		t.Fatalf("unexpected rest: %v", rest)
	}

	// Missing values
	if _, _, _, _, _, _, _, _, err := parseGlobal([]string{"--project-dir"}); err == nil {
		t.Fatal("expected error for --project-dir without value")
	}
	if _, _, _, _, _, _, _, _, err := parseGlobal([]string{"--output"}); err == nil {
		t.Fatal("expected error for --output without value")
	}
	if _, _, _, _, _, _, _, _, err := parseGlobal([]string{"--lines"}); err == nil {
		t.Fatal("expected error for --lines without value")
	}
	if _, _, _, _, _, _, _, _, err := parseGlobal([]string{"--lines", "invalid"}); err == nil {
		t.Fatal("expected error for non-integer --lines")
	}
	if _, _, _, _, _, _, _, _, err := parseGlobal([]string{"--lines", "-10"}); err == nil {
		t.Fatal("expected error for negative --lines")
	}

	// Version flag stops global parsing
	_, _, _, _, _, _, _, rest, err = parseGlobal([]string{"--version"})
	if err != nil || len(rest) == 0 || rest[0] != "--version" {
		t.Fatalf("expected --version in rest: %v, err: %v", rest, err)
	}

	// No flags
	p, out, yes, dry, follow, mon, lines, rest, err = parseGlobal([]string{"doctor"})
	if err != nil || p != "" || out != "" || yes || dry || follow || mon || lines != 100 || len(rest) != 1 || rest[0] != "doctor" {
		t.Fatalf("unexpected default parse: rest=%v, err=%v", rest, err)
	}
}

func TestStripFlag(t *testing.T) {
	args := []string{"start", "--yes", "proxy"}
	stripped, found := stripFlag(args, "--yes")
	if !found || len(stripped) != 2 || stripped[0] != "start" || stripped[1] != "proxy" {
		t.Fatalf("unexpected stripFlag result: found=%v, stripped=%v", found, stripped)
	}

	stripped, found = stripFlag(args, "--dry-run")
	if found || len(stripped) != 3 {
		t.Fatalf("expected not found: %v, %v", found, stripped)
	}
}
