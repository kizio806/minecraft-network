# Plugin management

Plugin management is declarative and deterministic. The reusable Go core is available both as `plugin-manager` and through `network plugins`.

## Files

- `configs/plugins/manifest.yml` — desired plugin state and compatibility policy.
- `configs/plugins/lock.yml` — exact provider version, immutable version ID, URL, filename, size and trusted digest.
- `configs/plugins/components.yml` — standalone runtime components such as NanoLimbo; these are not plugins.
- `configs/plugins/components.lock.yml` — immutable artifact lock for standalone components.
- `configs/plugins/manifest.schema.json` — JSON Schema for external/editor validation.

The proxy manifest contains VeloAuth `1.6.2`, LuckPerms `v5.5.71-velocity`, MiniMOTD `2.2.2`, TAB `6.2.0`, VelocityScoreboardAPI `2.1.1` and SkinsRestorer `15.12.6`. The Lobby backend manifest contains FancyNpcs `2.12.1`, LuckPerms `v5.5.71-bukkit`, PlaceholderAPI `2.11.7`, WorldEdit `7.3.18`, WorldGuard `7.0.16`, WorldGuard Extra Flags `4.2.4`, EssentialsX `2.22.0`, EssentialsX Spawn `2.22.0`, DeluxeMenus `1.14.1-Release`, ItemJoin `6.1.9`, DecentHolograms `2.10.2`, GSit `3.8.0` and SkinsRestorer `15.12.6`. The Survival backend manifest contains the complete Vanilla+ Enhanced SMP suite including CoreProtect, GrimAC, Raycasted Anti-ESP, AnarchyExploitFixes, Chunky, TerraformGenerator, BetterStructures, GSit, BetterSleeping, InteractiveChat, the Auxilor Eco ecosystem (`eco`, `libreforge`, `EcoSkills`, `EcoEnchants`, `EcoItems`), PvPManager, UltimateTeams, and Simple Voice Chat. Dimension overhauls for the Nether and The End are managed declaratively as 100% server-side datapacks (`Incendium` and `Nullscape`). NanoLimbo is classified separately as a standalone runtime component.

### Velocity Proxy Plugins

| Plugin | Git source | Runtime path | Managed files | Generated/runtime data |
|---|---|---|---|---|
| VeloAuth | `configs/proxy/plugins/veloauth` | `plugins/veloauth/config.yml` | `config.yml` | `data/`, language/runtime state |
| LuckPerms | `configs/proxy/plugins/luckperms` | `plugins/luckperms/config.yml` | `config.yml` | PostgreSQL state; password marker rendered from `secrets/luckperms_password` before startup |
| MiniMOTD | `configs/proxy/plugins/minimotd` | `plugins/minimotd/main.conf`, `plugin_settings.conf` | both HOCON files | icons and runtime cache only |
| TAB | `configs/proxy/plugins/tab` (`config-target-dir: TAB`) | `plugins/TAB/config/*.yml` | `config/*.yml` | runtime cache and metrics only |
| VelocityScoreboardAPI | none | plugin API only | none | none |
| SkinsRestorer | `configs/proxy/plugins/skinsrestorer` (`config-target-dir: SkinsRestorer`) | `plugins/SkinsRestorer/config.yml` | `config.yml` | skin cache/database and generated metadata |

### Lobby Backend Plugins (Purpur)

| Plugin | Git source | Runtime path | Managed files | Generated/runtime data |
|---|---|---|---|---|
| LuckPerms | `configs/lobby/plugins/luckperms` (`config-target-dir: LuckPerms`) | `plugins/LuckPerms/config.yml` | `config.yml` | PostgreSQL shared state; password rendered from `secrets/luckperms_password` |
| PlaceholderAPI | `configs/lobby/plugins/placeholderapi` (`config-target-dir: PlaceholderAPI`) | `plugins/PlaceholderAPI/config.yml` | `config.yml` | `expansions/` cache |
| FancyNpcs | none | `plugins/FancyNpcs-2.12.1.jar` | none | NPC definitions and runtime state configured in-game |
| WorldEdit | `configs/lobby/plugins/worldedit` (`config-target-dir: WorldEdit`) | `plugins/WorldEdit/config.yml` | `config.yml` | schematics, session history |
| WorldGuard | `configs/lobby/plugins/worldguard` (`config-target-dir: WorldGuard`) | `plugins/WorldGuard/config.yml` | `config.yml` | `worlds/<world>/regions.yml` |
| WorldGuard Extra Flags | `configs/lobby/plugins/worldguardextraflags` (`config-target-dir: WorldGuardExtraFlags`) | `plugins/WorldGuardExtraFlags/config.yml` | `config.yml` | custom flags state |
| EssentialsX | `configs/lobby/plugins/essentialsx` (`config-target-dir: Essentials`) | `plugins/Essentials/config.yml` | `config.yml` | `userdata/`, `warps/` |
| EssentialsX Spawn | `configs/lobby/plugins/essentialsxspawn` (`config-target-dir: EssentialsSpawn`) | `plugins/EssentialsSpawn/config.yml` | `config.yml` | spawn state |
| DeluxeMenus | `configs/lobby/plugins/deluxemenus` (`config-target-dir: DeluxeMenus`) | `plugins/DeluxeMenus/config.yml` | `config.yml`, `gui_menus/*.yml` | menu state |
| ItemJoin | `configs/lobby/plugins/itemjoin` (`config-target-dir: ItemJoin`) | `plugins/ItemJoin/config.yml` | `config.yml`, `items.yml` | join item setup, navigation compass, anti-tamper inventory slots |
| DecentHolograms | `configs/lobby/plugins/decentholograms` (`config-target-dir: DecentHolograms`) | `plugins/DecentHolograms/config.yml` | `config.yml` | `holograms/` state |
| SkinsRestorer | `configs/lobby/plugins/skinsrestorer` (`config-target-dir: SkinsRestorer`) | `plugins/SkinsRestorer/config.yml` | `config.yml` | backend skin applicator state |

### Survival Backend Plugins (Purpur)

| Plugin | Git source | Runtime path | Managed files | Purpose & Details |
|---|---|---|---|---|
| CoreProtect | `configs/survival/plugins/coreprotect` | `plugins/CoreProtect/config.yml` | `config.yml` | Block and container transaction auditing, rollback engine |
| GrimAC | `configs/survival/plugins/grimac` | `plugins/GrimAC/config.yml` | `config.yml` | Physics-simulation asynchronous movement & combat anticheat |
| Raycasted Anti-ESP | `configs/survival/plugins/raycasted-anti-esp` | `plugins/RaycastedAntiESP/config.yml` | `config.yml` | Raytracing raycast engine blocking entity/chest x-ray |
| AnarchyExploitFixes | `configs/survival/plugins/anarchyexploitfixes` | `plugins/AnarchyExploitFixes/config.yml` | `config.yml` | Crash, dupe, lag machine, and packet overflow prevention |
| Chunky | `configs/survival/plugins/chunky` | `plugins/Chunky/config.yml` | `config.yml` | Asynchronous chunk pregenerator |
| TerraformGenerator | `configs/survival/plugins/terraformgenerator` | `plugins/TerraformGenerator/config.yml` | `config.yml` | Procedural Overworld terrain generation |
| BetterStructures | `configs/survival/plugins/betterstructures` | `plugins/BetterStructures/config.yml` | `config.yml` | Custom dungeons, ruins, and underground structures |
| GSit | `configs/survival/plugins/gsit` | `plugins/GSit/config.yml` | `config.yml` | Poses (`/sit`, `/lay`, `/crawl`, `/spin`), seat sneaking, Polish strings |
| BetterSleeping | `configs/survival/plugins/bettersleeping` | `plugins/BetterSleeping4/config.yml` | `config.yml` | 20% sleeping threshold to skip night, morning buffs, Polish alerts |
| InteractiveChat | `configs/survival/plugins/interactivechat` | `plugins/InteractiveChat/config.yml` | `config.yml` | Chat tags `[item]`, `[inv]`, `[ender]` with interactive GUI previews |
| InteractiveChat-PacketEvents | `configs/survival/plugins/interactivechat-packetevents` | `plugins/InteractiveChat-PacketEvents/config.yml` | `config.yml` | PacketEvents protocol provider bridge for InteractiveChat |
| eco | `configs/survival/plugins/eco` | `plugins/eco/config.yml` | `config.yml` | Core Auxilor runtime and item engine |
| libreforge | `configs/survival/plugins/libreforge` | `plugins/libreforge/config.yml` | `config.yml` | Unified effect/condition engine for Eco content |
| EcoSkills | `configs/survival/plugins/ecoskills` | `plugins/EcoSkills/config.yml` | `config.yml` | 6-path skill progression (Mining, Farming, Combat, etc.) with Polish GUI |
| EcoEnchants | `configs/survival/plugins/ecoenchants` | `plugins/EcoEnchants/config.yml` | `config.yml` | Balanced custom enchantments |
| EcoItems | `configs/survival/plugins/ecoitems` | `plugins/EcoItems/config.yml` | `config.yml` | Custom items and equipment with Paper block-update safeguards |
| PvPManager | `configs/survival/plugins/pvpmanager` | `plugins/PvPManager/config.yml` | `config.yml` | Player PvP toggling (`/pvp on|off`), combat tagging |
| UltimateTeams | `configs/survival/plugins/ultimate-teams` | `plugins/UltimateTeams/config.yml` | `config.yml` | Team clanning, friendly-fire immunity, team chat |
| VoiceChat | `configs/survival/plugins/voicechat` | `plugins/voicechat/voicechat-server.properties` | `voicechat-server.properties` | Backend proximity voice chat server |
| ToggleDimensions | `configs/survival/plugins/dimension-toggler` | `plugins/ToggleDimensions/config.yml` | `config.yml` | Live dimension toggling commands (`/toggleend`, `/togglenether`) |
| InvSee++ | `configs/survival/plugins/invsee-plus-plus` | `plugins/InvSeePlusPlus/config.yml` | `config.yml` | Administrative inventory and enderchest inspection |
| InventoryRollbackPlus | `configs/survival/plugins/inventoryrollbackplus` | `plugins/InventoryRollbackPlus/config.yml` | `config.yml`, `messages.yml` | Automatic player inventory backups and administrative rollbacks |
| Anti-AutoTotem | `configs/survival/plugins/anti-autototem` | `plugins/Anti-AutoTotem/config.yml` | `config.yml` | PacketEvents-based cheat detection blocking automated totem equipping |
| LibertyBans | `configs/survival/plugins/libertybans` | `plugins/LibertyBans/config.yml` | `config.yml` | Network punishment management with PostgreSQL backing |

### Server-Side Datapacks (`configs/plugins/components.yml`)

| Datapack | Version | Installation Directory | Source | Description |
|---|---|---|---|---|
| **Structory** | `1.3.17` | `servers/survival/world/datapacks/` | Modrinth | Atmospheric Overworld structures, settlements, and ruins. 100% server-side, zero client downloads. |
| **Structory: Towers** | `1.0.17` | `servers/survival/world/datapacks/` | Modrinth | Custom observation towers and exploration landmarks. 100% server-side, zero client downloads. |
| **Dungeons and Taverns** | `v5.1.0` | `servers/survival/world/datapacks/` | Modrinth | Sprawling dungeons, taverns, castles, and underground labyrinths. 100% server-side, zero client downloads. |
| **Incendium** | `5.4.11` | `servers/survival/world/datapacks/` | Modrinth | Complete overhaul of the **Nether** dimension (8 biomes, Sanctum fortresses). 100% server-side, zero client downloads. |
| **Nullscape** | `1.2.17` | `servers/survival/world/datapacks/` | Modrinth | Complete overhaul of **The End** dimension (massive islands up to Y=384, Crystalline Peaks). 100% server-side, zero client downloads. |


The exact filenames and schemas were checked against the pinned artifacts. VelocityScoreboardAPI has no external configuration file. SkinsRestorer runs on both the Velocity proxy and every backend (Lobby, and future Survival) in accordance with official SkinsRestorer proxy architecture; backend instances communicate with the proxy via plugin messaging without requiring a separate database or extra ports.

The LuckPerms password marker is deliberately not a valid production credential. Runtime preparation renders it from the local mode-0600 `secrets/luckperms_password` after synchronization; the Git file remains secret-free. PostgreSQL initializes the `luckperms` database and dedicated `luckperms_user` on first startup via `infrastructure/postgres/init-databases.sh`. Both Proxy and backends connect to PostgreSQL over `data_net` to synchronize permissions network-wide. Local H2/SQLite storage for LuckPerms is prohibited in production as it would break permission synchronization across servers.

### LuckPerms PostgreSQL Failure Modes and Synchronization
When PostgreSQL is temporarily unavailable:
- **In-memory cache evaluation:** Existing connected players continue to have their permissions evaluated without disruption from LuckPerms' in-memory cache (`PermissionHolder` / `NodeMap`).
- **New player logins:** When new players join, LuckPerms attempts to fetch their permissions from PostgreSQL. On Velocity and Paper, database connection failures log errors, and unretrieved player data falls back to default in-memory permissions or is gated by proxy auth.
- **Administrative operations:** Modifying permissions, groups, or tracks (`/lp user ...`, `/lp group ...`) requires database writes and will fail or time out during an outage.
- **Cluster recovery:** Upon database restoration, LuckPerms automatically restores connections via its HikariCP pool. Cross-server synchronization resumes immediately via SQL messaging queue (`luckperms_messenger`) with a periodic 3-minute reconciliation fallback (`sync-minutes: 3`).

### Matching backend configuration for LuckPerms (Lobby and Survival)
When LuckPerms is installed on backend Paper/Purpur servers, its configuration (`plugins/LuckPerms/config.yml`) must mirror the proxy database storage settings:
```yaml
server: lobby # or 'survival' on Survival server
storage-method: postgresql
data:
  address: postgres:5432
  database: luckperms
  username: luckperms_user
  password: __LUCKPERMS_PASSWORD__
  pool-settings:
    maximum-pool-size: 10
    minimum-idle: 2
    connection-timeout: 5000
  table-prefix: luckperms_
  properties:
    sslmode: disable
sync-minutes: 3
messaging-service: pluginmsg
```
Using the same database and credentials ensures consistent rank, group, and permission state across Proxy, Lobby, and Survival.

All plugins with external configuration are repository-managed through `config-dir`. FancyNpcs has no repository-managed config directory; its NPCs are configured with the plugin commands and remain runtime data. The manager copies managed directories atomically into the corresponding runtime plugin paths and includes replaced files in rollback metadata. VelocityScoreboardAPI has no configuration file. Runtime databases, caches, metrics and player data are never copied from Git and are never deleted by synchronization.

### FancyNpcs and Java 25

FancyNpcs `2.12.1` is pinned to Modrinth version ID `LigxTtVw`. The lockfile records the official download URL, size, SHA-256 and SHA-512. Its artifact declares Paper API `1.19`, supports Paper/Folia and declares Citizens only as an optional dependency; Citizens is therefore not installed. The project documentation states that Paper forks such as Purpur should work, but this repository has not yet performed a runtime test on Purpur `1.21.11`.

The Lobby container uses the separately pinned `PURPUR_LOBBY_IMAGE` based on Java 25. Existing Lobby plugins declare Java 8–21 minimums and their pinned artifacts target Minecraft `1.21.11`/Paper; Java 25 is backward-compatible with these classfiles at the static level. This is declared/static compatibility evidence, not proof of runtime compatibility. Docker execution and staging remain required before production approval.

## Commands

```bash
./bin/network plugins validate
./bin/network plugins list
./bin/network plugins plan
./bin/network plugins verify
./bin/network plugins update-check
./bin/network plugins sync --dry-run
./bin/network plugins sync --apply
./bin/network plugins components --apply
./bin/network plugins rollback
```

`sync --apply` applies only the exact manifest and lockfile state. It does not query for newer releases, restart services or install arbitrary transitive dependencies. Target services must be stopped. `rollback` also requires stopped target services and restores the most recent transaction snapshot.

`network install` performs plugin and NanoLimbo component synchronization before rendering runtime secrets and before any service is started. A later `network start` only renders the secret-backed runtime values and starts Compose; it does not silently download or replace artifacts.

## Lobby configuration templates

The versioned Lobby plugin configuration is stored under `configs/lobby/plugins/<plugin>/`. The complete directory tree is copied to `servers/lobby/plugins/<PluginDirectory>/` during Plugin Manager synchronization, including nested YAML, JSON and text files declared by the plugin. Runtime-only data such as JARs, caches, databases, libraries, logs, remapped artifacts and player data is excluded from the template and remains local.

Edit the template under `configs/lobby/plugins/`, commit it to Git, and run `network install` or `network plugins sync --apply` on another machine to reproduce the Lobby configuration. Secret-backed values must use placeholders such as `__LUCKPERMS_PASSWORD__`; the native CLI renders them only into the ignored runtime tree.

## Service configuration boundaries

Plugin configuration must live next to the service that owns it. Proxy plugins use
`configs/proxy/plugins/`, Lobby plugins use `configs/lobby/plugins/`, and future
Survival plugins use `configs/survival/plugins/`. NanoLimbo is configured from
`configs/auth/settings.yml`. The Go CLI is the only component that renders these
templates into `servers/*`; this keeps a fresh machine reproducible without making
runtime directories part of Git.

Global network branding is defined in `configs/network.yml`. The native renderer
replaces the `__NETWORK_*__`, `__LOBBY_*__` and `__AUTH_*__` tokens in managed
configuration files and rejects unresolved tokens before startup.

## Providers

The provider contract requires exact immutable identifiers. Modrinth is implemented against its version API and consumes file SHA-512/SHA-256 metadata. GitHub is supported through exact repository/tag/asset metadata and never falls back to a floating release. Hangar is deliberately not supported until an official API adapter with deterministic checksums is required; it is not exposed as an available provider.

URLs are HTTPS-only and restricted to trusted hosts. Redirects are bounded and cannot leave the trusted host set. Downloads are size-limited, staged outside runtime directories, hashed before publication and checked as JAR/ZIP metadata without executing the artifact.

## Placement

The manager maps targets to the real Compose volumes:

| Target | Runtime directory |
|---|---|
| `proxy` | `servers/proxy/plugins` |
| `lobby` | `servers/lobby/plugins` |
| `survival` | `servers/survival/plugins` |

Velocity plugins are rejected for backend targets and Paper plugins are rejected for the Proxy target. JAR metadata is checked for `velocity-plugin.json`, `plugin.yml` or `paper-plugin.yml`.

## Safe lifecycle

1. Strictly parse manifest and lockfile.
2. Check version IDs, compatibility, dependency graph and hashes.
3. Verify all target services are stopped.
4. Download missing artifacts into a temporary staging directory.
5. Validate digest, size, filename and plugin descriptor.
6. Validate every managed configuration directory before installation; YAML files must parse and symlinks/non-regular files are rejected.
7. Back up replaced files into `.plugin-manager/rollback/`.
8. Atomically rename staged JARs and configuration files into their target directories.
9. Write installed state and rollback metadata.

If a manifest entry has `config-dir`, its regular files are copied into a directory named after that source directory below the target plugin directory. Symlinks and non-regular files are rejected, and the copy is included in the same rollback transaction as the JAR.

Any failure rolls back files changed by the transaction. Removal is not implicit; disabled entries are reported in the plan and future CLI confirmation must be explicit. Plugin data directories are never deleted automatically.

## Updates and custom plugins

Updates are manual. A new provider version must be reviewed, locked and approved before sync. Configuration migrations are not automatic: operators must compare the new upstream schema, preserve the old config and add an explicit migration note before changing the pinned version. The manager never deletes unmanaged runtime files or plugin data. First-party custom plugins are not currently exposed through the automated manager; adding them requires a reviewed provider and lockfile contract.
