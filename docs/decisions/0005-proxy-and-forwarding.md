# ADR 0005: Velocity modern forwarding

## Status

Accepted

## Decision

Velocity is the only public Minecraft endpoint. Modern forwarding is used on the private proxy network. VeloAuth owns the Premium/Non-Premium decision; backend Paper/Purpur velocity support is enabled with the same forwarding secret. Backend ports are never published.

## Security implications

The forwarding secret is a Docker Secret, separate from VeloAuth and database secrets. A backend must not be reachable from the Internet or from an unapproved container. Legacy Bungee forwarding is not used.
