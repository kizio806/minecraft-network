#!/usr/bin/env bash
set -Eeuo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"
command -v docker >/dev/null || { echo "Docker is required" >&2; exit 1; }
test -f .env || cp .env.example .env
docker compose -f docker/docker-compose.yml config >/dev/null
echo "Configuration is valid. Review .env and security docs before starting."

