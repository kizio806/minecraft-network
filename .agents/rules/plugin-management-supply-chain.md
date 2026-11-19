# Rule: Plugin Management & Supply Chain

## Purpose
Ensure deterministic, cryptographically verified, and reproducible plugin synchronization while eliminating supply-chain attacks and untracked binary bloat in Git.

## Scope
Applies to `configs/plugins/manifest.yml`, `configs/plugins/lock.yml`, `cmd/plugin-manager/`, `internal/pluginmanager/`, and all plugin directories.

## MUST
- **MUST** declare all plugins in `configs/plugins/manifest.yml` with project slug, exact version, platform, target servers, and license.
- **MUST** record resolved download URLs, filenames, sizes, and exact SHA-512 cryptographic hashes in `configs/plugins/lock.yml`.
- **MUST** verify SHA-512 checksums before extracting or installing any plugin JAR into `servers/*/plugins/`.
- **MUST** maintain declarative configuration templates in `configs/<server>/plugins/<plugin>/` with placeholders (`__PASSWORD__`) rather than hardcoded credentials.
- **MUST** run `./bin/network plugins validate` prior to committing any manifest or lockfile change.

## MUST NOT
- **MUST NOT** commit raw `.jar` files to Git.
- **MUST NOT** manually drop JAR files into `servers/*/plugins/` outside the automated `network install` command.
- **MUST NOT** accept floating versions (e.g. `latest`, `*`) in `manifest.yml`.
- **MUST NOT** introduce closed-source or unverified third-party plugins without explicit architectural approval.

## Verification
Validate manifest and lockfile cryptographic consistency:
```bash
./bin/network plugins validate
```
Command must exit with code 0.
