# Developer & Engineering Guide — minecraft-network

## 1. Overview

This document provides developer guidelines for building, debugging, extending, and testing the `minecraft-network` codebase.

The repository consists of two primary Go binaries:
1. `cmd/network`: Operational CLI for initialization, supply-chain validation, backup management, and orchestration.
2. `cmd/discord-bot`: Discord daemon managing slash commands, interactive modals, player rank synchronization, and RCON administration.

---

## 2. Prerequisites & Environment Setup

- **Go 1.24+** (Optional: Makefile includes containerized fallback to `golang:1.24`)
- **Docker 26+** and **Docker Compose v2.27+**
- **Git**
- Optional: `golangci-lint` (v1.64.5+ via container or local binary)

---

## 3. Building Binaries

Build both tools using the native Makefile:

```bash
# Build the management CLI (outputs to ./bin/network)
make build

# Build the Discord bot daemon (outputs to ./bin/discord-bot)
make build-bot

# Build both binaries simultaneously
make build-all
```

If Go is not installed natively on your host system, the Makefile automatically compiles inside a pinned `golang:1.24-alpine` Docker container.

---

## 4. Running the Discord Bot Locally

The Discord bot can run in standalone development mode:

```bash
# Verify healthcheck handler
./bin/discord-bot -health

# Run locally against custom configuration and locales
./bin/discord-bot -config configs/discord/config.yml -locales locales -lang en
```

### Environment Variables
- `DISCORD_BOT_TOKEN`: The Discord bot token (or secret in `/run/secrets/discord_bot_token` / `secrets/discord_bot_token`). If empty or placeholder, the bot enters safe standby mode.
- `DATABASE_URL`: PostgreSQL connection string (defaults to `postgres://minecraft:<password>@postgres:5432/luckperms?sslmode=disable`).
- `RCON_PASSWORD`: Password for remote RCON console execution.
- `RCON_LOBBY_HOST`: RCON address for lobby (default: `lobby:25575` or `127.0.0.1:25575`).
- `RCON_SURVIVAL_HOST`: RCON address for survival (default: `survival:25575` or `127.0.0.1:25575`).

---

## 5. Quality Assurance & Testing

### Running Tests with Race Detector
Always execute tests with `-race` to ensure race-condition immunity:

```bash
# Native test execution
go test -v -race ./...

# Containerized test execution
make test
```

### Code Formatting & Linting
Enforce code formatting and static analysis:

```bash
# Format source files with gofmt and goimports
make fmt

# Run comprehensive linters (revive, gosec, bodyclose, errorlint)
make lint
```

---

## 6. Architecture & Security Invariants

When adding or modifying code, adhere strictly to the invariants defined in [AGENTS.md](../AGENTS.md):
1. **Never use Docker CLI from inside non-privileged containers:** Use native network protocols (TCP RCON, PostgreSQL wire protocol) rather than shelling out to `docker compose exec`.
2. **Sanitize all user input:** Always validate Minecraft usernames against `^[a-zA-Z0-9_]{3,16}$` and strip newline characters from punishment reasons.
3. **Keep secrets secure:** Files in `secrets/` must strictly maintain `0600` permissions.
4. **Preserve Zero-Trust topology:** Only the Velocity proxy container exposes external ports (`25565`).
