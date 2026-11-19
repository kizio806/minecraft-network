# Workflow: Infrastructure Changes

## Purpose & Scope
Define the standard procedure for modifying Docker Compose service definitions, memory allocations, CPU quotas, network bridges, volume mounts, or Linux host configuration scripts.

## When to Use
Use this workflow whenever modifying `docker-compose.yml`, `.env.example`, network attachments, container resource ceilings, or host initialization scripts in `scripts/`.

## Steps

### Step 1: Architectural Assessment & Calculation
1. Calculate memory ceiling according to the invariant formula:
   $$\text{Container Limit} \ge 1.3 \times \text{JVM Max Heap (Xmx)}$$
2. Verify total combined container memory remains under 12 GB RAM on the 16 GB host.
3. Confirm that no backend ports are exposed to host interfaces (`0.0.0.0`).

### Step 2: Image Digest Pinning
1. Retrieve immutable `@sha256:` digest for any new or updated container image.
2. Update `docker-compose.yml` and `.env.example`.

### Step 3: Staging Verification
1. Validate Compose syntax:
   ```bash
   docker compose -f docker-compose.yml config --quiet
   docker compose -f docker-compose.yml --profile monitoring config --quiet
   ```
2. Restart modified services and verify container health:
   ```bash
   ./bin/network restart <service>
   ./bin/network health
   ```

### Step 4: Documentation Synchronization
1. Update `.agents/context/infrastructure.md` with new limits or networks.
2. Update `PROJECT.md` if container boundaries changed.

## Required Checks
- [ ] Docker Compose syntax validates cleanly for core and monitoring profiles.
- [ ] No backend ports exposed to public interfaces (`docker compose ps`).
- [ ] All containers report `healthy` in `./bin/network health`.
- [ ] Host memory budget remains $\le 12\text{ GB}$.

## Expected Result
Infrastructure modifications are deployed safely with immutable image digests, correct memory headroom, and zero security regressions.
