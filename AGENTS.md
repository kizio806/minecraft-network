# Minecraft Network — AI Agent Guidelines & Architecture Manual

## Purpose
This document is the authoritative entrypoint and navigation index for AI agents (Claude Code, Cursor, GitHub Copilot, Codex, Antigravity) and human engineers working on the **Minecraft Network** codebase.

## Scope
Governs all architectural boundaries, container configurations, Go tooling, plugin management, security standards, and operational workflows across the repository.

---

## 1. The 5 Golden Invariants (Hard Constraints)

Under no circumstance may an AI agent violate these five core architectural rules:

1. **Port Isolation:** The ONLY container permitted to publish ports to the public network is `proxy` (`0.0.0.0:25565:25565/tcp`). Game backends (`auth`, `lobby`, `survival`) and `postgres` must NEVER bind to host interfaces.
2. **Immutable Image Digests:** Every container image in `docker-compose.yml`, `.env.example`, and CI must be pinned to an immutable `@sha256:` digest. Mutable tags (`latest`, `alpine`, `java25`) are strictly forbidden.
3. **Declarative Plugin Supply Chain:** Never manually place JAR files into `servers/*/plugins/`. All plugins must be declared in `configs/plugins/manifest.yml`, verified with SHA-512 digests in `configs/plugins/lock.yml`, and synchronized via `./bin/network install`.
4. **Secret Protection:** Passwords, HMAC tokens, and private keys reside in `./secrets/` with strict mode `0600` (directory `0700`). Never commit secrets, and never inject passwords as plaintext string literals in Compose files.
5. **Database Consolidation & Zero Redis:** All persistent authentication and permissions reside in isolated schemas on PostgreSQL 16 Alpine. Redis is strictly banned across all services.
6. **Native Protocols & Zero Docker Socket Mounts:** Daemon containers such as `discord-bot` must NEVER mount `/var/run/docker.sock` or invoke the host Docker CLI. Inter-service administration must strictly use authenticated TCP network protocols (Source RCON, PostgreSQL wire protocol). All user arguments must be sanitized against command injection.

---

## 2. Directory Navigation & Context Router

Detailed documentation is modularized under `.agents/` by responsibility:

### Context & System Specifications (`.agents/context/`)
- [`architecture.md`](.agents/context/architecture.md) — System boundaries, proxy routing, game engine roles, and traffic flows.
- [`infrastructure.md`](.agents/context/infrastructure.md) — Docker Compose services, container resource formulas ($\ge 1.3 \times \text{Xmx}$), storage, and telemetry.
- [`security.md`](.agents/context/security.md) — Zero-Trust network segmentation, Docker Secrets, permissions, and age encryption.
- [`cli-reference.md`](.agents/context/cli-reference.md) — Complete command reference, global flags, and exit codes for `./bin/network`.
- [`plugin-matrix.md`](.agents/context/plugin-matrix.md) — Active plugin inventory, versioning, open-source licenses, and prohibited plugins.
- [`decision-index.md`](.agents/context/decision-index.md) — Architectural decisions index and summary for AI agents.
- [`development.md`](docs/development.md) — Developer setup, local testing, and build targets.
- [`discord-bot.md`](docs/discord-bot.md) — Discord daemon architecture, privileged intents, and TCP RCON.
- **Master ADR Repository:** [`docs/decisions/`](docs/decisions/) — The single source of truth for all Architecture Decision Records.

### Agent Roles (`.agents/roles/`)
Assume the appropriate persona for specialized tasks:
- [`architect.md`](.agents/roles/architect.md) — Principal Architect: system boundaries, ADRs, trade-offs, and memory budgets.
- [`infra-devops.md`](.agents/roles/infra-devops.md) — DevOps Engineer: Docker Compose, pinned digests, CI/CD, and host scripts.
- [`plugin-specialist.md`](.agents/roles/plugin-specialist.md) — Plugin Specialist: manifest, lockfile, templates, and gameplay integrity.
- [`security-auditor.md`](.agents/roles/security-auditor.md) — Security Auditor: secret mode `0600`, zero port leaks, and age encryption.
- [`qa-tester.md`](.agents/roles/qa-tester.md) — QA Engineer: Go unit tests with `-race`, healthcheck audits, and staging verification.

### Domain Rules (`.agents/rules/`)
Consult mandatory rules before modifying code:
- [`docker-and-compose-topology.md`](.agents/rules/docker-and-compose-topology.md) — Container limits, network isolation, pinned digests.
- [`security-secrets-and-permissions.md`](.agents/rules/security-secrets-and-permissions.md) — File mode `0600`, Modern forwarding HMAC, age keys.
- [`go-standards-and-cli.md`](.agents/rules/go-standards-and-cli.md) — Idiomatic Go 1.24+, race immunity, Zip Slip prevention.
- [`plugin-management-supply-chain.md`](.agents/rules/plugin-management-supply-chain.md) — Manifest, lockfile, SHA-512 verification.
- [`database-and-persistence.md`](.agents/rules/database-and-persistence.md) — PostgreSQL consolidation, user isolation, zero Redis.
- [`documentation-and-commits.md`](.agents/rules/documentation-and-commits.md) — Conventional Commits, single source of truth.
- [`multi-agent-coordination.md`](.agents/rules/multi-agent-coordination.md) — Multi-agent boundaries, lock avoidance, atomic edits.
- [`gameplay-and-world-integrity.md`](.agents/rules/gameplay-and-world-integrity.md) — Survival mechanics, spawn protection, zero world saves in git.

### Workflows (`.agents/workflows/`)
Follow structured lifecycles for repeatable procedures:
- [`add-plugin.md`](.agents/workflows/add-plugin.md) — Vetting, declaring, locking, and configuring a plugin.
- [`bug-fix.md`](.agents/workflows/bug-fix.md) — Root cause analysis, regression testing, fix, and verification.
- [`disaster-recovery.md`](.agents/workflows/disaster-recovery.md) — Encrypted backup generation and atomic restoration.
- [`infrastructure-change.md`](.agents/workflows/infrastructure-change.md) — Modifying Compose, networks, or memory ceilings.
- [`new-feature.md`](.agents/workflows/new-feature.md) — RFC review, implementation, unit testing, and documentation.
- [`release.md`](.agents/workflows/release.md) — Quality gates, staging smoke tests, tagging, and release notes.

### Interactive Skills (`.agents/skills/`)
Execute standardized operational procedures:
- [`verify-network/SKILL.md`](.agents/skills/verify-network/SKILL.md) — Full automated doctor, test, and health validation.
- [`security-scan/SKILL.md`](.agents/skills/security-scan/SKILL.md) — Audit file permissions, secret leakage, and network isolation.
- [`audit-codebase/SKILL.md`](.agents/skills/audit-codebase/SKILL.md) — Architectural audit of manifests vs. runtime.
- [`plugin-sync/SKILL.md`](.agents/skills/plugin-sync/SKILL.md) — Safe download and synchronization of server plugins.
- [`backup-drill/SKILL.md`](.agents/skills/backup-drill/SKILL.md) — Backup creation, age encryption test, and restore drill.
- [`console-inspect/SKILL.md`](.agents/skills/console-inspect/SKILL.md) — Interactive console access, log streaming, and JVM metrics.

---

## 3. Mandatory Quality Gate

Before concluding any task or committing changes, execute the verification suite:

```bash
# 1. System doctor audit (secrets, files, docker, compose)
./bin/network doctor

# 2. Container healthchecks (all running services must be healthy)
./bin/network health

# 3. Plugin manifest and lockfile cryptographic integrity
./bin/network plugins validate

# 4. Go unit test suite with race detector
docker run --rm -v "$(pwd)":/app -w /app golang:1.24 go test -v -race ./...

# 5. Docker Compose syntax validation
docker compose -f docker-compose.yml config --quiet
docker compose -f docker-compose.yml --profile monitoring config --quiet
```

All 5 commands must pass with exit code 0.
