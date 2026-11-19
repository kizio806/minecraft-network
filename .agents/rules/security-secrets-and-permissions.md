# Rule: Security, Secrets & Permissions

## Purpose
Prevent credential leakage, enforce strict filesystem permissions on cryptographic secrets, and maintain zero-trust communication between proxy and backend servers.

## Scope
Applies to the `./secrets/` directory, `.env` file, Velocity Modern Forwarding, Docker Secrets, and backup encryption keys.

## MUST
- **MUST** enforce directory mode `0700` (`drwx------`) on `./secrets/`.
- **MUST** enforce file mode `0600` (`-rw-------`) on all secret files inside `./secrets/`.
- **MUST** inject passwords and sensitive tokens using native Docker Secrets (`/run/secrets/*`) or runtime templating (`__PLACEHOLDER__`) rendered into git-ignored `servers/` directories.
- **MUST** configure Velocity Modern Forwarding with a shared HMAC secret between `proxy` and all backend servers.
- **MUST** encrypt all backup archives with asymmetric `age` (X25519) encryption before writing to disk.
- **MUST** ensure `.gitignore` excludes `secrets/`, `.env`, and unencrypted credentials from version control.

## MUST NOT
- **MUST NOT** commit plaintext passwords, tokens, private keys, or `.env` files into Git.
- **MUST NOT** inject database passwords directly as string literals inside `docker-compose.yml` `environment:` blocks.
- **MUST NOT** accept secret files with permissive file permissions (e.g. `0644` or `0777`).
- **MUST NOT** disable modern forwarding or use legacy BungeeCord IP forwarding without HMAC signature validation.

## Verification
Execute the automated doctor audit and secret permission check:
```bash
./bin/network doctor
stat -c "%a %n" secrets/*
```
The doctor check must report `secrets OK` and all secret files must have mode `600`.
