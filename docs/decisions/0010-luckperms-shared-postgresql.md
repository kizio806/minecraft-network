# ADR 0010: PostgreSQL as required shared LuckPerms storage

## Status

Accepted

## Context

ADR 0006 originally designated PostgreSQL as an optional `application-db` Compose profile, with no current consumer. After selecting LuckPerms v5.5.71-velocity as the approved network-wide permissions plugin, PostgreSQL became a required dependency because:

1. LuckPerms must share permissions, groups and user assignments across Proxy, Lobby and Survival.
2. Separate local storage (H2 or SQLite) per server would prevent synchronization and create inconsistent permission states when players move between backends.
3. LuckPerms SQL messaging requires a shared database for real-time cross-server permission updates.

## Decision

PostgreSQL is promoted from the optional `application-db` profile to a standard required service of the Minecraft Network.

### Database isolation

- PostgreSQL administrative credentials (`postgres_password`) remain reserved for the superuser and database maintenance.
- LuckPerms connects with a dedicated least-privilege user `luckperms_user` to a dedicated database `luckperms`.
- The `luckperms_password` secret is a separate Docker Secret, never shared with the administrative password.
- VeloAuth continues to use its own persistent SQLite database in the Proxy runtime volume. VeloAuth does not use PostgreSQL.
- No plugin shares credentials with another plugin or with the administrative superuser.

### Docker networking

- PostgreSQL remains exclusively on `data_net` (internal, no published ports).
- Proxy, Lobby and Survival are connected to both `proxy_net` and `data_net`.
- NanoLimbo (auth) remains on `proxy_net` only; it has no database dependency.
- Redis remains optional under the `cache` profile.

### Initialization

The init script `infrastructure/postgres/init-luckperms.sh` is mounted into `/docker-entrypoint-initdb.d/` and runs on first PostgreSQL startup. It creates the `luckperms_user` role and the `luckperms` database with least-privilege permissions using the `luckperms_password` Docker Secret.

### Backup integration

The existing `network backup` command already includes PostgreSQL dumps when the `postgres` container is running. Since PostgreSQL is now a standard service, backups always include the LuckPerms database.

## Supersedes

This decision supersedes the PostgreSQL-related portion of ADR 0006. ADR 0006's VeloAuth SQLite decision remains valid and is reaffirmed.

## Alternatives considered

- **H2 per-server**: Would prevent cross-server synchronization. Rejected.
- **SQLite per-server**: Same isolation problem. Rejected.
- **Redis-backed messaging only**: Does not provide shared storage; still requires a shared database. Redis remains optional for caching.
- **VeloAuth on PostgreSQL**: VeloAuth works correctly with SQLite for a single-proxy deployment. Adding a PostgreSQL dependency for authentication would increase complexity without benefit.

## Consequences

- PostgreSQL must be running before Proxy, Lobby or Survival start with LuckPerms enabled.
- The `luckperms_password` secret must be provisioned by `network install` alongside other secrets.
- Future plugins requiring PostgreSQL must use their own dedicated database and user; they must not reuse `luckperms_user` or the administrative superuser.
