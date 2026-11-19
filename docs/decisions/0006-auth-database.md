# ADR 0006 - SQLite for VeloAuth Authentication State

## Status
Deprecated

## Superseded By
ADR 0011 - Complete PostgreSQL Consolidation for Authentication and Permissions

## Date
2026-11-12

## Decision
Originally, VeloAuth was configured to store persistent authentication state in a local SQLite file within the Proxy runtime volume (`servers/proxy/plugins/VeloAuth/veloauth.db`), while PostgreSQL was kept as an optional profile.

## Reason
Initial rationale assumed that running SQLite locally avoided network dependencies and database credentials for a small deployment, keeping auth state inside server backups.

## Alternatives
- **PostgreSQL in optional profile:** Considered in ADR 0006, but initially deferred to minimize infrastructure complexity.
- **MySQL / MariaDB:** Rejected in favor of SQLite for zero-configuration local storage.

## Impact
- Local SQLite caused file lock contention on rapid restarts.
- Lacked transactional consistency guarantees during live backups while the proxy was running.
- Prevented unified relational database disaster recovery alongside LuckPerms.
- Formally superseded by ADR 0011, which consolidated VeloAuth into a dedicated PostgreSQL schema (`veloauth`).

## Verification
Confirm that VeloAuth no longer uses SQLite:
```bash
./bin/network doctor
# Inspect proxy VeloAuth configuration:
grep "database" configs/proxy/plugins/veloauth/config.yml
```
Configuration must specify PostgreSQL connection parameters rather than SQLite file paths.
