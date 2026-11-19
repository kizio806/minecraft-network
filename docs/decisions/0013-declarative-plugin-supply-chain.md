# ADR 0013 - Declarative Plugin Supply Chain with SHA-512 Verification

## Status
Accepted

## Superseded By
None

## Date
2026-11-15

## Decision
Prohibit manual placement of plugin JAR files inside `servers/*/plugins/`. All plugins must be declared in `configs/plugins/manifest.yml`, resolved with exact download URLs and SHA-512 cryptographic digests in `configs/plugins/lock.yml`, and installed deterministically using the Go-based `plugin-manager` (`./bin/network install`).

## Reason
Manual plugin downloads from third-party sites present severe supply-chain attack risks (malicious JARs, backdoors), version drift across environments, and untracked binary bloat in Git. Enforcing a manifest and lockfile guarantees 100% reproducible builds, eliminates tampered JARs via SHA-512 checksum validation, and keeps the repository lightweight.

## Alternatives
- **Directly committing plugin JARs to Git:** Rejected because binary files bloat Git history, obscure changes, and bypass security verification.
- **Dynamic in-container downloads during startup:** Rejected because dynamic downloads cause network dependency failures, break offline repeatability, and introduce startup latency.

## Impact
- Zero raw JAR files committed to Git.
- Cryptographic verification of all plugins prior to container startup.
- Automated validation via `./bin/network plugins validate`.

## Verification
Validate manifest and lockfile cryptographic consistency:
```bash
./bin/network plugins validate
```
Command must exit with code 0.
