# ADR 0009 - Docker Network Boundaries

## Status
Deprecated

## Superseded By
ADR 0002 - Zero-Trust Network Segmentation and Port Isolation

## Date
2026-11-12

## Decision
Originally created as a standalone summary of Docker container network bindings (`proxy_net`, `data_net`, `monitoring`), specifying that no backend, database, or monitoring ports are public.

## Reason
This document duplicated the architectural scope and principles already established in ADR 0002.

## Alternatives
- Retaining two separate ADRs for network segmentation and network boundaries was rejected to eliminate documentation drift and ambiguity.

## Impact
- All network isolation rules, bridge definitions, and port isolation constraints are formally unified under ADR 0002.

## Verification
Refer to the verification procedure defined in [ADR 0002](0002-network-segmentation.md):
```bash
docker compose ps --format "table {{.Service}}\t{{.Ports}}"
```
