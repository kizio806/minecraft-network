# Network Topology & Zero-Trust Architecture

Minecraft Network adheres to strict Zero-Trust network segmentation. Direct access from the public Internet to game backend servers or database stores is physically impossible at the network layer.

---

## 1. Docker Networks

The infrastructure isolates services into three isolated Docker bridge networks:

1. **`proxy_net` (Bridge):**
   - Connects the public entrypoint (`proxy`) with internal game services (`auth`, `lobby`, `survival`), `minecraft-exporter`, and `uptime-kuma`.
   - Velocity proxy routes traffic to backend nodes using Modern Forwarding HMAC-SHA256 signatures.

2. **`data_net` (Internal Bridge, `internal: true`):**
   - Completely isolated network with no default gateway or outbound Internet access.
   - Connects `postgres` with backend consumers (`proxy`, `lobby`, `survival`).
   - Redis is completely prohibited across the network.

3. **`monitoring` (Internal Bridge):**
   - Dedicated telemetry network linking `prometheus`, `node-exporter`, `minecraft-exporter`, `grafana`, and `uptime-kuma`.

---

## 2. Port Inventory & Scopes

| Port / Protocol | Network Scope | Target Service | Purpose |
|---|---|---|---|
| `25565/tcp` | Public (`0.0.0.0`) | `proxy` (Velocity) | Player connection gateway |
| `25565/udp` | Public (`0.0.0.0`) | `proxy` / `survival` | Simple Voice Chat (proximity audio) |
| `SSH_PORT/tcp` | Public (Restricted) | Host Linux OS | Administrative SSH access |
| `3000/tcp` | Localhost (`127.0.0.1`) | `grafana` | Monitoring dashboard UI |
| `3001/tcp` | Localhost (`127.0.0.1`) | `uptime-kuma` | Status page and health probes |
| `25565/tcp` | Docker `proxy_net` only | `auth`, `lobby`, `survival` | Backend Minecraft server listeners |
| `5432/tcp` | Docker `data_net` only | `postgres` | Consolidated relational database |
| `9090/tcp` | Docker `monitoring` only | `prometheus` | Prometheus metrics TSDB |
| `9100/tcp` | Docker `monitoring` only | `node-exporter` | Host Linux telemetry |
| `8080/tcp` | Docker `monitoring` only | `minecraft-exporter` | Minecraft server ping metrics |

---

## 3. Forwarding & Authentication Security

- Modern Velocity Forwarding is enforced across all Paper/Purpur backends.
- Every player login packet must contain an authentic HMAC-SHA256 signature generated with `/run/secrets/forwarding_secret`.
- Connecting directly to backend instances (`lobby:25565` or `survival:25565`) without a valid proxy signature immediately drops the connection.
