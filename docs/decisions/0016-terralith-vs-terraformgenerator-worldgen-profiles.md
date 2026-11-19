# ADR 0016 - Standard Overworld Generation via TerraformGenerator & Deprecation of Terralith

## Status
Accepted

## Superseded By
None

## Date
2026-11-15

## Decision
Adopt **TerraformGenerator (v28.1.0)** as the sole, standard Overworld world generator across the Survival SMP server, and **completely deprecate and remove Terralith**:

1. **Overworld Terrain Generation**:
   - **Engine:** **TerraformGenerator v28.1.0** (Bukkit/Paper Java plugin), active via `worlds.world.generator: TerraformGenerator` in `bukkit.yml`.
   - Injects custom procedural terrain, realistic biomes, custom trees, rivers, and mountain ranges without reliance on experimental datapack noise codecs.
2. **Exploration & Structure Suite**:
   - **Datapacks:** **Structory (v1.3.17)**, **Structory: Towers (v1.0.17)**, and **Dungeons and Taverns (v5.1.0)** running 100% server-side in `world/datapacks/`.
   - **Plugins:** **BetterStructures (v2.7.12)** procedural generation.
3. **Nether & End Overhauls**:
   - **Nether:** **Incendium (v5.4.11)** datapack in `world/datapacks/`.
   - **The End:** **Nullscape (v1.2.17)** datapack in `world/datapacks/`.
4. **Complete Deprecation & Deletion of Terralith**:
   - The Terralith datapack is permanently removed from `configs/survival/datapacks/`, `servers/survival/world/datapacks/`, `components.yml`, and `components.lock.yml`.
   - Eliminates all dimension codec incompatibilities (`Overworld settings missing`) and prevents chunk boundary clashing.

## Reason
1. **Resolution of the World Generator Conflict**:
   - `TerraformGenerator` overrides chunk generation at the Bukkit level via custom `ChunkGenerator` and `BiomeProvider` interfaces.
   - When running alongside a data-driven world generator datapack like `Terralith`, the two systems conflict: TerraformGenerator suppresses Terralith noise registers, and removing Terralith after world creation causes vanilla Minecraft deserialization errors (`java.lang.IllegalStateException: Overworld settings missing`).
   - Committing definitively to **TerraformGenerator** as the single authoritative Overworld engine resolves this conflict permanently.
2. **Procedural Gameplay Features**:
   - TerraformGenerator provides rich procedural landscapes, underground river networks, lush custom foliage, and smooth integration with plugin-based structure generation (`BetterStructures`).
3. **100% Server-Side Execution**:
   - Players connect using standard vanilla Minecraft 1.21 clients without needing client-side mods or resource pack downloads.

## Technical Configuration
In `configs/survival/bukkit.yml` and `servers/survival/bukkit.yml`:
```yaml
settings:
  allow-end: true

worlds:
  world:
    generator: TerraformGenerator
```

## Invariant Compliance
- **Invariant #1 (Port Isolation):** Compliant. No ports exposed.
- **Invariant #2 (Immutable Image Digests):** Compliant.
- **Invariant #3 (Declarative Supply Chain):** Datapacks declared in `components.yml` and plugins declared in `manifest.yml`.
- **Invariant #4 (Secret Protection):** Compliant.
- **Invariant #5 (Database Consolidation):** Compliant.
