# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Native Source RCON TCP client (`internal/discord/rcon_client.go`) with thread-safe connection handling, deadline propagation, and authentication.
- `NetworkRconExecutor` implementation in `internal/discord` replacing runtime shell dependencies in Discord bot container.
- Strict input validation `ValidateMinecraftUsername` (`^[a-zA-Z0-9_]{3,16}$`) and sanitization `SanitizePunishReason` against console command injection.
- Unit tests for native RCON client and executor with loopback mock server (`internal/discord/rcon_test.go`).
- Developer guide (`docs/development.md`) and Discord bot architecture guide (`docs/discord-bot.md`).
- Live Demo & Local Sandbox Guide in `README.md`.

### Changed
- Database target for Discord bot persistent storage updated to `luckperms` database to allow cross-schema player rank synchronization (`luckperms_players`).
- Moderation punishment commands routed to `survival` backend where LibertyBans operates.
- Updated `docker-compose.yml` to supply database connection environment variables (`POSTGRES_DB: luckperms`) to `discord-bot`.
- Updated `init-databases.sh` to grant full privileges to superuser on `luckperms` schema and sequences.
- Updated `.gitignore` to explicitly ignore test artifacts `cov.txt`, `coverage*.txt`, `*.test`, `*.out`.
- Updated `internal/networkcli/install.go` to explicitly surface file read errors for optional plugin passwords instead of silently ignoring failures.

### Fixed
- Fixed command injection vulnerability in Discord bot moderation modal where unescaped player names and reasons could inject console commands.
- Fixed `relation "luckperms_players" does not exist` query failure during Discord rank synchronization by aligning bot database target.
- Fixed Discord bot RCON execution failing in container due to missing Docker binary in Alpine image.
- Removed stray untracked `cov.txt` file from repository root.
