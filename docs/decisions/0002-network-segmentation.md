# ADR 0002: Network segmentation

## Status

Accepted

## Decision

`proxy_net` carries proxy-to-backend traffic, `data_net` is internal and carries backend-to-PostgreSQL/Redis traffic, and `monitoring` is internal with host-bound UI ports. Databases have no host ports.

## Consequences

The backend services are attached to two networks by design. A future plugin that needs a database must use an explicitly approved service/network path; it does not receive Internet exposure.
