# Workflow: Adding a Plugin

## Purpose & Scope
Define the standard lifecycle for evaluating, declaring, locking, configuring, and verifying a new Minecraft plugin across proxy or backend server instances.

## When to Use
Use this workflow whenever introducing a new plugin to `configs/plugins/manifest.yml` or updating an existing plugin to a new version.

## Steps

### Step 1: Vetting & Compatibility
1. **Search Upstream:** Locate the plugin on Modrinth or GitHub Releases.
2. **Verify Compatibility:** Check active maintenance, Java 25 compatibility, and open-source licensing (MIT, Apache-2.0, GPL-3.0, LGPL, BSD).
3. **Assess Resource Footprint:** Ensure the plugin does not duplicate existing functionality or introduce banned dependencies (e.g. Redis).
4. **Obtain Approval:** Present the proposal (slug, version, license, target servers) and wait for authorization.

### Step 2: Manifest & Lock Declaration
1. Add plugin entry to `configs/plugins/manifest.yml`:
   ```yaml
   - project: example-plugin
     version: 1.2.3
     version-id: abc123def
     platform: paper
     servers: [survival]
     license: MIT
   ```
2. Resolve the download URL, exact filename, and SHA-512 cryptographic hash.
3. Record the resolved entry in `configs/plugins/lock.yml`.

### Step 3: Configuration Templating
1. Create source configuration template in `configs/<server>/plugins/<plugin>/config.yml`.
2. Use template placeholders (`__POSTGRES_PASSWORD__`, `__FORWARDING_SECRET__`) for sensitive credentials.

### Step 4: Installation & Deployment
1. Validate manifest and lockfile:
   ```bash
   ./bin/network plugins validate
   ```
2. Install the plugin JAR and render configuration files:
   ```bash
   ./bin/network install
   ```
3. Restart the target service:
   ```bash
   ./bin/network restart <service>
   ```

## Required Checks
- [ ] Manifest and lockfile validation passes: `./bin/network plugins validate`.
- [ ] No untracked JAR files remain in `servers/*/plugins/` outside `lock.yml`.
- [ ] Service container logs verify clean startup: `./bin/network logs <service> --tail 50`.
- [ ] System healthcheck confirms all services healthy: `./bin/network health`.

## Expected Result
The new plugin is deterministically downloaded, verified via SHA-512 checksum, rendered with secure configuration templates, and cleanly loaded into the running server container without downtime or regression.
