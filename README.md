# Minecraft Network

[![CI](https://github.com/kizio806/minecraft-network/actions/workflows/ci.yml/badge.svg)](https://github.com/kizio806/minecraft-network/actions/workflows/ci.yml)
[![CodeQL](https://github.com/kizio806/minecraft-network/actions/workflows/codeql.yml/badge.svg)](https://github.com/kizio806/minecraft-network/actions/workflows/codeql.yml)
[![codecov](https://codecov.io/gh/kizio806/minecraft-network/graph/badge.svg)](https://codecov.io/gh/kizio806/minecraft-network)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=kizio806_minecraft-network&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=kizio806_minecraft-network)
[![Go Report Card](https://goreportcard.com/badge/github.com/kizio806/minecraft-network)](https://goreportcard.com/report/github.com/kizio806/minecraft-network)
[![Go Version](https://img.shields.io/badge/go-1.24-00ADD8?logo=go)](go.mod)
[![Minecraft Version](https://img.shields.io/badge/minecraft-1.21.11-brightgreen?logo=minecraft)](configs/network.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> Architectural specification: [PROJECT.md](PROJECT.md)  
> Operational & AI manual: [AGENTS.md](AGENTS.md)

**Minecraft Network** is a production-grade, highly optimized, declarative Infrastructure-as-Code (IaC) server network designed for Minecraft 1.21.x. It combines modern containerization, Zero-Trust networking, encrypted asymmetric backups, semantic color theming, and multi-language localization.

---

## Key Highlights

- **Zero-Trust Network Topology:** Backend servers (`auth`, `lobby`, `survival`) and database (`postgres`) publish zero host ports. Only Velocity proxy exposes `25565/tcp` to the public network.
- **Strict Mixed-Mode Authentication:** Hybrid Mojang Premium cryptographic handshake + BCrypt Non-Premium limbo authentication with PostgreSQL backing and nickname spoofing protection (`allow-cracked-on-premium-nicks: false`).
- **Declarative Plugin Supply Chain:** Every plugin JAR and datapack is declared in `configs/plugins/manifest.yml` and pinned by cryptographic SHA-512 / SHA-256 digests in lockfiles. Zero binary bloat in Git.
- **Encrypted Age Backups:** Native asymmetric cryptography via `age` (`*.tar.gz.age`) with offsite identity separation and SHA-256 integrity files.
- **Database Consolidation & Zero Redis:** All permissions (LuckPerms), logins (VeloAuth), teams, and punishments (LibertyBans) reside in isolated schemas on PostgreSQL 16 Alpine. Zero Redis bloat.
- **Dynamic Theming & Localization (i18n):** Switch color palettes (`amber-sunset`, `cyber-blue`, `emerald-forest`, `amethyst-purple`) and language bundles (`en`, `pl`) in seconds from `configs/network.yml`.
- **Turnkey Administration & Moderation Suite:** Declarative maintenance schedules (`Maintenance`), network-wide proxy staff chat (`CleanStaffChat`), proxy reporting system (`VelocityReport`), dynamic priority queues (`ajQueue`), and stealth administrator vanish (`SuperVanish`).
- **Native Discord Bot (Go 1.24):** Containerized bot using modern Discord Components V2 (ActionRow buttons, ephemeral modals for `/link` and verification, live status dashboard, whitelist management, and chat bridge) directly connected to the stack.
- **Vanilla+ Enhanced SMP:** 3-dimension world overhauls (TerraformGenerator Overworld, Incendium Nether, Nullscape End) running 100% server-side with zero required client-side mods.

---

## Architecture Overview

```text
Internet
   |
TCP 25565
   v
[Velocity Proxy] -- proxy_net --> [NanoLimbo auth]
       |                           [Lobby Hub]
       |                           [Survival SMP]
       |                                |
       +-------- data_net (internal) ---+
                         |
                  [PostgreSQL 16] (luckperms, veloauth, ultimateteams, libertybans)
```

---

## Quickstart

### 1. Prerequisites
- Linux host (Ubuntu 22.04+ LTS recommended) or macOS
- Docker Engine 26+ and Docker Compose v2.27+
- Go 1.24+ (Optional: if Go is not installed on the host, `make` automatically builds and tests inside a pinned Docker container)

### 2. Interactive Setup Wizard
Clone the repository and run the setup wizard:
```bash
git clone https://github.com/kizio806/minecraft-network.git
cd minecraft-network

# Build the native Go management CLI (via host Go or automated Docker fallback)
make build

# Launch the interactive terminal onboarding wizard (TUI)
./bin/network init
```

The wizard will interactively prompt you for:
- Network title & short tag
- Primary domain connection address
- Active localization (`en` or `pl`)
- Semantic color theme preset (`amber-sunset`, `cyber-blue`, `emerald-forest`, `amethyst-purple`)
- Global player capacity & social links

It then automatically initializes `.env`, generates 256-bit cryptographically secure secrets (`0600`), validates the Docker topology, and synchronizes all pinned plugins.

*For automated or CI deployments, pass the non-interactive flags:*
```bash
./bin/network init --non-interactive --name "Aetheria SMP" --locale en --theme cyber-blue --max-players 100
```

### 3. Start the Network
```bash
# Start all network services (Velocity, NanoLimbo, Lobby, Survival, PostgreSQL)
./bin/network start

# Check real-time container health
./bin/network health
```

---

## Live Demo & Local Sandbox Guide

You can run and evaluate the entire network stack locally on any development machine without public DNS, registered domain names, or external Discord bot credentials.

### Step 1: Initialize Local Sandbox
Run the non-interactive setup targeting localhost:
```bash
./bin/network init --non-interactive --name "Local Sandbox" --domain "localhost" --locale en --theme cyber-blue --max-players 20
```

### Step 2: Boot Services & Check Health
```bash
# Start containers (daemonized via docker compose)
./bin/network start

# Confirm all healthchecks pass
./bin/network health
```

### Step 3: Connect with Minecraft Client
1. Launch **Minecraft Java Edition** (version **1.21.x**).
2. Open **Multiplayer** -> **Direct Connection** (or **Add Server**).
3. Connect to server address: `127.0.0.1` or `localhost` (port `25565` is default).
4. **Gameplay Flow:**
   - Premium Mojang accounts complete cryptographic verification and route directly to the **Lobby**.
   - Offline / Non-premium accounts enter the **NanoLimbo** holding pen and receive an on-screen prompt to register or authenticate via VeloAuth (`/register <password>` or `/login <password>`).
   - Use `/server survival` or pass through the lobby portal to transition to the enhanced Survival SMP world.

### Step 4: Local Discord Bot & Console Testing
The Discord bot supports running locally without a Discord token in standby mode for testing, or with a developer bot token:
```bash
# Standby check (verifies configuration and healthcheck handler)
make build-bot
./bin/discord-bot -health

# Execute console commands directly via CLI or native RCON
./bin/network status
docker compose exec -T survival rcon-cli list
```

### Step 5: Teardown Sandbox
```bash
# Gracefully stop containers and preserve persistent database volumes
./bin/network stop
```

---

## Management CLI (`./bin/network`)

The native Go binary `./bin/network` manages the entire operational lifecycle:

```text
network [flags] <command>

Commands:
  init               Launch interactive terminal wizard to configure branding, locale and limits
  install            Prepare directories, render configs, generate secrets (0600), sync plugins & datapacks
  start              Launch services (optional: --monitoring)
  stop               Safely shut down services
  restart            Restart services with recalculated templates
  status             Display container states and health checks
  health             Verify cluster readiness and port availability
  logs [service]     Tail service logs (--follow, --lines N)
  console <service>  Attach interactively to server console (proxy, lobby, survival)
  backup             Create age-encrypted archive with PostgreSQL dump
  restore <file>     Atomically restore state from encrypted backup (--yes)
  update             Check pinned images and plugin versions (--dry-run, --apply)
  plugins <cmd>      Validate, list, plan, verify, sync, components, rollback
  completion [sh]    Generate shell autocompletion script (bash, zsh, fish)
```

### Shell Autocompletion
Generate instant tab-completion for commands, console-enabled servers, container services, and flags:
```bash
# Bash
source <(./bin/network completion bash)

# Zsh
source <(./bin/network completion zsh)

# Fish
./bin/network completion fish | source
```

---

## Developer Tooling & Makefile

The repository includes a comprehensive `Makefile` with zero host dependencies (automatically falling back to isolated Docker containers if `go` or `golangci-lint` are not installed locally):

```bash
make build       # Compile ./bin/network with embedded Git commit and build date
make build-bot   # Compile ./bin/discord-bot native Go binary
make test        # Run unit tests across all packages
make test-race   # Run unit tests with Go race detector (-race)
make lint        # Run golangci-lint with gosec, errcheck, staticcheck
make fmt         # Format Go source code
make vet         # Run go vet
make doctor      # Execute full system diagnostic check
make health      # Inspect running container health status
```

---

## Testing & Quality Assurance

Run the comprehensive verification suite locally:
```bash
# Unit tests with race detection
go test -v -race ./...

# Validate plugin and component supply chain manifests
./bin/network plugins validate
./bin/network plugins components --dry-run

# Compose syntax verification
docker compose -f docker-compose.yml config --quiet
docker compose -f docker-compose.yml --profile monitoring config --quiet
```

---

## Contributing, Support & License

- **Governance & Contributing:** [CONTRIBUTING.md](CONTRIBUTING.md)
- **Code of Conduct:** [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- **Support & Inquiries:** [SUPPORT.md](SUPPORT.md)
- **Project Roadmap:** [ROADMAP.md](ROADMAP.md)
- **Security Policy:** [SECURITY.md](SECURITY.md)

This project is licensed under the [MIT License](LICENSE).
