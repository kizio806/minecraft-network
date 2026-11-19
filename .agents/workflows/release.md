# Workflow: Preparing a Release

## Purpose & Scope
Define the standard verification gauntlet, versioning, staging smoke tests, and git tagging procedure for production or milestone releases.

## When to Use
Use this workflow whenever cutting a new stable version or milestone tag of the Minecraft Network platform.

## Steps

### Step 1: Pre-Release Quality Gates
1. Run complete system audit:
   ```bash
   ./bin/network doctor
   ./bin/network health
   ./bin/network plugins validate
   docker run --rm -v "$(pwd)":/app -w /app golang:1.24 go test -v -race ./...
   docker compose -f docker-compose.yml config --quiet
   docker compose -f docker-compose.yml --profile monitoring config --quiet
   ```

### Step 2: Security & Clean State Verification
1. Verify all container images are pinned to `@sha256:` immutable digests.
2. Confirm git working tree is clean:
   ```bash
   git status -s
   ```
   No untracked scratch files, session locks, or unencrypted keys may exist.

### Step 3: Staging Backup Drill
1. Create pre-release encrypted backup:
   ```bash
   ./bin/network backup
   ```
2. Verify archive restorability via dry-run:
   ```bash
   LATEST=$(ls -t backups/*.tar.gz.age | head -n 1)
   ./bin/network restore "$LATEST" --dry-run
   ```

### Step 4: Version Bump & Release Notes
1. Update version constants and changelog entries.
2. Recompile final production binaries:
   ```bash
   go build -trimpath -ldflags="-s -w" -o bin/network ./cmd/network
   go build -trimpath -ldflags="-s -w" -o bin/plugin-manager ./cmd/plugin-manager
   ```

### Step 5: Tagging & Release Commit
1. Commit release metadata:
   ```bash
   git commit -m "chore(release): v1.X.Y"
   ```
2. Create annotated git tag:
   ```bash
   git tag -a v1.X.Y -m "Release v1.X.Y"
   ```

## Required Checks
- [ ] 100% test pass rate with `-race`.
- [ ] All containers healthy and doctor reports all systems `OK`.
- [ ] Staging backup and dry-run restore complete with exit code 0.
- [ ] Zero unpinned images or plaintext secrets in git history.

## Expected Result
A cryptographically verified, fully tested, tagged release ready for production deployment.
