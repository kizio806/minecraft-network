# Troubleshooting & Monitoring

Operational health checks, monitoring dashboards, and error diagnostics for **Minecraft Network**.

---

## 📊 Observability Stack

- **Prometheus:** Metrics scraping across Velocity, Paper nodes, Redis, and PostgreSQL.
- **Grafana:** Visual dashboards tracking active player counts, tick rates (TPS/MSPT), JVM heap allocation, and network bandwidth.
- **Node Exporter:** Host CPU, memory, and disk I/O monitoring.

---

## 🔍 Common Triage Scenarios

### 1. Players Cannot Connect ("Unable to verify forwarding secret")
- **Cause:** Discrepancy between the secret in `velocity.toml` and backend `paper-global.yml`.
- **Resolution:** Re-run the configuration synchronization script to ensure identical secrets are propagated.

### 2. High Tick Latency (Low TPS)
- **Cause:** Chunk generation or entity overcrowding on a backend node.
- **Resolution:** Inspect Prometheus MSPT graphs to isolate which node is experiencing load. Allocate additional RAM or adjust view distance in `server.properties`.
