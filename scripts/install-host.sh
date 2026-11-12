#!/usr/bin/env bash
set -Eeuo pipefail

[[ $EUID -eq 0 ]] || { echo "Run as root on the target Ubuntu host." >&2; exit 1; }
source /etc/os-release
[[ ${ID:-} == ubuntu ]] || { echo "Only Ubuntu is supported: ${ID:-unknown}" >&2; exit 1; }
[[ ${VERSION_ID%%.*} -ge 22 ]] || { echo "Ubuntu 22.04 LTS or newer is required." >&2; exit 1; }

network_user="${NETWORK_USER:-minecraft}"
install -d -m 0755 "/opt/$network_user"
if ! id "$network_user" >/dev/null 2>&1; then
  useradd --system --create-home --home-dir "/opt/$network_user" --shell /usr/sbin/nologin "$network_user"
fi

apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl openssl git jq shellcheck ufw fail2ban
if ! command -v docker >/dev/null; then
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc
  printf 'Types: deb\nURIs: https://download.docker.com/linux/ubuntu\nSuites: %s\nComponents: stable\nArchitectures: %s\nSigned-By: /etc/apt/keyrings/docker.asc\n' "$VERSION_CODENAME" "$(dpkg --print-architecture)" > /etc/apt/sources.list.d/docker.sources
  apt-get update
  DEBIAN_FRONTEND=noninteractive apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
fi

install -d -o "$network_user" -g "$network_user" -m 0750 "/opt/$network_user"
echo "Host prepared. Existing SSH configuration and firewall rules were not removed."
echo "Before adding a firewall rule, verify the active SSH port with: ss -ltnp | grep ssh"
