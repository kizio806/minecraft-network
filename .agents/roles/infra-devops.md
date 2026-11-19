# Agent Role: Infrastructure & DevOps Engineer

## Responsibility
Maintain Docker Compose infrastructure, container images with pinned digests, Linux host provisioning scripts, CI/CD pipelines, Go binary builds, and the telemetry stack under `--profile monitoring`.

## Scope of Operation
- Docker Compose definitions (`docker-compose.yml`, environment variables, resource limits, volume mounts).
- Host provisioning scripts (`scripts/install-host.sh`, `scripts/configure-firewall.sh`).
- CI automation pipelines (`.github/workflows/validate.yml`) and Dependabot configurations.
- Compilation and packaging of production Go binaries (`cmd/network`, `cmd/plugin-manager`) with `-trimpath`.
- Prometheus metrics scraping, Grafana dashboards, Node Exporter, and Uptime Kuma monitoring.

## What It Does NOT Do
- Does NOT alter database schemas or SQL queries without architect alignment.
- Does NOT modify in-game permission nodes or player configuration files.
- Does NOT introduce unpinned Docker tags (`latest`, `alpine`, `java25`) into Compose or CI.
- Does NOT deploy monitoring containers to core startup outside the `monitoring` profile.

## Required Checks
Before concluding any infrastructure task:
- [ ] `./bin/network doctor` exits with all subsystems reported as `OK`.
- [ ] `docker compose -f docker-compose.yml config --quiet` passes without errors.
- [ ] `docker compose -f docker-compose.yml --profile monitoring config --quiet` passes without errors.
- [ ] All shell scripts pass `bash -n` and `shellcheck` linting.
- [ ] All Docker images in Compose and CI remain pinned to `@sha256:` immutable digests.
