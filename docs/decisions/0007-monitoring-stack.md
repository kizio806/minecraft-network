# ADR 0007 - Observability and Monitoring Stack

## Status
Accepted

## Superseded By
None

## Date
2026-11-12

## Decision
Deploy a standardized open-source observability stack consisting of Prometheus (metrics collection), Grafana (visualization dashboards), Uptime Kuma (external availability monitoring), Node Exporter (host CPU/RAM/disk metrics), and Minecraft Prometheus Exporter (ping, status, player counts). Resource isolation of this stack is governed by ADR 0014.

## Reason
Operating a multi-container Minecraft network requires comprehensive real-time visibility into server tick rates, JVM heap utilization, container memory saturation, and network latency to preempt OutOfMemory (OOM) crashes and diagnose tick degradation.

## Alternatives
- **Proprietary SaaS monitoring (Datadog, New Relic):** Rejected due to recurring costs, external outbound traffic requirements, and privacy concerns regarding player IP telemetry.
- **In-game command monitoring only (`/tps`, `/spark`):** Rejected because in-game tools do not provide historical trends, external alerting, or container-level memory saturation visibility.

## Impact
- Unified metric collection across physical host, Docker containers, and Minecraft engines.
- Dashboards provisioned automatically via YAML in `infrastructure/monitoring/grafana/`.
- Web dashboards bind exclusively to `127.0.0.1` (`grafana:3000`, `uptime-kuma:3001`).

## Verification
Access local monitoring endpoints when started with `--monitoring`:
```bash
./bin/network start --monitoring
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:3000/api/health
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:3001
```
Both endpoints must return HTTP status 200.
