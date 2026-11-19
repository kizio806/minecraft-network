# ADR 0004 - Purpur as the Backend Game Engine

## Status
Accepted

## Superseded By
None

## Date
2026-11-12

## Decision
Use Purpur 1.21.11 on Java 25 as the standard game engine for both Lobby and Survival backend servers. Folia is explicitly rejected.

## Reason
The network requires a robust, high-performance, Paper-compatible server engine capable of supporting modern Spigot/Paper plugins (LuckPerms, CoreProtect, WorldGuard, DeluxeMenus) for ~20 concurrent players. Purpur provides built-in performance optimizations (alternative tick loops, configurable entity despawn ranges), security hardening (suppression of illegal packet exploits), and 100% Paper API compatibility without breaking existing plugins.

## Alternatives
- **Paper:** Valid fallback, but lacks the granular configuration controls and gameplay tuning options present in Purpur.
- **Folia:** Rejected because its regionized multithreading breaks standard Bukkit/Paper event scheduling, rendering essential plugins (CoreProtect, WorldGuard, DeluxeMenus) incompatible.
- **Vanilla / Fabric:** Rejected due to lack of standard Bukkit/Paper enterprise plugin ecosystem.

## Impact
- All gameplay plugins must maintain compatibility with Purpur/Paper 1.21.11.
- Both `lobby` and `survival` containers run `itzg/minecraft-server:java25` configured for Purpur.
- Consistent configuration model across all backend instances.

## Verification
Verify the backend engine version from the server console:
```bash
./bin/network console survival
# Within console:
version
```
Output must identify the engine as Purpur git-Purpur-1.21.11.
