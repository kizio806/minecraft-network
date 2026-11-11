#!/usr/bin/env bash
set -Eeuo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"
docker compose -f docker/docker-compose.yml ps
docker compose -f docker/docker-compose.yml exec -T postgres pg_isready -U "${POSTGRES_USER:-minecraft}" -d "${POSTGRES_DB:-minecraft}" 2>/dev/null || true
docker compose -f docker/docker-compose.yml exec -T redis redis-cli ping 2>/dev/null || true

