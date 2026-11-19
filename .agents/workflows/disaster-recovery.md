# Workflow: Disaster Recovery

## Purpose & Scope
Define the standard operational procedure for creating encrypted backups, validating archive integrity, and executing atomic restorations across PostgreSQL schemas and Minecraft world saves.

## When to Use
Use this workflow before executing major server updates, migrating host infrastructure, after critical system failures, or during scheduled quarterly disaster recovery drills.

## Steps

### Step 1: Pre-flight Verification
1. Ensure age encryption keys exist with mode `0600`:
   ```bash
   ls -la secrets/backup.age-*
   ```
2. Verify sufficient disk storage on the host filesystem.

### Step 2: Execute Automated Backup
1. Trigger the Go CLI backup routine:
   ```bash
   ./bin/network backup
   ```
2. The CLI performs:
   - Atomic PostgreSQL database dump (`pg_dumpall`).
   - World and plugin configuration archiving.
   - Streaming gzip compression and asymmetric `age` encryption.
   - Writing output archive to `backups/backup-<timestamp>.tar.gz.age`.

### Step 3: Archive Validation
1. Verify the archive file is non-empty and identified as age encrypted data:
   ```bash
   ls -lh backups/
   file backups/backup-*.tar.gz.age | tail -n 1
   ```

### Step 4: Dry-Run Restoration
1. Verify the archive can be decrypted and unpacked without touching live files:
   ```bash
   LATEST=$(ls -t backups/*.tar.gz.age | head -n 1)
   ./bin/network restore "$LATEST" --dry-run
   ```

### Step 5: Live Restoration (If in Recovery or Drill Mode)
1. Stop running containers:
   ```bash
   ./bin/network stop
   ```
2. Execute atomic restoration:
   ```bash
   ./bin/network restore "$LATEST" --yes
   ```
3. Restart containers and verify health:
   ```bash
   ./bin/network start --monitoring
   ./bin/network health
   ```

## Required Checks
- [ ] Backup archive exists and contains valid age encrypted headers.
- [ ] Dry-run decryption succeeds without error.
- [ ] In a live drill: all services start cleanly and `./bin/network health` reports `healthy`.
- [ ] PostgreSQL schemas (`luckperms`, `veloauth`) contain all pre-backup data intact.

## Expected Result
Full state recovery of PostgreSQL databases, server configs, and player worlds with zero data loss or database corruption.
