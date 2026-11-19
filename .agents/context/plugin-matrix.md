# Context: Plugin, Component & Datapack Matrix

## Purpose
Catalog all active plugins, datapacks, components, versions, platforms, licensing, and strictly prohibited components across the network.

## Scope
Applies to plugin declarations in `configs/plugins/manifest.yml`, lockfiles in `configs/plugins/lock.yml`, components in `configs/plugins/components.yml`, and runtime plugin/datapack installations.

## Rules / Information

### 1. Active Plugins Inventory

| Plugin | Version | Platform | Target Servers | Source | License | Purpose |
| :--- | :--- | :---: | :--- | :--- | :--- | :--- |
| **VeloAuth** | `1.6.2` | Velocity | `[proxy]` | Modrinth | MIT | Hybrid authentication (Mojang API verification + NanoLimbo redirect). |
| **LuckPerms** | `5.5.71` | Velocity / Paper | `[proxy, lobby, survival]` | Modrinth | MIT | Network-wide permissions management backed by PostgreSQL schemas. |
| **MiniMOTD** | `2.2.2` | Velocity | `[proxy]` | Modrinth | MIT | Multi-line dynamic server list MOTD and favicon display. |
| **TAB** | `6.2.0` | Velocity | `[proxy]` | Modrinth | Apache-2.0 | Tablist layout, header, footer, and player sorting. |
| **TAB Bridge** | `6.2.3` | Paper | `[survival]` | Modrinth | Apache-2.0 | Bukkit bridge for placeholder synchronization with Velocity TAB. |
| **SkinsRestorer**| `15.12.6` | Velocity / Paper | `[proxy, lobby, survival]` | Modrinth | GPL-3.0 | Restores player skins for Non-Premium accounts. |
| **PlaceholderAPI**| `2.11.7`| Paper | `[lobby, survival]` | Modrinth | GPL-3.0 | Variable placeholders for DeluxeMenus, TAB, and gameplay plugins. |
| **FancyNpcs** | `2.12.1` | Paper | `[lobby]` | Modrinth | MIT | Server selection NPC (`survival_guide`) with `send_to_server` action. |
| **DeluxeMenus** | `1.14.1` | Paper | `[lobby]` | Modrinth | GPL-3.0 | Interactive GUI menus for server navigation and information. |
| **ItemJoin** | `6.1.9` | Paper | `[lobby]` | Modrinth | GPL-3.0 | Custom join items, navigation compass, and locked anti-tamper inventory filler. |
| **WorldEdit** | `7.3.18` | Paper | `[lobby, survival]` | Modrinth | GPL-3.0 | World manipulation and schematic management for administration. |
| **WorldGuard** | `7.0.16` | Paper | `[lobby, survival]` | Modrinth | GPL-3.0 | Region protection (PvP deny, block break/place deny, hunger deny). |
| **WG Extra Flags**| `4.2.4` | Paper | `[lobby, survival]` | GitHub | MIT | Additional WorldGuard flags (fly, gamemode, command execution). |
| **EssentialsX** | `2.22.0` | Paper | `[lobby, survival]` | Modrinth | GPL-3.0 | Core utilities, gamemode control, and teleport commands (`/spawn`, `/home`). |
| **EssentialsX Spawn**| `2.22.0`| Paper | `[lobby, survival]` | GitHub | GPL-3.0 | Enforces spawn point positioning upon connection and respawn. |
| **EssentialsX Chat** | `2.22.0`| Paper | `[lobby, survival]` | GitHub | GPL-3.0 | Formats chat messages with LuckPerms group prefixes and channels. |
| **Vault** | `1.7.3` | Paper | `[lobby, survival]` | GitHub | LGPL-3.0 | Standard permission and economy bridge API for Bukkit plugins. |
| **DecentHolograms**| `2.10.2`| Paper | `[lobby]` | Modrinth | GPL-3.0 | Informational floating text holograms at lobby spawn. |
| **CoreProtect** | `24.1` | Paper | `[survival]` | Modrinth | CoreProtect | Block transaction logging, griefing rollback, container inspection. |
| **PacketEvents** | `2.14.0` | Paper | `[survival]` | Modrinth | LGPL-3.0 | Low-level packet manipulation library required by GrimAC and InteractiveChat. |
| **NBT-API** | `2.16.1` | Paper | `[survival]` | Modrinth | MIT | Direct NBT manipulation API for custom item and block data. |
| **GrimAC** | `2.3.75` | Paper | `[survival]` | Modrinth | GPL-3.0 | Fully asynchronous physics simulation anticheat against movement/combat cheats. |
| **Raycasted Anti-ESP** | `0.7.2` | Paper | `[survival]` | Modrinth | MIT | Raycasting entity hiding engine preventing ChestESP, Freecam, and PlayerESP. |
| **AnarchyExploitFixes** | `2.7.12` | Paper | `[survival]` | Modrinth | GPL-3.0 | Prevents dupe exploits, packet spam, lag machines, and coordinate leaks. |
| **Chunky** | `1.5.66` | Paper | `[survival]` | Modrinth | GPL-3.0 | Asynchronous chunk pre-generation to eliminate worldgen stuttering. |
| **TerraformGenerator** | `28.1.0` | Paper | `[survival]` | Modrinth | MIT | Procedural terrain generation engine for realistic Overworld biomes. |
| **BetterStructures** | `2.7.12` | Paper | `[survival]` | Modrinth | Custom | Procedural generation of custom dungeons, towers, ruins, and underground halls. |
| **GSit** | `3.8.0` | Paper | `[lobby, survival]` | Modrinth | GPL-3.0 | Player poses (`/sit`, `/lay`, `/crawl`, `/spin`), seat sneaking, and block sitting. |
| **BetterSleeping** | `4.0.2` | Paper | `[survival]` | GitHub | GPL-3.0 | Percentage sleeping requirement (20%), morning buffs, and Polish chat alerts. |
| **InteractiveChat**| `2026.1.3.0` | Paper | `[survival]` | Modrinth | GPL-3.0 | Interactive chat tags (`[item]`, `[inv]`, `[ender]`) with GUI previews. |
| **InteractiveChat-PacketEvents** | `1.3.2` | Paper | `[survival]` | GitHub | GPL-3.0 | PacketEvents protocol bridge enabling InteractiveChat packet interception without ProtocolLib. |
| **eco** | `2026.46` | Paper | `[survival]` | Modrinth | Custom | Core runtime library and item/recipe framework for Auxilor plugins. |
| **libreforge** | `2026.46` | Paper | `[survival]` | Auxilor | Proprietary | Unified effect, condition, and mechanic engine powering Eco plugins. |
| **EcoSkills** | `2026.46` | Paper | `[survival]` | Auxilor | Proprietary | RPG skill progression (Mining, Farming, Combat, Woodcutting, Fishing, Agility). |
| **EcoEnchants** | `2026.46` | Paper | `[survival]` | Auxilor | Proprietary | Balanced custom enchantments integrated into enchanting tables, anvils, and loot. |
| **EcoItems** | `2026.46` | Paper | `[survival]` | Auxilor | Proprietary | Custom equipment, runes, and special items with Paper block-update safeguards. |
| **PvPManager** | `4.0.30` | Paper | `[survival]` | Modrinth | GPL-3.0 | Player PvP state control (`/pvp on|off`), combat tagging, and newbie protection. |
| **UltimateTeams** | `5.0.3` | Paper | `[survival]` | Modrinth | GPL-3.0 | Player clans and teams with friendly-fire immunity and team chat. |
| **Simple Voice Chat (Velocity)** | `2.6.26` | Velocity | `[proxy]` | Modrinth | GPL-3.0 | Proxy-side routing component for proximity voice chat. |
| **Simple Voice Chat (Bukkit)** | `2.6.27` | Paper | `[survival]` | Modrinth | GPL-3.0 | Backend proximity voice chat server running on UDP 24454. |
| **ToggleDimensions** | `1.0.0` | Paper | `[survival]` | Modrinth | MIT | Dynamic runtime enabling/disabling of Nether and End portals without restart. |
| **InvSee++** | `0.31.22` | Paper | `[survival]` | Modrinth | GPL-3.0 | Administrative inspection and modification of online/offline player inventories. |
| **InventoryRollbackPlus** | `1.8.5` | Paper | `[survival]` | Modrinth | GPL-3.0 | Automated player inventory backups and administrative rollbacks with Polish messages. |
| **Anti-AutoTotem** | `1.0` | Paper | `[survival]` | Modrinth | GPL-3.0 | PacketEvents-based cheat detection blocking automated totem equipping in PvP. |
| **LibertyBans** | `1.1.4` | Paper | `[survival]` | Modrinth | GPL-3.0 | Comprehensive punishment management (bans, mutes, warnings) with cross-server sync. |

---

### 2. Datapacks & World Overhauls (100% Server-Side)

| Datapack | Version | Target Server | Source | License | Purpose & Details |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Structory** | `1.3.17` | `survival` (`world/datapacks/`) | Modrinth | MIT | Atmospheric settlements, ruins, camps, and exploration waypoints across Overworld biomes. Zero client downloads. |
| **Structory: Towers** | `1.0.17` | `survival` (`world/datapacks/`) | Modrinth | MIT | Procedural watchtowers, observation outposts, and wilderness beacons. Zero client downloads. |
| **Dungeons and Taverns** | `5.1.0` | `survival` (`world/datapacks/`) | Modrinth | Custom | Sprawling underground labyrinths, taverns, keeps, castles, and dungeons with rich loot tables. Zero client downloads. |
| **Incendium** | `5.4.11` | `survival` (`world/datapacks/`) | Modrinth | Custom | Complete **Nether** overhaul: 8 new biomes (Toxic Heap, Ash Barrens, Volcanic Delta, Infernal Dunes, Quartz Flats), Sanctum fortresses, Piglin factories. Zero client downloads. |
| **Nullscape** | `1.2.17` | `survival` (`world/datapacks/`) | Modrinth | Custom | Complete **The End** overhaul: monumental islands spanning up to Y=384, Crystalline Peaks, void chasms, layered terrain. Zero client downloads. |

---

### 3. Standalone Components

| Component | Version | Target Server | Source | License | Purpose |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **NanoLimbo** | `1.12.0` | `auth` | GitHub | Apache-2.0 | Ultra-lightweight End dimension holding server for Non-Premium auth. |

---

### 4. Prohibited & Replaced Components

- ❌ **`Redis` / `RedisBungee`**: Prohibited. Velocity handles modern player forwarding and tablist natively. PostgreSQL provides unified relational state.
- ❌ **`CommandPanels`**: Prohibited. Redundant; DeluxeMenus is the sole approved GUI menu framework.
- ❌ **`Multiverse-Core`**: Prohibited. Lobby operates a single flat void world; Survival handles Nether and The End natively via Purpur.
- ❌ **`VelocityScoreboardAPI`**: Prohibited. Unneeded dependency; TAB manages scoreboard display natively.
- ❌ **`FarmLimiter`**: Prohibited. Dropped in favor of Purpur entity tick limiting and native mob activation range optimizations.

## Verification
Validate plugin manifest and lockfile integrity:
```bash
./bin/network plugins validate
./bin/network plugins plan
```
Must report exit code 0 and confirm all active plugins match their cryptographic hashes.
