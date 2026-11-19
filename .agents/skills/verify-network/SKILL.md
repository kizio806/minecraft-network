---
name: verify-network
description: Run the full health, doctor, compose, and unit test verification suite on Minecraft Network.
---

# Skill: Network Verification

## Use Case
Execute this skill as the mandatory quality gate before concluding any development task, committing code, or declaring an issue resolved.

## Inputs & Prerequisites
- Docker daemon and Docker Compose v2 active.
- Compiled `./bin/network` binary.
- Access to run containers via Docker.

## Procedure

### 1. Execute System Doctor Audit
```bash
./bin/network doctor
```
Verifies Docker, Compose, manifest, lockfile, `.env`, and secret permissions.

### 2. Verify Container Runtime Health
```bash
./bin/network health
```
Verifies all active containers are running and in `healthy` state.

### 3. Validate Docker Compose Syntax
```bash
docker compose -f docker-compose.yml config --quiet
docker compose -f docker-compose.yml --profile monitoring config --quiet
```

### 4. Run Go Test Suite with Race Detection
```bash
docker run --rm -v "$(pwd)":/app -w /app golang:1.24 go test -v -race ./...
```

### 5. Validate Plugin Manifest Integrity
```bash
./bin/network plugins validate
```

## Validation
- All doctor components report `OK`.
- All running containers report `healthy`.
- Compose syntax exits with code 0.
- All Go unit tests pass with zero race conditions.
- Manifest and lockfile are 100% synchronized.

## Limitations & Edge Cases
- Requires network connectivity if Go modules or container images need to be pulled.
- Does not test Minecraft in-game player connectivity (use manual client connection for gameplay verification).
