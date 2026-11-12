# ADR 0007: Monitoring stack

## Status

Accepted

## Decision

Use Prometheus, Grafana, Loki, Grafana Alloy, node-exporter, cAdvisor, mc-monitor and Uptime Kuma. mc-monitor provides actual Minecraft status/version/player-count metrics; node-exporter and cAdvisor cover host/container resources; Alloy ships Docker logs to Loki.

## Deferred metrics

TPS, MSPT and JVM internals require a server/plugin exporter that is not part of this baseline. They are deliberately not represented as available metrics until a compatible source is selected.
