# Discord Bot Architecture & Integration Guide — minecraft-network

## 1. Overview

The `discord-bot` daemon is a native Go 1.24 service communicating with Discord via `discordgo`. It acts as an out-of-band operations bridge for player account linking, in-game moderation reports, real-time status dashboards, and remote server administration.

---

## 2. Privileged Gateway Intents

The bot requires the following Discord application intents configured in the Discord Developer Portal:
- **Server Members Intent (`discordgo.IntentsGuildMembers`):** Required for synchronizing user roles upon account verification (`/link` and `/sync`).
- **Guild Messages Intent (`discordgo.IntentsGuildMessages`):** Required for ingesting chat bridge messages when enabled.
- **Message Content Intent:** Required for chat bridge text decoding.

---

## 3. Slash Commands & UI Components

The bot leverages modern Discord Components V2 (ActionRows, Buttons, TextInputs, Modals):

| Command / Component | Type | Description |
|---|---|---|
| `/status` | Slash Command | Posts or updates real-time server player counts and ping status. |
| `/players` | Slash Command | Lists online players per backend server (`lobby`, `survival`). |
| `/link [pin]` | Slash Command | Links a Minecraft account to Discord using a verification PIN. |
| `/sync` | Slash Command | Synchronizes LuckPerms group permissions to Discord guild roles. |
| `/report` | Slash Command | Submits an in-game moderation report to the staff review channel. |
| `/rcon` | Slash Command (Admin) | Executes an administrative console command via native TCP RCON. |
| `IDBtnReportPunish` | Interactive Button | Opens an ephemeral modal allowing staff to issue bans or tempbans. |
| `IDBtnReportDismiss` | Interactive Button | Marks an in-game report dismissed without action. |

---

## 4. PostgreSQL Persistence & Schema

The bot connects directly to the consolidated PostgreSQL 16 container (`luckperms` database):

### Managed Schemas
1. **`discord_links`:**
   - `discord_id VARCHAR(64) PRIMARY KEY`
   - `username VARCHAR(64) NOT NULL`
   - `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
2. **`discord_whitelist`:**
   - `username VARCHAR(64) PRIMARY KEY`
   - `added_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
3. **`discord_reports`:**
   - `id BIGSERIAL PRIMARY KEY`
   - `reporter_id VARCHAR(64) NOT NULL`
   - `reporter_tag VARCHAR(100) NOT NULL`
   - `target_player VARCHAR(64) NOT NULL`
   - `reason TEXT NOT NULL`
   - `status VARCHAR(32) NOT NULL DEFAULT 'pending'`
   - `resolved_by VARCHAR(64)`
   - `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
   - `resolved_at TIMESTAMPTZ`

### Read-Only Permissions Lookup
- **`luckperms_players`:** Queries `primary_group` for rank synchronization.

---

## 5. Native TCP RCON Architecture

Prior versions used Docker CLI calls (`docker compose exec`) from inside the container, which failed in lightweight scratch or Alpine containers without a Docker socket.

The bot now connects directly over the container network (`proxy_net`) using native Source RCON protocol over TCP (`internal/discord/rcon_client.go`):
- **Port:** `25575/tcp`
- **Network Targets:** `lobby:25575`, `survival:25575`
- **Security:**
  - Input validation: Usernames are verified with `^[a-zA-Z0-9_]{3,16}$`.
  - Sanitization: All punishment reasons are stripped of newline (`\n`, `\r`) and null (`\0`) control characters to prevent command injection.
  - No Docker socket or elevated host privileges required.
