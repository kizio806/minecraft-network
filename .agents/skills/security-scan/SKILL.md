---
name: security-scan
description: Execute an automated security audit checking file permissions, secret leakage, image pinning, and network isolation.
---

# Skill: Security Scan

## Use Case
Execute this skill before committing code, after updating Docker Compose, or during security compliance audits.

## Inputs & Prerequisites
- Linux shell with `stat`, `grep`, and `docker` commands.
- Read access to repository root, `.env`, and `./secrets/`.

## Procedure

### 1. Verify Secret Directory and File Permissions
```bash
stat -c "%a %n" secrets/
stat -c "%a %n" secrets/*
```
Remediate immediately if permissions deviate:
```bash
chmod 700 secrets
chmod 600 secrets/*
```

### 2. Verify Port Isolation
```bash
docker compose ps --format "table {{.Service}}\t{{.Ports}}"
```
Verify that ONLY `proxy` publishes to `0.0.0.0:25565`. Backend services (`auth`, `lobby`, `survival`, `postgres`) must NOT have host bindings.

### 3. Verify Docker Image Pinning
```bash
grep -n "image:" docker-compose.yml
```
Confirm every image contains `@sha256:` immutable digest hashes.

### 4. Audit Untracked Sensitive Files
```bash
git status -s
```
Confirm no `.env`, secret files, or private keys are untracked or committed.

### 5. Verify Non-Root Execution
```bash
grep -n "user:" docker-compose.yml
```

## Validation
- `./bin/network doctor` exits with `secrets OK`.
- Only `proxy` binds port `25565` to `0.0.0.0`.
- All container images use `@sha256:` digests.

## Limitations & Edge Cases
- Does not inspect external firewall rules on the physical host (use `scripts/configure-firewall.sh` for host iptables/ufw setup).
