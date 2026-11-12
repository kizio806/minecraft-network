# ADR 0008: Local daily backup with external production copy

## Status

Accepted

## Decision

The native Go CLI creates a local backup with checksum and archive validation. It stops mutable Minecraft services for world consistency, dumps optional PostgreSQL when enabled, and includes runtime configuration and server data.

## Production requirement

The local destination is not sufficient for production. A systemd timer or external backup job must replicate encrypted archives to a separate host/object-storage account, with restricted write credentials and periodic isolated restore tests. The external provider is intentionally deferred until deployment constraints are known.
