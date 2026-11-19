# Workflow: Bug Fixing

## Purpose & Scope
Define the standard lifecycle for diagnosing, reproducing, fixing, and verifying defects in Go CLI tooling, Docker Compose configurations, plugin templates, or database scripts.

## When to Use
Use this workflow whenever addressing unexpected runtime errors, failed healthchecks, CLI crashes, configuration rendering errors, or permission violations.

## Steps

### Step 1: Reproduction & Triage
1. Capture error output, stack traces, and exit codes:
   ```bash
   ./bin/network doctor
   ./bin/network logs <service> --tail 100
   ```
2. Isolate the defective component boundary (CLI Go code, Docker Compose, PostgreSQL schema, or plugin template).
3. Ensure git working tree is clean with `git status -s`.

### Step 2: Root Cause Analysis
1. Inspect file permissions, template placeholders (`__PLACEHOLDER__`), or Go error handling.
2. Check for race conditions, nil pointer dereferences, or unclosed file descriptors.
3. Validate path traversal sanitization on all file operations.

### Step 3: Test-Driven Fix
1. Write a failing regression test in `internal/networkcli/cli_test.go` or `internal/pluginmanager/manager_test.go`.
2. Implement the minimal targeted fix resolving the defect without touching unrelated files.
3. Ensure backward compatibility of CLI flags and configuration formats.

### Step 4: Verification & Rebuild
1. Run all unit tests with the race detector:
   ```bash
   docker run --rm -v "$(pwd)":/app -w /app golang:1.24 go test -v -race ./...
   ```
2. Recompile production binaries with `-trimpath`:
   ```bash
   go build -trimpath -ldflags="-s -w" -o bin/network ./cmd/network
   go build -trimpath -ldflags="-s -w" -o bin/plugin-manager ./cmd/plugin-manager
   ```
3. Run the doctor audit and healthcheck:
   ```bash
   ./bin/network doctor
   ./bin/network health
   ```

## Required Checks
- [ ] Regression test passes with race detection: `go test -v -race ./...`.
- [ ] No regression in `./bin/network doctor` or `./bin/network health`.
- [ ] Git status confirms no untracked scratch files.
- [ ] Commit message follows Conventional Commits (`fix(<scope>): <description>`).

## Expected Result
The bug is fixed at the root cause, backed by a persistent regression test, and all healthcheck and doctor gates exit with status 0.
