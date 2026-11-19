---
name: console-inspect
description: Safely inspect interactive server consoles, stream real-time logs, and monitor JVM resource metrics across containers.
---

# Skill: Console & Resource Inspection

## Use Case
Execute this skill when diagnosing server startup failures, inspecting in-game errors, monitoring memory saturation, or streaming live server logs.

## Inputs & Prerequisites
- Docker daemon running with active Minecraft containers.
- Compiled `./bin/network` binary.
- Target service name: `proxy`, `survival`, `lobby`, `auth`, or `postgres`.

## Procedure

### 1. Stream Live Container Logs
```bash
./bin/network logs <service> --follow --lines 50
```

### 2. Attach Interactive TTY Console
```bash
./bin/network console <service>
```
Supported services: `survival`, `lobby`, `proxy`.
*To safely detach without stopping the container, press `Ctrl+P, Ctrl+Q`.*

### 3. Inspect Memory & CPU Saturation
```bash
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}"
```

### 4. Inspect Container Health Details
```bash
docker inspect --format='{{json .State.Health}}' <container_name> | jq
```

## Validation
- Log stream reflects real-time server activity.
- Console attaches without error and responds to non-destructive commands (e.g. `/version`).
- Memory usage is within container limits (Survival < 5632M, Lobby < 1536M, Proxy < 768M).

## Limitations & Edge Cases
- **NEVER** issue `/stop` directly inside the interactive console; use `./bin/network stop` to ensure proper shutdown.
- Do not run CPU-intensive commands from the console while players are active.
