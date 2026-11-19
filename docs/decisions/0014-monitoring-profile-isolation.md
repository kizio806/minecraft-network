# ADR 0014 - Monitoring Stack Isolation Behind Compose Profiles

## Status
Accepted

## Superseded By
None

## Date
2026-11-15

## Decision
Gate all telemetry and monitoring containers (Prometheus, Grafana, Uptime Kuma, Node Exporter, Minecraft Exporter) behind Docker Compose profile `profiles: [monitoring]`. By default, core server commands (`./bin/network start`) run only essential game and database services. Telemetry is activated optionally via `--monitoring` flag.

This decision amends and extends ADR 0007.

## Reason
Running continuous Prometheus scraping, Grafana dashboards, and multiple exporter containers consumes ~1.6 GB of host RAM and constant CPU cycles. On a 16 GB host where game servers require ~8.8 GB RAM to ensure smooth tick performance (20 TPS), keeping monitoring optional ensures that resource-constrained development and staging environments can run leanly while production environments retain full observability with a single flag.

## Alternatives
- **Always running monitoring containers alongside game backends:** Rejected because it wastes ~1.6 GB RAM during local testing and development where metrics scraping is not needed.
- **External SaaS monitoring (e.g. Datadog):** Rejected due to cost, privacy of player telemetry, and unnecessary internet outbound dependency.

## Impact
- Core startup footprint is limited to ~8.8 GB RAM.
- Production telemetry is started deterministically with `./bin/network start --monitoring`.
- Monitoring web interfaces bind strictly to `127.0.0.1` (`grafana:3000`, `uptime-kuma:3001`).

## Verification
Validate compose profile isolation:
```bash
# Core start without monitoring:
docker compose -f docker-compose.yml config --services | grep -v "prometheus\|grafana"
# Start with monitoring:
./bin/network start --monitoring
./bin/network health
```
Services `grafana` and `prometheus` must only run when `--monitoring` is supplied.
