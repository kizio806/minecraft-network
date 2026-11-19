# Infrastructure Context

## Purpose
Define container resource sizing, physical host memory budgets, Docker Compose service specifications, volume persistence models, and the opt-in telemetry stack.

## Scope
Applies to all Docker Compose services, container resource limits, JVM heap ceilings, named volumes, and telemetry containers under `--profile monitoring`.

## Rules / Information

### 1. Memory Sizing & JVM Heap Ceiling Formula

To prevent Docker OOM kills caused by native JVM memory (Metaspace, native memory, thread stacks, garbage collection buffers), the container hard limit must maintain at least a 30% overhead above the maximum JVM heap:

$$\text{Container Limit} \ge 1.3 \times \text{JVM Max Heap (Xmx)}$$

### 2. Service Resource Allocation Table

| Service | Max JVM Heap (`-Xmx`) | Container Memory Limit | CPU Limit | Network Attachment | Volume / Storage Mount |
| :--- | :---: | :---: | :---: | :--- | :--- |
| **`survival`** | 4096M (4G) | 5632M (5.5 GB) | 3.0 cores | `proxy_net`, `data_net` | Bind: `./servers/survival:/data` |
| **`lobby`** | 1024M (1G) | 1536M (1.5 GB) | 1.5 cores | `proxy_net`, `data_net` | Bind: `./servers/lobby:/data` |
| **`proxy`** | 512M | 768 MB | 1.0 core | `proxy_net` | Bind: `./servers/proxy:/server` |
| **`auth`** | 256M | 384 MB | 0.5 cores | `proxy_net`, `data_net` | Bind: `./servers/auth:/data` |
| **`postgres`** | N/A | 512 MB | 1.0 core | `data_net` | Named: `postgres-data:/var/lib/postgresql/data` |
| **Core Budget Total** | **~5.8 GB** | **8.8 GB** | **7.0 cores** | — | **Max host RAM: 16 GB** |

### 3. Optional Telemetry Services (`--profile monitoring`)

When enabled with `./bin/network start --monitoring`, the monitoring stack consumes ~1.6 GB additional RAM:

| Service | Image & Version | Port / Binding | Memory Limit | Storage Mount |
| :--- | :--- | :--- | :---: | :--- |
| **`prometheus`** | `prom/prometheus:v2.54.1` | Internal (9090) | 512 MB | Named: `prometheus-data` |
| **`grafana`** | `grafana/grafana:11.2.0` | `127.0.0.1:3000:3000` | 512 MB | Named: `grafana-data` |
| **`uptime-kuma`** | `louislam/uptime-kuma:1.23.16` | `127.0.0.1:3001:3001` | 384 MB | Named: `uptime-data` |
| **`node-exporter`** | `prom/node-exporter:v1.8.2` | Internal (9100) | 128 MB | Host `/proc`, `/sys` (ro) |
| **`minecraft-exporter`**| `dirien/minecraft-prometheus-exporter:v0.17.0` | Internal (9150) | 128 MB | Scrapes Minecraft RCON |
| **Total with Monitoring**| — | — | **~10.5 GB** | **Well within 12 GB budget** |

### 4. Non-Root Execution & Security Options

All game servers execute with non-root privileges:
```yaml
user: "${MC_UID:-1000}:${MC_GID:-1000}"
security_opt:
  - no-new-privileges:true
```

## Verification
Validate Docker Compose syntax and resource declarations:
```bash
docker compose -f docker-compose.yml config --quiet
docker compose -f docker-compose.yml --profile monitoring config --quiet
```
Both commands must exit with code 0.
