# ADR 0008 - Asymmetric Encrypted Backup Strategy

## Status
Accepted

## Superseded By
None

## Date
2026-11-12

## Decision
Implement disaster recovery using the native Go CLI (`./bin/network backup`), producing gzip-compressed archives encrypted with modern asymmetric `age` (X25519) cryptography. Backups capture an atomic PostgreSQL dump (`pg_dumpall`), server world saves, and plugin configuration templates. Restoration is executed via `./bin/network restore <archive> --yes` using the private identity key (`backup.age-identity`).

## Reason
Traditional unencrypted backups stored on the local host are vulnerable to data tampering, host compromise, and credential extraction. Asymmetric encryption ensures that the server can create backups using a public recipient key (`backup.age-recipient`) without requiring the private decryption key to reside in runtime memory during production gameplay.

## Alternatives
- **Plain unencrypted tar.gz:** Rejected because backups contain full database dumps with hashed user passwords, HMAC secrets, and world data.
- **Symmetric GPG encryption with passphrase:** Rejected because storing a symmetric passphrase on disk or in environment variables allows anyone with server access to decrypt all past backups.

## Impact
- Encryption public key resides in `./secrets/backup.age-recipient` (mode `0600`).
- Decryption private key resides in `./secrets/backup.age-identity` (mode `0600`).
- Backups are written to `backups/backup-<timestamp>.tar.gz.age`.
- Restorations can be tested non-destructively via `--dry-run`.

## Verification
Execute the automated backup command and verify encryption header:
```bash
./bin/network backup
LATEST=$(ls -t backups/*.tar.gz.age | head -n 1)
file "$LATEST"
./bin/network restore "$LATEST" --dry-run
```
Archive must be identified as `age encrypted data` and dry-run restore must succeed with code 0.
