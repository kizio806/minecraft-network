# Telemetry and Monitoring Architecture

The monitoring stack provides resource metrics, network availability, and alerting while maintaining minimal resource overhead (Variant 2 Balanced Stack, ~1.4 GB RAM total).

---

## 1. Stack Components

All telemetry services run under the optional Compose profile `monitoring`:

| Service | Image | Role | Port Scope |
|---|---|---|---|
| `prometheus` | `prom/prometheus:v2.55.1` | TSDB metric store (15-day retention) | Internal `monitoring` network |
| `node-exporter` | `prom/node-exporter:v1.8.2` | Linux host CPU, disk, memory metrics | Internal `monitoring` network |
| `minecraft-exporter` | `itzg/mc-monitor:0.17.0` | Queries Proxy, Lobby, and Survival via Minecraft ping | Internal `proxy_net` + `monitoring` |
| `grafana` | `grafana/grafana:11.2.0` | Dashboards with pre-provisioned Prometheus datasource | `127.0.0.1:3000` (Localhost only) |
| `uptime-kuma` | `louislam/uptime-kuma:1.23.16` | External availability probing and Discord alert webhooks | `127.0.0.1:3001` (Localhost only) |

---

## 2. Operational Lifecycle

Start services with the monitoring stack:
```bash
./bin/network start --monitoring
```

Stop services including monitoring:
```bash
./bin/network stop --monitoring
```

Check status and health:
```bash
./bin/network status
./bin/network health
```

---

## 3. Network & Security Isolation

- Telemetry endpoints are strictly bound to localhost (`MONITORING_BIND=127.0.0.1`) or restricted to the internal Docker `monitoring` network. No monitoring ports are exposed to the public Internet.
- Grafana admin password is read from the Docker secret `secrets/grafana_admin_password` (file mode `0600`).
- Heavy logging collectors (Loki, Alloy) and redundant container monitors (cAdvisor) are permanently excluded in favor of standard Docker JSON logging (`json-file` with rotation limits: `10m`, `max-file: 5`).
