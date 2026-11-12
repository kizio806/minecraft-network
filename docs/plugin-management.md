# Plugin management

Plugin management is declarative and deterministic. The reusable Go core is available both as `plugin-manager` and through `network plugins`.

## Files

- `configs/plugins/manifest.yml` — desired plugin state and compatibility policy.
- `configs/plugins/lock.yml` — exact provider version, immutable version ID, URL, filename, size and trusted digest.
- `configs/plugins/components.yml` — standalone runtime components such as NanoLimbo; these are not plugins.
- `configs/plugins/components.lock.yml` — immutable artifact lock for standalone components.
- `configs/plugins/manifest.schema.json` — JSON Schema for external/editor validation.

The proxy manifest contains VeloAuth `1.6.2`, LuckPerms `v5.5.71-velocity`, MiniMOTD `2.2.2`, TAB `6.2.0`, VelocityScoreboardAPI `2.1.1` and SkinsRestorer `15.12.6`. The Lobby backend manifest contains FancyNpcs `2.12.1`, LuckPerms `v5.5.71-bukkit`, PlaceholderAPI `2.11.7`, WorldEdit `7.3.18`, WorldGuard `7.0.16`, WorldGuard Extra Flags `4.2.4`, Multiverse-Core `5.5.3`, EssentialsX `2.22.0`, EssentialsX Spawn `2.22.0`, DeluxeMenus `1.14.1-Release`, CommandPanels `4.1.6`, DecentHolograms `2.10.2` and SkinsRestorer `15.12.6`. Citizens and ProtocolLib are not part of the active set and are not installed. NanoLimbo is classified separately as a standalone runtime component. Survival backend plugins are intentionally deferred.

### Velocity Proxy Plugins

| Plugin | Git source | Runtime path | Managed files | Generated/runtime data |
|---|---|---|---|---|
| VeloAuth | `configs/velocity/plugins/veloauth` | `plugins/veloauth/config.yml` | `config.yml` | `data/`, language/runtime state |
| LuckPerms | `configs/velocity/plugins/luckperms` | `plugins/luckperms/config.yml` | `config.yml` | PostgreSQL state; password marker rendered from `secrets/luckperms_password` before startup |
| MiniMOTD | `configs/velocity/plugins/minimotd` | `plugins/minimotd/main.conf`, `plugin_settings.conf` | both HOCON files | icons and runtime cache only |
| TAB | `configs/velocity/plugins/tab` (`config-target-dir: TAB`) | `plugins/TAB/config/*.yml` | `config/*.yml` | runtime cache and metrics only |
| VelocityScoreboardAPI | none | plugin API only | none | none |
| SkinsRestorer | `configs/velocity/plugins/skinsrestorer` (`config-target-dir: SkinsRestorer`) | `plugins/SkinsRestorer/config.yml` | `config.yml` | skin cache/database and generated metadata |

### Lobby Backend Plugins (Purpur)

| Plugin | Git source | Runtime path | Managed files | Generated/runtime data |
|---|---|---|---|---|
| LuckPerms | `configs/purpur/plugins/luckperms` (`config-target-dir: LuckPerms`) | `plugins/LuckPerms/config.yml` | `config.yml` | PostgreSQL shared state; password rendered from `secrets/luckperms_password` |
| PlaceholderAPI | `configs/purpur/plugins/placeholderapi` (`config-target-dir: PlaceholderAPI`) | `plugins/PlaceholderAPI/config.yml` | `config.yml` | `expansions/` cache |
| FancyNpcs | none | `plugins/FancyNpcs-2.12.1.jar` | none | NPC definitions and runtime state configured in-game |
| WorldEdit | `configs/purpur/plugins/worldedit` (`config-target-dir: WorldEdit`) | `plugins/WorldEdit/config.yml` | `config.yml` | schematics, session history |
| WorldGuard | `configs/purpur/plugins/worldguard` (`config-target-dir: WorldGuard`) | `plugins/WorldGuard/config.yml` | `config.yml` | `worlds/<world>/regions.yml` |
| WorldGuard Extra Flags | `configs/purpur/plugins/worldguardextraflags` (`config-target-dir: WorldGuardExtraFlags`) | `plugins/WorldGuardExtraFlags/config.yml` | `config.yml` | custom flags state |
| Multiverse-Core | `configs/purpur/plugins/multiverse-core` (`config-target-dir: Multiverse-Core`) | `plugins/Multiverse-Core/config.yml` | `config.yml` | `worlds.yml` |
| EssentialsX | `configs/purpur/plugins/essentialsx` (`config-target-dir: Essentials`) | `plugins/Essentials/config.yml` | `config.yml` | `userdata/`, `warps/` |
| EssentialsX Spawn | `configs/purpur/plugins/essentialsxspawn` (`config-target-dir: EssentialsSpawn`) | `plugins/EssentialsSpawn/config.yml` | `config.yml` | spawn state |
| DeluxeMenus | `configs/purpur/plugins/deluxemenus` (`config-target-dir: DeluxeMenus`) | `plugins/DeluxeMenus/config.yml` | `config.yml`, `gui_menus/*.yml` | menu state |
| CommandPanels | `configs/purpur/plugins/commandpanels` (`config-target-dir: CommandPanels`) | `plugins/CommandPanels/config.yml` | `config.yml` | panels state |
| DecentHolograms | `configs/purpur/plugins/decentholograms` (`config-target-dir: DecentHolograms`) | `plugins/DecentHolograms/config.yml` | `config.yml` | `holograms/` state |
| SkinsRestorer | `configs/purpur/plugins/skinsrestorer` (`config-target-dir: SkinsRestorer`) | `plugins/SkinsRestorer/config.yml` | `config.yml` | backend skin applicator state |

The exact filenames and schemas were checked against the pinned artifacts. VelocityScoreboardAPI has no external configuration file. SkinsRestorer runs on both the Velocity proxy and every backend (Lobby, and future Survival) in accordance with official SkinsRestorer proxy architecture; backend instances communicate with the proxy via plugin messaging without requiring a separate database or extra ports.

The LuckPerms password marker is deliberately not a valid production credential. Runtime preparation renders it from the local mode-0600 `secrets/luckperms_password` after synchronization; the Git file remains secret-free. PostgreSQL initializes the `luckperms` database and dedicated `luckperms_user` on first startup via `infrastructure/postgres/init-luckperms.sh`. Both Proxy and backends connect to PostgreSQL over `data_net` to synchronize permissions network-wide. Local H2/SQLite storage for LuckPerms is prohibited in production as it would break permission synchronization across servers.

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
messaging-service: sql
```
Using the same database and credentials ensures consistent rank, group, and permission state across Proxy, Lobby, and Survival.

All plugins with external configuration are repository-managed through `config-dir`. FancyNpcs has no repository-managed config directory; its NPCs are configured with the plugin commands and remain runtime data. The manager copies managed directories atomically into the corresponding runtime plugin paths and includes replaced files in rollback metadata. VelocityScoreboardAPI has no configuration file. Runtime databases, caches, metrics and player data are never copied from Git and are never deleted by synchronization.

### FancyNpcs and Java 25

FancyNpcs `2.12.1` is pinned to Modrinth version ID `LigxTtVw`. The lockfile records the official download URL, size, SHA-256 and SHA-512. Its artifact declares Paper API `1.19`, supports Paper/Folia and declares Citizens only as an optional dependency; Citizens is therefore not installed. The project documentation states that Paper forks such as Purpur should work, but this repository has not yet performed a runtime test on Purpur `1.21.11`.

The Lobby container uses the separately pinned `PURPUR_LOBBY_IMAGE` based on Java 25. Existing Lobby plugins declare Java 8–21 minimums and their pinned artifacts target Minecraft `1.21.11`/Paper; Java 25 is backward-compatible with these classfiles at the static level. This is declared/static compatibility evidence, not proof of runtime compatibility. Docker execution and staging remain required before production approval.

## Commands

```bash
go run ./cmd/plugin-manager validate
go run ./cmd/plugin-manager list
go run ./cmd/plugin-manager plan
go run ./cmd/plugin-manager verify
go run ./cmd/plugin-manager update-check
go run ./cmd/plugin-manager sync --dry-run
go run ./cmd/plugin-manager sync --apply
go run ./cmd/plugin-manager components --apply
go run ./cmd/plugin-manager rollback
```

`sync --apply` applies only the exact manifest and lockfile state. It does not query for newer releases, restart services or install arbitrary transitive dependencies. Target services must be stopped. `rollback` also requires stopped target services and restores the most recent transaction snapshot.

`network install` performs plugin and NanoLimbo component synchronization before rendering runtime secrets and before any service is started. A later `network start` only renders the secret-backed runtime values and starts Compose; it does not silently download or replace artifacts.

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
