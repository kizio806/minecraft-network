# Security Context

## Purpose
Define zero-trust network segmentation, cryptographic secret handling, credential injection methods, filesystem permissions, and age-encrypted disaster recovery mechanisms.

## Scope
Applies to Docker network bridges, the secrets directory `./secrets/`, Velocity Modern Forwarding, PostgreSQL user isolation, and backup encryption.

## Rules / Information

### 1. Zero-Trust Network Topology

The network uses 3 segmented Docker networks:

1. **`proxy_net` (Bridge Network):**
   - Connects the public Velocity reverse proxy to internal backend Minecraft servers (`lobby`, `survival`, `auth`).
   - Only `proxy` binds to host port `0.0.0.0:25565:25565/tcp`.
   - Backend servers have zero host port mappings in `docker-compose.yml`.
2. **`data_net` (`internal: true`):**
   - Strict internal network without default gateway or internet route.
   - Connects `postgres` strictly to `proxy` (VeloAuth), `lobby` (LuckPerms), and `survival` (LuckPerms).
3. **`monitoring` (Internal Bridge):**
   - Connects Prometheus scraper to `node-exporter` and `minecraft-exporter`.
   - Web UIs (`grafana:3000`, `uptime-kuma:3001`) bind exclusively to loopback `127.0.0.1`.

### 2. Secret Registry & Filesystem Permissions

All secrets reside in `./secrets/` and are excluded from Git via `.gitignore`.
- **Directory Mode:** `0700` (`drwx------`)
- **File Mode:** `0600` (`-rw-------`)

| Secret File | Required Mode | Purpose & Usage |
| :--- | :---: | :--- |
| `postgres_password` | `0600` | PostgreSQL root `POSTGRES_PASSWORD_FILE`. |
| `luckperms_password` | `0600` | Password for PostgreSQL user `luckperms_user` on database `luckperms`. |
| `veloauth_password` | `0600` | Password for PostgreSQL user `veloauth_user` on database `veloauth`. |
| `forwarding_secret` | `0600` | Shared HMAC secret for Velocity Modern Forwarding between proxy and backends. |
| `grafana_admin_password` | `0600` | Initial admin credential for Grafana dashboard. |
| `backup.age-identity` | `0600` | Private X25519 age identity key for decrypting backups during restoration. |
| `backup.age-recipient` | `0600` | Public X25519 age recipient key for encrypting backups during creation. |

### 3. Runtime Secret Injection & Templating

- **Docker Secrets:** Used for container-native file mounting (`/run/secrets/*`).
- **Template Rendering:** Configuration templates stored in `configs/<server>/plugins/<plugin>/` contain placeholders (`__FORWARDING_SECRET__`, `__POSTGRES_PASSWORD__`, `__VELOAUTH_PASSWORD__`, `__LUCKPERMS_PASSWORD__`).
- The Go CLI `./bin/network install` reads secrets from `./secrets/`, interpolates placeholders into memory, and writes rendered configs into `servers/` (which are git-ignored).
- Passwords must NEVER be passed as plain text in `docker-compose.yml` `environment:` blocks.

### 4. Backup Encryption & Integrity

- Backups are compressed with `gzip` and encrypted with asymmetric `age` (X25519) in streaming mode.
- Output archives are written to `backups/backup-<timestamp>.tar.gz.age`.
- Restorations require `--yes` confirmation and decrypt using `backup.age-identity`.

## Verification
Validate secret permissions and system security:
```bash
# 1. Audit secret permissions and environment
./bin/network doctor

# 2. Verify port bindings (only proxy on 0.0.0.0:25565)
docker compose ps --format "table {{.Service}}\t{{.Ports}}"
```
The output of `doctor` must report `secrets OK` and `ports` must only show `0.0.0.0:25565` for `proxy`.
