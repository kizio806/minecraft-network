# Backup and Disaster Recovery

Automated cryptographic backups and atomic restores are managed via the native Go CLI (`network backup` and `network restore`).

---

## 1. Backup Strategy & Cryptography

The backup process ensures consistent state across running services:
1. Signals Proxy, Lobby, and Survival servers to flush world caches.
2. Executes a consistent PostgreSQL database dump (`pg_dump`) covering `luckperms`, `veloauth`, `ultimateteams`, and `libertybans`.
3. Packages configuration templates, world directories, server settings, and secrets into a compressed archive (`tar.gz`).
4. Encrypts the archive using modern asymmetric cryptography via **age** (`filippo.io/age`).
5. Generates an accompanying SHA-256 checksum file (`.sha256`) for immutable integrity verification.

### Key Separation Principle
- **Production Server:** Holds **only the public recipient key** (`BACKUP_AGE_RECIPIENT_FILE`). The server can encrypt backups but cannot decrypt them.
- **Offsite / Administrator Storage:** Holds the **private identity key** (`BACKUP_AGE_IDENTITY_FILE`). Required only during disaster recovery drills or emergency restores.

---

## 2. Backup CLI Commands

Create an encrypted backup:
```bash
./bin/network backup
```

List existing backups and checksums:
```bash
./bin/network backup list
```

Restore from an encrypted backup:
```bash
./bin/network restore /var/backups/minecraft-network/minecraft-network-<timestamp>.tar.gz.age --yes
```

The restore operation:
1. Requires explicit `--yes` confirmation flag.
2. Verifies the SHA-256 checksum of the target archive.
3. Decrypts the archive using `BACKUP_AGE_IDENTITY_FILE` with strict path traversal (Zip Slip / Tar Slip) protection.
4. Safely stops backend services, extracts configurations and world directories, imports the PostgreSQL dump, and restarts services.
