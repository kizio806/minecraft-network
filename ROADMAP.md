# Project Roadmap — Minecraft Network

This roadmap outlines key milestones, active architectural tracks, and future vision for the **Minecraft Network** production infrastructure platform.

---

## Current Release Track (v1.0.x — Stability & Security Hardening)
- [x] **Native RCON Client:** Decouple Discord bot from external Docker CLI binary by implementing pure Go Source RCON TCP protocol.
- [x] **Input Sanitization:** Strict regex username gating and command sanitization against injection in moderation channels.
- [x] **Database Alignment:** Direct Discord bot ranking sync to dedicated `luckperms` schema on consolidated PostgreSQL container.
- [x] **Plugin Supply Chain Hardening:** Fail-fast diagnostics for missing secret passwords and hash-verified plugin downloads.
- [x] **Comprehensive Testing:** Go unit test suite with `-race` detection and Docker Compose configuration validation.

---

## Upcoming Milestones

### Phase 2: Observability & Auto-Healing (v1.1.0)
- [ ] **Grafana Dashboard Pack:** Pre-configured Grafana dashboards for Velocity bandwidth, Paper tick times (MSPT), and PostgreSQL query latency.
- [ ] **Automated Backup Verification:** Scheduled backup verification drill container that validates restic/tar archives against test restores.
- [ ] **Discord Health Notifications:** Webhook alerting for server crashes, memory pressure, or database disconnections.

### Phase 3: Dynamic Scaling & Modern Minecraft Matrix (v1.2.0)
- [ ] **Minecraft 26.x Version Track:** Evaluate Paper and Velocity upstream builds for latest Minecraft feature drops.
- [ ] **Zero-Downtime Proxy Reloads:** Automated graceful routing migrations during proxy configuration changes.
- [ ] **CLI Expansion:** `networkcli status --live` terminal dashboard with real-time player counts and node health metrics.

---

## Long-Term Vision
- **Multi-Region Sync:** Multi-datacenter database replication and edge Velocity proxies for ultra-low latency player ingress.
- **GitOps Orchestration:** Full declarative GitOps syncing where repository commits trigger automated staging deployments.
