# ADR 0011 - Complete PostgreSQL Consolidation for Authentication and Permissions

## Status
Accepted

## Superseded By
None

## Date
2026-11-15

## Decision
Consolidate network-wide permissions (LuckPerms) and hybrid player authentication (VeloAuth) into a single PostgreSQL 16 Alpine container with isolated relational schemas (`luckperms` and `veloauth`), each assigned a distinct least-privilege database user (`luckperms_user` and `veloauth_user`). Unmanaged SQLite files are completely removed from production runtime.

This formally supersedes ADR 0006 and extends ADR 0010.

## Reason
Running unmanaged SQLite files inside server containers leads to file locking contention, lack of transactional consistency across proxy restarts, and severe backup difficulties during live gameplay. Storing auth and permissions in separate databases on a single PostgreSQL instance provides strict relational ACID guarantees, transactional schema migrations, and atomic backup capabilities (`pg_dumpall`).

## Alternatives
- **Separate SQLite files per container (ADR 0006 approach):** Rejected due to lack of network-wide synchronization, risk of database corruption on abrupt restarts, and backup locking issues.
- **Multiple database containers (e.g. MySQL for LuckPerms + SQLite for VeloAuth):** Rejected because running multiple database engines wastes memory and CPU overhead on a 16 GB host.
- **MongoDB / NoSQL:** Rejected as unnecessary; relational tables are natively supported by official LuckPerms and VeloAuth drivers.

## Impact
- Single relational database container (`postgres`) consuming ~512 MB RAM.
- Complete isolation between `luckperms_user` and `veloauth_user`.
- Relational schema backups are performed atomically in the `./bin/network backup` routine.
- Zero local database corruption risk during abrupt container restarts.

## Verification
Inspect database schemas and user isolation:
```bash
docker compose exec postgres psql -U postgres -c "\l"
./bin/network doctor
```
Databases `luckperms` and `veloauth` must exist with their respective owners, and `doctor` must pass.
