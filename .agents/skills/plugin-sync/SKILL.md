---
name: plugin-sync
description: Safely download, cryptographically verify, and synchronize Minecraft plugins without downtime risks.
---

# Skill: Plugin Synchronization

## Use Case
Execute this skill when introducing, upgrading, or auditing plugins across proxy and backend game servers.

## Inputs & Prerequisites
- Valid entries in `configs/plugins/manifest.yml` and `configs/plugins/lock.yml`.
- Internet connectivity to Modrinth / GitHub Releases for downloading new plugins.
- Compiled `./bin/network` binary.

## Procedure

### 1. Validate Manifest and Lockfile Integrity
```bash
./bin/network plugins validate
```

### 2. Synchronize Plugins and Render Configs
```bash
./bin/network install
```
This downloads missing JARs, verifies SHA-512 hashes, resolves template placeholders, and copies rendered configs into `servers/*/plugins/`.

### 3. Restart Targeted Services
```bash
./bin/network restart <service>
```

### 4. Verify Server Logs & Plugin State
```bash
./bin/network logs <service> --tail 50
./bin/network health
```

## Validation
- Plugin validation exits with code 0.
- All plugin JARs in `servers/*/plugins/` match SHA-512 checksums in `lock.yml`.
- Server logs indicate plugins loaded successfully with 0 exceptions.

## Limitations & Edge Cases
- Do not run `plugin-sync` while target servers are under heavy player load; restart during quiet periods.
- Failed checksum downloads are automatically aborted without touching live files.
