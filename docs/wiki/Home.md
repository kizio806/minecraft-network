# Minecraft Network Wiki

Welcome to the **Minecraft Network** engineering and operations wiki. This repository provides an automated, production-grade infrastructure orchestration platform for modern multi-server Minecraft networks running Velocity, Paper, Purpur, and Folia.

---

## 📚 Wiki Navigation

- [Network Architecture](Network-Architecture.md) — High-level architecture, proxy routing, and backend isolation.
- [Deployment & Operations](Deployment-and-Operations.md) — Docker Compose environments, cold starts, and automated backups.
- [Plugin Management](Plugin-Management.md) — Modrinth/Spiget API integration, checksum pinning, and plugin bundles.
- [Security & Hardening](Security-and-Hardening.md) — Firewall rules, Velocity modern forwarding secrets, and container least-privilege.
- [Troubleshooting & Monitoring](Troubleshooting-and-Monitoring.md) — Health checks, Prometheus/Grafana metrics, and triage logs.
- [Developer Guide](Developer-Guide.md) — CLI tooling in Go, environment variables, and automated quality gates.

---

## 🌐 Network Overview

The platform coordinates a multi-tier server topology:
1. **Edge Proxy Tier:** Velocity proxy managing traffic distribution, player authentication, and modern velocity forwarding.
2. **Backend Server Tier:** Dedicated Paper/Purpur game nodes (Lobby, Survival, Minigames) isolated on an internal bridge network.
3. **Database & Cache Tier:** Centralized PostgreSQL and Redis for cross-server player synchronization and permissions.
4. **Operations & Automation Tier:** Custom Go CLI orchestrator for plugin downloads, configuration templating, and health verification.
