---
name: backup-drill
description: Run an end-to-end backup creation, cryptographic encryption verification, and atomic restoration test.
---

# Skill: Backup Drill

## Use Case
Execute this skill periodically or before significant infrastructure migrations to confirm that the automated backup and atomic restoration pipeline is fully functional.

## Inputs & Prerequisites
- Functioning `./bin/network` binary.
- Encryption keys in `./secrets/backup.age-identity` and `./secrets/backup.age-recipient` (mode `0600`).
- Running PostgreSQL container (`postgres`) for database dump generation.

## Procedure

### 1. Trigger Automated Encrypted Backup
```bash
./bin/network backup
```

### 2. Verify Output Archive Header
```bash
LATEST=$(ls -t backups/*.tar.gz.age | head -n 1)
ls -lh "$LATEST"
file "$LATEST"
```
The file must be non-empty and reported as `age encrypted data`.

### 3. Execute Dry-Run Restoration
```bash
./bin/network restore "$LATEST" --dry-run
```

### 4. Execute Full Staging Restoration (Drill Mode Only)
```bash
./bin/network stop
./bin/network restore "$LATEST" --yes
./bin/network start --monitoring
./bin/network health
```

## Validation
- `./bin/network backup` exits with code 0 and produces a valid archive in `backups/`.
- Dry-run decryption verifies archive readability without filesystem changes.
- In staging drill: all 10 containers start cleanly and report `healthy`.

## Limitations & Edge Cases
- Do not run live restore (`--yes`) in production while players are connected; always run in staging or during scheduled maintenance windows.
- Requires recipient key for encryption and identity key for decryption.
