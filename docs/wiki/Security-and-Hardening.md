# Security & Hardening Guide

Security policies, container isolation standards, and network hardening guidelines for **Minecraft Network**.

---

## 🛡️ Ingress & Proxy Hardening

- **Velocity Modern Forwarding:** Prevents UUID spoofing and IP bypasses by cryptographically signing connection metadata sent from Velocity to backend servers.
- **Firewall & Binding:** Backend game servers listen exclusively on internal Docker IPs (`172.x.x.x`). Host firewall (UFW/iptables) rejects external traffic targeting backend ports.
- **Rate Limiting & Anti-DDoS:** Ingress rate limiting on TCP connection handshakes prevents SYN flood and packet spam attacks.

---

## 🔐 Container Least Privilege

- Server processes execute under non-root system users (`UID 1000`).
- Container capabilities are dropped (`cap_drop: [ALL]`, adding only `NET_BIND_SERVICE` where required).
- Read-only container root filesystems with ephemeral volumes for `/tmp`.
