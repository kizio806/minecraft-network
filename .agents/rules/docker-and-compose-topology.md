# Rule: Docker & Compose Topology

## Purpose
Enforce container security, deterministic image builds, memory safety margins, and zero-trust network isolation across the entire Docker Compose infrastructure.

## Scope
Applies to `docker-compose.yml`, `.env.example`, network configurations, container resource limits, and service definitions.

## MUST
- **MUST** pin every container image to an immutable `@sha256:` digest (e.g. `itzg/minecraft-server:java25@sha256:...`).
- **MUST** isolate internal databases on an internal network (`data_net` with `internal: true`).
- **MUST** enforce the memory sizing formula: $\text{Container Limit} \ge 1.3 \times \text{JVM Max Heap (Xmx)}$ to accommodate native JVM memory overhead.
- **MUST** maintain total combined container memory limits within the 12 GB physical host RAM ceiling.
- **MUST** run game containers as non-root using `user: "${MC_UID:-1000}:${MC_GID:-1000}"` with `security_opt: [no-new-privileges:true]`.
- **MUST** gate telemetry containers (Prometheus, Grafana, exporters, Uptime Kuma) behind `profiles: [monitoring]`.

## MUST NOT
- **MUST NOT** expose backend ports (`auth`, `lobby`, `survival`, `postgres`) to `0.0.0.0` or external host interfaces. The only allowed public binding is `proxy` (`0.0.0.0:25565:25565/tcp`).
- **MUST NOT** use mutable or unpinned Docker tags (`latest`, `alpine`, `java25`) in `docker-compose.yml` or CI workflows.
- **MUST NOT** bind monitoring web dashboards (`grafana`, `uptime-kuma`) to public interfaces; they must bind strictly to `127.0.0.1`.
- **MUST NOT** run containers as `root` or without `no-new-privileges:true`.

## Verification
Validate Docker Compose syntax and pinned images:
```bash
docker compose -f docker-compose.yml config --quiet
docker compose -f docker-compose.yml --profile monitoring config --quiet
```
Both commands must exit with code 0.
