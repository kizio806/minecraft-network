# ADR 0012 - Strict Zero-Redis Architectural Policy

## Status
Accepted

## Superseded By
None

## Date
2026-11-15

## Decision
Enforce a strict zero-Redis architectural rule across the entire infrastructure. Redis is prohibited from Docker Compose, server plugin configurations, and caching tiers.

## Reason
For a single-proxy network topology supporting ~20 concurrent players on a 16 GB RAM host, Velocity natively handles modern forwarding, tablist synchronization, and server switching in-memory. Velocity does not require RedisBungee. VeloAuth and LuckPerms maintain efficient local in-memory caches backed directly by PostgreSQL connections. Adding a Redis container would introduce an unnecessary failure mode, consume RAM budget, and require extra secrets and maintenance without any tangible performance gain.

## Alternatives
- **Redis for cross-server messaging & caching:** Rejected because Velocity natively manages inter-server player transfers and Modern Forwarding handles session handoffs.
- **Valkey / KeyDB:** Rejected for the same reasons as Redis; standard JVM memory and PostgreSQL connection pooling are sufficient.

## Impact
- Saves ~256 MB host RAM and eliminates cache invalidation bugs.
- Simplifies operational lifecycle and disaster recovery (only PostgreSQL and filesystem state to restore).
- Hard constraint: no plugin may introduce Redis dependencies unless explicitly approved via architectural RFC.

## Verification
Confirm absence of Redis containers and configurations:
```bash
grep -in "redis" docker-compose.yml configs/network.yml configs/plugins/manifest.yml
```
Command must return zero active Redis service definitions.
