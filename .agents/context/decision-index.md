# Architecture Decisions Index

## Single Source of Truth Notice
The **authoritative single source of truth** for all Architecture Decision Records (ADRs) is located in:
> [`docs/decisions/`](../../docs/decisions/)

AI agents must **never** duplicate, create separate decision files in `.agents/`, or alter decisions without updating the formal ADR register in `docs/decisions/`.

---

## Key Architectural Decisions Summary for AI Agents

| ADR | Title | Status | Core Rule & Summary for AI |
| :---: | :--- | :---: | :--- |
| **[0001](../../docs/decisions/0001-hybrid-authentication.md)** | Hybrid Authentication | `Accepted` | VeloAuth on Velocity + NanoLimbo. Mojang handshake for Premium; strict mixed mode (`allow-cracked-on-premium-nicks=false`). |
| **[0002](../../docs/decisions/0002-network-segmentation.md)** | Network & Port Isolation | `Accepted` | **ONLY** `proxy` publishes port `25565:25565` to `0.0.0.0`. Game backends (`auth`, `lobby`, `survival`) and `postgres` must NEVER bind host ports. `data_net` is `internal: true`. |
| **[0003](../../docs/decisions/0003-hybrid-uuid-strategy.md)** | Hybrid UUID Strategy | `Accepted` | Premium accounts use Mojang UUID; Non-Premium use offline deterministic UUID. |
| **[0004](../../docs/decisions/0004-backend-engine.md)** | Purpur Backend Engine | `Accepted` | Purpur 1.21.11 on Java 25. Folia is strictly rejected due to plugin incompatibility. |
| **[0005](../../docs/decisions/0005-proxy-and-forwarding.md)** | Modern Forwarding | `Accepted` | Velocity Modern Forwarding with shared HMAC secret (`forwarding_secret`). Legacy Bungee forwarding forbidden. |
| **[0006](../../docs/decisions/0006-auth-database.md)** | SQLite for VeloAuth | `Deprecated` | **Superseded by ADR 0011**. SQLite is completely removed from runtime. |
| **[0007](../../docs/decisions/0007-monitoring-stack.md)** | Monitoring Stack | `Accepted` | Prometheus, Grafana, Node Exporter, Minecraft Exporter, Uptime Kuma. |
| **[0008](../../docs/decisions/0008-backup-strategy.md)** | Encrypted Backup Strategy| `Accepted` | Atomic PostgreSQL dump + world saves compressed with gzip and encrypted with asymmetric `age` (X25519). |
| **[0009](../../docs/decisions/0009-docker-networking.md)** | Docker Network Boundaries| `Deprecated` | **Superseded by ADR 0002**. |
| **[0010](../../docs/decisions/0010-luckperms-shared-postgresql.md)** | Shared PostgreSQL for LP | `Accepted` | PostgreSQL as required storage for network-wide LuckPerms permissions. Extended by ADR 0011. |
| **[0011](../../docs/decisions/0011-postgresql-consolidation.md)** | PostgreSQL Consolidation | `Accepted` | PostgreSQL 16 Alpine manages isolated relational schemas `luckperms` and `veloauth`. Separate least-privilege users. Zero SQLite. |
| **[0012](../../docs/decisions/0012-zero-redis.md)** | Strict Zero-Redis Policy | `Accepted` | Redis is strictly banned across all containers, plugins, and caching layers. |
| **[0013](../../docs/decisions/0013-declarative-plugin-supply-chain.md)** | Declarative Plugin Supply Chain | `Accepted` | Zero manual JARs in `servers/*/plugins/`. All plugins declared in `manifest.yml` and verified via SHA-512 in `lock.yml`. |
| **[0014](../../docs/decisions/0014-monitoring-profile-isolation.md)** | Monitoring Profile Isolation | `Accepted` | Telemetry containers gated behind Compose `profiles: [monitoring]`. Activated optionally via `--monitoring` flag. |
| **[0015](../../docs/decisions/0015-server-side-datapacks-dimension-overhauls.md)** | Server-Side Datapacks | `Accepted` | Nether (Incendium) and End (Nullscape) overhauls run 100% server-side in `world/datapacks/`. Zero client downloads. |
| **[0016](../../docs/decisions/0016-terralith-vs-terraformgenerator-worldgen-profiles.md)** | TerraformGenerator Overworld & Deprecation of Terralith | `Accepted` | TerraformGenerator selected as standard Overworld engine; Terralith removed to eliminate noise codec conflicts and preserve chunk integrity. |

---

## Verification
To inspect the master register of architectural decisions:
```bash
ls -la docs/decisions/
cat docs/decisions/README.md
```
