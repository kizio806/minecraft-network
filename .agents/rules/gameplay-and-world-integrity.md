# Rule: Gameplay & World Integrity

## Purpose
Protect gameplay mechanics, spawn security, griefing rollback capabilities, and prevent runtime world data from polluting version control.

## Scope
Applies to world data folders (`servers/*/world/`, `servers/*/lobby/`), WorldGuard region protections, CoreProtect block logging, and spawn configuration templates.

## MUST
- **MUST** exclude all live Minecraft world saves, dynamic chunk data, player stats, and session locks from Git:
  - `session.lock`, `uid.dat`, `level.dat_old`, `playerdata/`, `stats/`, `advancements/`.
- **MUST** protect the Lobby spawn world declaratively using WorldGuard flags in `configs/lobby/plugins/worldguard/`:
  - Deny: `pvp`, `mob-spawning`, `block-break`, `block-place`, `fire-spread`, `hunger`, `fall-damage`.
- **MUST** enable CoreProtect on the Survival server to record all block placements, breaks, container transactions, and chat events.
- **MUST** keep world template backups inside `world-archives/` rather than committing live world folders.
- **MUST** enforce TerraformGenerator as the standard Overworld generator in `bukkit.yml` (ADR 0016), keeping Terralith permanently excluded to prevent dimension codec and biome provider conflicts.
- **MUST** deploy all dimension and exploration datapacks (**Structory**, **Structory: Towers**, **Dungeons and Taverns**, **Incendium**, **Nullscape**) in 100% server-side mode within `servers/survival/world/datapacks/`.
- **MUST** configure all player-facing strings, menus, actionbars, and alerts for GSit, BetterSleeping, InteractiveChat, EcoSkills, InvSee++, and InventoryRollbackPlus in **Polish**.
- **MUST** enforce team friendly-fire immunity in UltimateTeams (`friendly-fire: false`) and voluntary PvP toggling (`/pvp on|off`).
- **MUST** protect custom block textures in Paper (`block-updates.disable-noteblock-updates: true`, `block-updates.disable-tripwire-updates: true`) to prevent EcoItems visual glitches.

## MUST NOT
- **MUST NOT** track dynamic player or session files in Git.
- **MUST NOT** allow PvP, mob griefing, or block destruction in the Lobby world.
- **MUST NOT** disable CoreProtect transaction logging on the Survival gameplay instance.
- **MUST NOT** reintroduce `Terralith` to the Overworld server stack to prevent Bukkit biome provider overriding and dimension deserialization failure (ADR 0016).
- **MUST NOT** require players to install client-side mods or download resource packs to experience Structory, Structory: Towers, Dungeons and Taverns, Incendium, and Nullscape dimension overhauls.
- **MUST NOT** alter EcoSkills GUI rows without ensuring all components fit within the allocated row count.
- **MUST NOT** execute `/stop` directly from server consoles; always use `./bin/network stop` for safe volume flushing.

## Verification
Inspect git status for world and session files:
```bash
git status -s servers/
```
No world data, region chunk files, or player data must appear as untracked or modified.
