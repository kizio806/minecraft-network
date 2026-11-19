# ADR 0015 - Server-Side Datapacks for Dimension Overhauls (Incendium & Nullscape)

## Status
Accepted

## Superseded By
None

## Date
2026-11-15

## Decision
Enhance Nether and The End dimensions on the Survival backend exclusively using server-side datapacks (**Incendium v5.4.11** for the Nether and **Nullscape v1.2.17** for The End), pinned by cryptographic SHA-256 digests in `configs/plugins/components.lock.yml` and synchronized during `./bin/network install`.

Datapacks must run 100% on the server side (`servers/survival/world/datapacks/`). No client-side mods (Fabric/Forge/NeoForge), loaders, or required resource packs are permitted for dimension generation.

## Reason
1. **Zero Client Barrier**: Players connect to the Survival SMP using standard vanilla Minecraft 1.21 clients. Requiring client-side mod installation creates high friction for a casual community server.
2. **Standardized Mojang Worldgen Format**: Minecraft 1.21.11 uses modern worldgen datapacks (Pack Format 94.1) with noise settings, multi-noise biomes, and structure templates processed entirely by the Purpur/Paper server runtime.
3. **High Performance & Stability**: Incendium and Nullscape use native vanilla world generation pipelines instead of injecting custom Java entity ticking or foreign block entities, keeping MSPT stable and memory usage within the 4 GB JVM budget.
4. **Supply Chain Integrity**: Datapack archives are versioned in `configs/survival/datapacks/` and tracked in `components.yml` / `components.lock.yml` alongside standalone components.

## Alternatives Considered
- **Modded Forge / NeoForge Server**: Rejected. Destroys vanilla client compatibility, consumes massive RAM (>8 GB), and breaks Velocity modern player forwarding.
- **Third-Party Dimension Generator Plugins**: Evaluated alternative plugins for Nether and End generation; rejected due to unmaintained codebases for 1.21.11, memory leaks, and high crash risk during chunk generation.
- **Vanilla Default Generation**: Rejected. Standard Nether and End lack deep exploration, biomes, and structures desired for the long-term SMP experience.

## Technical Details
- **Nether Overhaul**: Incendium adds 8 biomes, Sanctum castles, and Piglin factories within native Nether world limits.
- **End Overhaul**: Nullscape raises End island generation up to Y=384, introducing Crystalline Peaks and void chasms.
- **Deployment**: `internal/networkcli/operations.go` copies `configs/survival/datapacks/*.zip` to `servers/survival/world/datapacks/` during `./bin/network install`.

## Invariant Compliance
- **Invariant #1 (Port Isolation):** Compliant. No ports exposed.
- **Invariant #2 (Immutable Image Digests):** Compliant. Standard Purpur image used.
- **Invariant #3 (Declarative Supply Chain):** Compliant. Declared in `components.yml` and verified via SHA-256 in `components.lock.yml`.
- **Invariant #4 (Secret Protection):** Compliant. Zero secrets involved.
- **Invariant #5 (Database Consolidation):** Compliant. Uses existing architecture.
