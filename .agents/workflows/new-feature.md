# Workflow: Implementing a New Feature

## Purpose & Scope
Define the standard lifecycle for designing, reviewing, implementing, testing, and documenting a new feature in Go CLI tooling or infrastructure services.

## When to Use
Use this workflow whenever introducing new CLI subcommands, new configuration capabilities, automated tooling, or major system enhancements.

## Steps

### Step 1: Architectural Proposal & Alignment
1. Check alignment with [`PROJECT.md`](../../PROJECT.md), [`AGENTS.md`](../../AGENTS.md), and architectural decisions in [`docs/decisions/`](../../docs/decisions/).
2. Confirm zero port exposure, adherence to the 12 GB RAM budget, and zero Redis.
3. Present RFC/proposal and obtain approval before coding.

### Step 2: Interface & Schema Design
1. Design CLI subcommands with descriptive help text and standardized exit codes (`0`, `2`, `3`, `4`, `5`).
2. Provide sensible defaults and backward-compatible schemas in configuration files.

### Step 3: Implementation
1. Write idiomatic Go 1.24+ in standard package layout (`cmd/` and `internal/`).
2. Implement explicit error handling and path traversal protection (`filepath.Clean`).
3. Compile with `-trimpath -ldflags="-s -w"`.

### Step 4: Comprehensive Testing
1. Add table-driven unit tests in `*_test.go`.
2. Run test suite with the race detector:
   ```bash
   docker run --rm -v "$(pwd)":/app -w /app golang:1.24 go test -v -race ./...
   ```
3. Run doctor and healthchecks:
   ```bash
   ./bin/network doctor
   ./bin/network health
   ```

### Step 5: Documentation & Commit
1. Update `.agents/context/cli-reference.md` with new flags or commands.
2. Commit with Conventional Commits: `feat(<scope>): <description>`.

## Required Checks
- [ ] All unit tests pass with zero data races.
- [ ] Binaries compile cleanly with `-trimpath`.
- [ ] `./bin/network doctor` and `./bin/network health` report all systems healthy.
- [ ] Relevant documentation files are updated.

## Expected Result
A robust, idiomatic, fully tested, and documented feature integrated cleanly without regressions.
