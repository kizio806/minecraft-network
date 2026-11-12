# ADR 0004: Purpur as the initial backend engine

## Status

Accepted

## Context

The network needs a stable Paper-compatible backend for Lobby and Survival, approximately 20 players and future established/custom plugins. Folia would add scheduling and plugin compatibility constraints without a demonstrated need.

## Decision

Use Purpur for the initial production architecture. Purpur remains Paper API compatible, provides useful performance/configuration controls and is already aligned with the project target. Do not enable Folia.

## Alternatives

- Paper has the most conservative compatibility baseline and remains a valid fallback if a selected plugin documents Paper-only behavior or Purpur regression.
- Folia is deferred because no current workload requires regionized concurrency.

## Consequences

Plugin compatibility is tested against the exact Purpur/Minecraft version before deployment. Purpur-specific settings remain minimal; gameplay plugins are not selected by this architecture stage.
