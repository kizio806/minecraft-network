# Production Deployment Guide

This guide details the end-to-end production deployment procedure for **Minecraft Network** on Ubuntu Server LTS.

---

## 1. Host Preparation

1. On a clean Ubuntu Server 22.04+ LTS host, execute the host preparation script as root:
   ```bash
   sudo ./scripts/install-host.sh
   ```
   This script configures a non-login system user (`minecraft`), sets up Docker Engine with Docker Compose v2, and installs essential dependencies.

2. Verify your active SSH port (`ss -ltnp | grep ssh`) to avoid locking out administrative access.

3. Configure firewall rules using UFW:
   ```bash
   sudo SSH_PORT=<your-ssh-port> ENABLE_UFW=1 ./scripts/configure-firewall.sh
   ```
   This restricts incoming traffic to your verified SSH port, TCP 25565, and UDP 25565 (Voice Chat).

---

## 2. Repository & Secrets Configuration

1. Copy or clone the repository to `/opt/minecraft`:
   ```bash
   sudo cp -r . /opt/minecraft
   sudo chown -R minecraft:minecraft /opt/minecraft
   cd /opt/minecraft
   ```

2. Prepare the environment configuration:
   ```bash
   cp .env.example .env
   chmod 0600 .env
   ```
   Review and ensure all pinned image digests in `.env` match approved releases.

3. Customize network branding, active theme preset, and language in `configs/network.yml`:
   ```yaml
   network:
     name: "My Server Network"
     short-name: "MyServer"
     domain: "play.example.com"
   locale: "en"  # "en" or "pl"
   theme:
     preset: "cyber-blue"  # "amber-sunset", "cyber-blue", "emerald-forest", "amethyst-purple"
   ```

---

## 3. Tooling Build & Installation

Build the control plane CLI binary:
```bash
go build -trimpath -ldflags='-X github.com/kizio806/minecraft-network/internal/networkcli.Version=1.0.0' -o ./bin/network ./cmd/network
```

Run installation and preflight checks:
```bash
# Prepare runtime configs, generate missing secrets (0600), and synchronize plugins & datapacks
./bin/network install

# Verify system readiness and secret permissions
./bin/network doctor
```

---

## 4. Starting the Network

Start the network services:
```bash
./bin/network start
```

Or start with the optional telemetry stack (Prometheus + Grafana + Uptime Kuma):
```bash
./bin/network start --monitoring
```

Verify service health:
```bash
./bin/network health
./bin/network status
```

---

## 5. First-Time Database Seeding (LuckPerms & VeloAuth)

During the first start of PostgreSQL, the `init-databases.sh` script automatically:
- Creates isolated users and databases for `luckperms`, `veloauth`, `ultimateteams`, and `libertybans`.
- Applies `seed-luckperms.sql` into the `luckperms` database, establishing the 5-tier rank hierarchy (`default`, `helper`, `moderator`, `administrator`, `owner`), prefixes, weights, and staff track (`staff`).

To promote the network owner, execute via console:
```bash
./bin/network console lobby
# Inside console:
lp user <owner-uuid> parent set owner
```
