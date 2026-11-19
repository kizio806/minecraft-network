# Architecture Decision Records (ADR)

This directory is the **single source of truth** for all architectural and technical decisions governing the **Minecraft Network** project.

Every decision record documents the context, alternatives considered, technical trade-offs, and compliance verification.

## Decision Register

| ADR | Title | Date | Status | Superseded By |
| :---: | :--- | :---: | :---: | :--- |
| **[0001](0001-hybrid-authentication.md)** | Hybrid Premium and Non-Premium Authentication | 2026-11-12 | `Accepted` | None |
| **[0002](0002-network-segmentation.md)** | Zero-Trust Network Segmentation and Port Isolation | 2026-11-12 | `Accepted` | None |
| **[0003](0003-hybrid-uuid-strategy.md)** | Hybrid Player Identity and UUID Strategy | 2026-11-12 | `Accepted` | None |
| **[0004](0004-backend-engine.md)** | Purpur as the Backend Game Engine | 2026-11-12 | `Accepted` | None |
| **[0005](0005-proxy-and-forwarding.md)** | Velocity Modern Forwarding | 2026-11-12 | `Accepted` | None |
| **[0006](0006-auth-database.md)** | SQLite for VeloAuth Authentication State | 2026-11-12 | `Deprecated` | [ADR 0011](0011-postgresql-consolidation.md) |
| **[0007](0007-monitoring-stack.md)** | Observability and Monitoring Stack | 2026-11-12 | `Accepted` | None |
| **[0008](0008-backup-strategy.md)** | Asymmetric Encrypted Backup Strategy | 2026-11-12 | `Accepted` | None |
| **[0009](0009-docker-networking.md)** | Docker Network Boundaries | 2026-11-12 | `Deprecated` | [ADR 0002](0002-network-segmentation.md) |
| **[0010](0010-luckperms-shared-postgresql.md)** | Shared PostgreSQL Storage for LuckPerms | 2026-11-12 | `Accepted` | Extended by [ADR 0011](0011-postgresql-consolidation.md) |
| **[0011](0011-postgresql-consolidation.md)** | Complete PostgreSQL Consolidation for Authentication & Permissions | 2026-11-15 | `Accepted` | None |
| **[0012](0012-zero-redis.md)** | Strict Zero-Redis Architectural Policy | 2026-11-15 | `Accepted` | None |
| **[0013](0013-declarative-plugin-supply-chain.md)** | Declarative Plugin Supply Chain with SHA-512 Verification | 2026-11-15 | `Accepted` | None |
| **[0014](0014-monitoring-profile-isolation.md)** | Monitoring Stack Isolation Behind Compose Profiles | 2026-11-15 | `Accepted` | None |
| **[0015](0015-server-side-datapacks-dimension-overhauls.md)** | Server-Side Datapacks for Dimension Overhauls (Incendium & Nullscape) | 2026-11-15 | `Accepted` | None |
| **[0016](0016-terralith-vs-terraformgenerator-worldgen-profiles.md)** | Standard Overworld Generation via TerraformGenerator & Deprecation of Terralith | 2026-11-15 | `Accepted` | None |

---

## ADR Template Standard

Every record in this directory MUST follow the unified format:

```markdown
# ADR XXXX - Title

## Status
Accepted | Deprecated | Superseded

## Superseded By
None | ADR YYYY - Decision Title

## Date
YYYY-MM-DD

## Decision
Exact solution chosen.

## Reason
Detailed architectural rationale.

## Alternatives
Options evaluated and reasons for rejection.

## Impact
Operational and infrastructural consequences.

## Verification
Automated or manual procedure to verify compliance.
```
