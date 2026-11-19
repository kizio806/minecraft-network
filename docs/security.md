# Security Architecture & Threat Model

This document outlines the security controls, access models, and operational policies governing the **Minecraft Network** control plane and runtime containers.

---

## 1. Host and Container Hardening

- **Unprivileged Runtime Execution:** Application containers run under non-root UID `1000:1000` with `security_opt: [no-new-privileges:true]`.
- **System Call Protection:** Kernel privilege escalation is disabled across all Compose services.
- **Strict Host Firewall (UFW):**
  - Default incoming policy: `DENY`.
  - Allowed incoming ports: Verified administrative SSH port and TCP `25565` (plus UDP `25565` for Simple Voice Chat).
  - All database, telemetry, and backend ports are blocked from the host network interfaces.

---

## 2. Secrets Management & Protection

1. **Zero Secret Git Commits:** Passwords, tokens, and HMAC keys reside strictly in `./secrets/` with mode `0600` (directory `0700`).
2. **Container Injection via Docker Secrets:** Credentials are mounted as in-memory files under `/run/secrets/*`. Plaintext passwords must never be passed via Docker Compose environment strings.
3. **Age Asymmetric Backup Cryptography:** Backups are encrypted at rest with `age` public recipient keys. Decryption keys are stored offsite and never retained on the production server.

---

## 3. Hybrid Authentication Model (Strict Mixed-Mode)

- **VeloAuth Policy Engine:** Velocity acts as the sole gatekeeper for player authentication.
- **Mojang Session Verification:** Official Premium accounts are verified directly against Mojang session servers, preserving official UUIDs and skins.
- **Cracked Impersonation Prevention (`allow-cracked-on-premium-nicks: false`):** Non-Premium clients cannot join using nicknames registered in Mojang's official registry.
- **NanoLimbo Holding Pen:** Unauthenticated players are confined to an isolated limbo buffer (NanoLimbo) where chat, world interaction, and backend commands are disabled until authentication succeeds.
- **BCrypt Password Hashing:** Non-Premium credentials are encrypted with BCrypt and stored in the dedicated `veloauth` PostgreSQL database.

---

## 4. Threat Matrix & Mitigations

| Threat Vector | Mitigation Strategy |
|---|---|
| Premium nickname hijacking | VeloAuth strict session validation; `allow-cracked-on-premium-nicks: false` |
| Non-Premium account brute-force | Rate-limited login attempts, BCrypt hashing, IP admission throttling |
| Direct backend bypass | Backend servers bind only to private `proxy_net`; Modern Forwarding HMAC validation |
| Forwarding signature spoofing | Unique HMAC-SHA256 secret mounted from `/run/secrets/forwarding_secret` |
| Database unauthorized access | PostgreSQL isolated in internal `data_net`; role isolation per database service |
| Malicious plugin injection | Declarative supply chain (`manifest.yml` + `lock.yml`) with SHA-512 cryptographic verification |
| Backup tampering / theft | Asymmetric `age` encryption with offline identity keys and SHA-256 integrity digests |
