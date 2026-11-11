#!/usr/bin/env bash
set -Eeuo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"
docker compose -f docker/docker-compose.yml pull
echo "Images pulled. Review the change and restart during a maintenance window."

