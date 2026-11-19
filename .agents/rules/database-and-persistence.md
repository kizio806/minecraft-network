# Rule: Database & Persistence

## Purpose
Consolidate network persistence, ensure schema isolation, guarantee transactional ACID properties, and maintain a strict zero-Redis architecture.

## Scope
Applies to PostgreSQL configuration (`infrastructure/postgres/`), Docker Compose database volumes, LuckPerms storage, and VeloAuth authentication state.

## MUST
- **MUST** store all persistent player authentication (VeloAuth) and network permissions (LuckPerms) in dedicated schemas on the PostgreSQL 16 Alpine container.
- **MUST** isolate database access using separate, least-privilege users (`luckperms_user` and `veloauth_user`) with dedicated passwords.
- **MUST** attach PostgreSQL exclusively to the internal Docker network `data_net` (`internal: true`).
- **MUST** dump all relational databases atomically (`pg_dumpall`) during the `./bin/network backup` process.
- **MUST** store PostgreSQL data on the named Docker volume `postgres-data`.

## MUST NOT
- **MUST NOT** deploy or introduce Redis, KeyDB, or Valkey caching containers.
- **MUST NOT** use unmanaged SQLite files for proxy-wide authentication or permissions.
- **MUST NOT** grant superuser (`postgres`) credentials to game servers or plugin configuration templates.
- **MUST NOT** expose PostgreSQL port `5432` to the host network interface.

## Verification
Inspect database users and verify internal connectivity:
```bash
docker compose exec postgres psql -U postgres -c "\l"
./bin/network doctor
```
Databases `luckperms` and `veloauth` must exist with their respective owners, and `doctor` must pass.
