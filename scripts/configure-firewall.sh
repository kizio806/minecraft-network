#!/usr/bin/env bash
set -Eeuo pipefail

[[ $EUID -eq 0 ]] || { echo "Run as root." >&2; exit 1; }
command -v ufw >/dev/null || { echo "ufw is required." >&2; exit 1; }
ssh_port="${SSH_PORT:-$(ss -H -ltnp 2>/dev/null | sed -n 's/.*:\([0-9][0-9]*\).*sshd.*/\1/p' | head -n1)}"
[[ $ssh_port =~ ^[0-9]+$ ]] || { echo "Could not determine the active SSH port; set SSH_PORT explicitly." >&2; exit 1; }
ufw allow "$ssh_port/tcp" comment 'preserve current SSH access'
ufw allow 25565/tcp comment 'Minecraft Velocity'
if [[ ${ENABLE_UFW:-0} == 1 ]]; then
  ufw default deny incoming
  ufw default allow outgoing
  ufw --force enable
else
  echo "Rules added but UFW was not enabled. Review with 'ufw status numbered', then rerun ENABLE_UFW=1 $0."
fi
