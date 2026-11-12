# ADR 0006: SQLite for VeloAuth authentication state

## Status

Superseded in part by [ADR 0010](0010-luckperms-shared-postgresql.md) (PostgreSQL promoted from optional profile to required shared storage for LuckPerms; SQLite decision for VeloAuth reaffirmed).

## Decision

Use VeloAuth's persistent SQLite database in the Proxy runtime volume for the single-proxy, approximately 20-player deployment. VeloAuth does not use PostgreSQL. Redis remains an opt-in cache profile. (Note: PostgreSQL itself was promoted to a required service in ADR 0010 specifically for network-wide LuckPerms permissions storage).

## Rationale

VeloAuth supports SQLite and local databases. A remote database for authentication would add credentials, network dependency, pool operations and another failure mode without necessity. SQLite is included in the existing world/config backup and restored before services start.

## Consequences

Do not run multiple active Proxy instances against this SQLite file. If high availability or multiple authentication writers becomes a requirement, migrate to PostgreSQL in a tested staging migration with a defined locking and rollback plan. VeloAuth database credentials remain completely separate from LuckPerms and administrative credentials.
