# Architecture Context

## Purpose
Define the high-level software components, server execution boundaries, world generation layers, and player traffic flow across the Minecraft Network.

## Scope
Applies to the Velocity reverse proxy, NanoLimbo authentication limbo, Purpur lobby server, Purpur survival server, PostgreSQL relational persistence, and server-side datapacks.

## Rules / Information

### 1. Component Execution Boundaries

| Service | Engine & Version | Role & Boundary | Java Runtime | Base Image |
| :--- | :--- | :--- | :---: | :--- |
| **`proxy`** | Velocity 4.3.1 | Reverse proxy, modern player forwarding (HMAC-SHA256), rate-limiting, tablist, MOTD, SkinsRestorer, VeloAuth interceptor. Sole public entrypoint (`0.0.0.0:25565`). | Java 25 | `itzg/mc-proxy:java25` |
| **`auth`** | NanoLimbo 1.12.0 | Ultra-lightweight void server for unauthenticated non-premium registration/login. Zero terrain generation. | Java 21 | `eclipse-temurin:21-jre` |
| **`lobby`** | Purpur 1.21.11 | Read-only spawn server with FancyNpcs navigation, DeluxeMenus GUI, and WorldGuard global protection. Flat the_void world tracked in Git LFS. | Java 25 | `itzg/minecraft-server:java25` |
| **`survival`** | Purpur 1.21.11 | Core Vanilla+ Enhanced SMP gameplay server with 3-dimension overhaul (TerraformGenerator procedural Overworld, Incendium, Nullscape), exploration structures (Structory, Structory Towers, Dungeons and Taverns, BetterStructures), Auxilor RPG ecosystem (`eco`, `libreforge`, `EcoSkills`, `EcoEnchants`, `EcoItems`), QoL suite (`GSit`, `BetterSleeping`, `InteractiveChat`), PvP controls, InvSee++ & InventoryRollbackPlus admin tools, Anti-AutoTotem and GrimAC protection, and CoreProtect logging. | Java 25 | `itzg/minecraft-server:java25` |
| **`postgres`** | PostgreSQL 16.4 | Relational database providing schema-isolated storage for `luckperms` and `veloauth`. Isolated in internal `data_net`. | N/A | `postgres:16.4-alpine` |

### 2. Player Traffic & Lifecycle Flow

```text
Player Connection (:25565)
          │
          ▼
   [proxy] (Velocity)
          │
          ├── Mojang Premium Player? ──────► Forward to [lobby]
          │                                        │
          └── Non-Premium Player?                  │
                   │                               │
                   ▼                               ▼
            Route to [auth]                 Server Selector NPC
            (NanoLimbo login)               (FancyNpcs / DeluxeMenus)
                   │                               │
                   ▼                               ▼
            /login or /register              Route to [survival]
            (VeloAuth validation)           (Enhanced SMP Gameplay)
                   │                               │
                   └──────────────► Forward to [lobby]
```

### 3. State & Persistence Boundaries

- **Relational Schemas:**
  - `luckperms`: Stores network-wide player groups, permissions, prefixes, and inheritance rules.
  - `veloauth`: Stores hashed passwords, session tokens, and registration records for non-premium players.
- **World & Block Logging:**
  - `survival`: Dynamic region files (`world`, `world_nether`, `world_the_end`) and SQLite-based CoreProtect database are persistent runtime state.
  - `lobby`: Flat void world template protected declaratively via WorldGuard `__global__` flags in `configs/lobby/plugins/worldguard/`.

### 4. World Generation & Dimension Overhauls (100% Server-Side)

The Survival server features overhauled generation across all three Minecraft dimensions (ADR 0016):
- **Overworld Generation (TerraformGenerator Standard — ADR 0016)**:
  - Procedural landscape generation driven by **TerraformGenerator** (v28.1.0) declared via `worlds.world.generator: TerraformGenerator` in `bukkit.yml`.
  - Injects realistic biomes, custom trees, canyons, and mountain ranges via its Bukkit `ChunkGenerator` engine.
  - Terralith has been fully deprecated and removed to prevent custom multi-noise registry collisions and dimension deserialization errors.
  - **Structure Suite**: Exploration datapacks **Structory** (v1.3.17), **Structory: Towers** (v1.0.17), **Dungeons and Taverns** (v5.1.0), and procedural plugin structures from **BetterStructures** (v2.7.12).
- **Nether**: **Incendium** (v5.4.11) datapack introducing 8 volcanic/ash biomes, Sanctum castles, Piglin factories, and redesigned fortresses.
- **The End**: **Nullscape** (v1.2.17) datapack raising island generation up to Y=384 with Crystalline Peaks, void chasms, and layered terrain.
- **Strict Server-Side Boundary**: All datapacks reside in `servers/survival/world/datapacks/`. Players connect using standard vanilla Minecraft 1.21 clients without needing mods or resource pack downloads.

### 5. Auxilor Eco Ecosystem & Plugin Dependency Graph

- **Core Runtime & Engine**:
  - `eco` (v2026.46): Base API and item framework.
  - `libreforge` (v2026.46): Unified mechanics and effect engine, required by all Eco content plugins. Shaded with `libreforge-loader` and bytecode relocated for Java 25 runtime stability.
- **Content Plugins**:
  - `EcoSkills` (v2026.46): 6-path skill progression (Mining, Farming, Combat, Woodcutting, Fishing, Agility) with Polish GUI.
  - `EcoEnchants` (v2026.46): Balanced custom enchantments integrated with vanilla enchantment tables, anvils, and dungeon loot.
  - `EcoItems` (v2026.46): Custom equipment and items with Paper `block-updates` texture protection.
- **Social & Quality of Life Plugins**:
  - `GSit` (v3.8.0): Player poses (`/sit`, `/lay`, `/crawl`, `/spin`) and seat sneaking.
  - `BetterSleeping` (v4.0.2): 20% sleeping threshold for skipping night with morning regeneration buffs.
  - `InteractiveChat` (v2026.1.3.0): Chat tags (`[item]`, `[inv]`, `[ender]`) with interactive hover tooltips and GUI previews.
  - `PvPManager` (v4.0.30) & `UltimateTeams` (v5.0.3): Voluntary PvP toggling and team friendly-fire immunity.
  - `Simple Voice Chat` (v2.6.27): UDP-based proximity voice chat.
- **Exploration, Administration & Anticheat Plugins**:
  - `InvSee++` (v0.31.22): Polish-localized administrative inspection and real-time inventory/enderchest editing.
  - `InventoryRollbackPlus` (v1.8.5): Automated player inventory backups and administrative rollbacks with Polish messages.
  - `Anti-AutoTotem` (v1.0): PacketEvents-based cheat detection blocking automated totem equipping.

## Verification
Inspect container status, routing health, and plugin validation:
```bash
./bin/network doctor
./bin/network health
./bin/network plugins validate
```
All commands must report `OK` and exit code 0.
