# Deployment & Operations Guide

This guide covers deployment workflows, environment configuration, and automated operations for **Minecraft Network**.

---

## 🚀 Quick Start Deployment

1. **Prerequisites:**
   - Docker Engine 24.0+ & Docker Compose v2.
   - Host kernel with standard networking and IP forwarding enabled.
2. **Environment Configuration:**
   - Copy the environment template:
     ```bash
     cp .env.example .env
     ```
   - Configure secrets: `VELOCITY_FORWARDING_SECRET`, `POSTGRES_PASSWORD`, `GRAFANA_ADMIN_PASSWORD`.
3. **Provisioning & Startup:**
   - Start the core network stack:
     ```bash
     docker compose up -d
     ```
   - Verify health status:
     ```bash
     docker compose ps
     ```

---

## 💾 Backup & Disaster Recovery

- Automated snapshots run daily via cron to secure database schemas and player state.
- World state directories mount local host volumes under `data/` for rapid disaster recovery.
- Restoration procedures are detailed in [`docs/backup.md`](../backup.md).
