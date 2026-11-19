# ADR 0002 - Zero-Trust Network Segmentation and Port Isolation

## Status
Accepted

## Superseded By
None

## Date
2026-11-12

## Decision
Segment container traffic across three isolated Docker bridge networks:
1. `proxy_net`: Standard bridge network connecting the Velocity reverse proxy to internal backend Minecraft servers (`auth`, `lobby`, `survival`). Only `proxy` binds publicly (`0.0.0.0:25565:25565/tcp`).
2. `data_net`: Strict internal network (`internal: true`) connecting the database (`postgres`) exclusively to authenticated backend servers. Databases have zero host port mappings.
3. `monitoring`: Dedicated network for telemetry scrapers (Prometheus, Grafana, exporters). Telemetry web UIs bind strictly to `127.0.0.1`.

All backend Minecraft servers (`auth`, `lobby`, `survival`) must never publish host ports.

## Reason
Directly exposing backend Minecraft server ports allows malicious clients to bypass proxy authentication, forge player UUIDs, and gain unauthorized operator access. Isolating the database within a route-free internal network prevents network scanning, brute force, and external exploitation.

## Alternatives
- **Single flat bridge network:** Rejected because all services would have uninhibited layer-3 reachability to the database and monitoring agents, violating the principle of least privilege.
- **Publishing backend ports to host loopback (`127.0.0.1`):** Rejected because internal container-to-container communication occurs over Docker bridge DNS, making host interface exposure unnecessary and error-prone.

## Impact
- Only port `25565/tcp` is open on the public network interface.
- Backend Minecraft instances and PostgreSQL are completely unreachable from outside the Docker host.
- Internal network traffic between proxy and backends is contained within private Docker bridges.

## Verification
Inspect container port bindings and network isolation:
```bash
docker compose ps --format "table {{.Service}}\t{{.Ports}}"
```
Verify that only `proxy` publishes to `0.0.0.0:25565`, and backend services have no host port mappings.
