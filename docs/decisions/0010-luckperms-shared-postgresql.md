# ADR 0010 - Shared PostgreSQL Storage for LuckPerms

## Status
Accepted

## Superseded By
None

## Date
2026-11-12

## Decision
Promote PostgreSQL 16 Alpine from an optional Compose profile to a mandatory, standard service of the Minecraft Network to serve as the shared relational storage and messaging backend for LuckPerms v5.5.71 across Proxy, Lobby, and Survival servers. Dedicated database user `luckperms_user` connects to database `luckperms` over `data_net`.

## Reason
LuckPerms requires real-time permission and group synchronization across proxy and backend server instances. Local file storage (H2 or SQLite per server) isolates permissions, causing desynchronization when players traverse between servers. LuckPerms SQL messaging provides atomic cross-server updates over a shared database without requiring Redis.

## Alternatives
- **Local H2/SQLite per server:** Rejected because permissions created on Proxy or Lobby do not propagate to Survival.
- **MySQL / MariaDB:** Rejected in favor of PostgreSQL 16 Alpine, which provides superior container performance, smaller image footprint, and transactional schema safety.

## Impact
- PostgreSQL is a core required service in `docker-compose.yml`.
- `luckperms_user` has isolated least-privilege permissions on database `luckperms`.
- Atomic schema dumps included in `./bin/network backup`.
- Note: This decision was subsequently extended by ADR 0011 to also consolidate VeloAuth into PostgreSQL.

## Verification
Inspect database users and verify LuckPerms connectivity:
```bash
docker compose exec postgres psql -U postgres -d luckperms -c "\dt"
./bin/network doctor
```
Database `luckperms` must exist with populated LuckPerms relation tables.
