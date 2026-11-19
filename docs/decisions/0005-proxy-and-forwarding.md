# ADR 0005 - Velocity Modern Forwarding

## Status
Accepted

## Superseded By
None

## Date
2026-11-12

## Decision
Deploy Velocity 4.3.1 as the single public reverse proxy endpoint. Authenticate and forward player connections to backend servers (`auth`, `lobby`, `survival`) using Velocity Modern Forwarding secured with a cryptographically generated shared HMAC secret (`forwarding_secret`). Legacy BungeeCord forwarding is strictly prohibited.

## Reason
Legacy BungeeCord IP forwarding relies on easily spoofed header data and lacks cryptographic verification. If a backend port were ever accidentally reached or misconfigured, legacy forwarding allows attackers to spoof any player UUID or operator privileges. Velocity Modern Forwarding cryptographically signs every login packet using the shared secret, ensuring backends accept connections only from the authentic proxy.

## Alternatives
- **BungeeCord / Waterfall:** Rejected due to lack of native modern forwarding, inferior performance, higher memory footprint, and legacy architecture.
- **Legacy BungeeCord Forwarding:** Rejected due to lack of packet integrity and authentication spoofing vulnerabilities.

## Impact
- `proxy` is the sole service bound to host port `25565`.
- Backend servers enable `settings.velocity-support` in `paper-global.yml` and match the shared `forwarding_secret`.
- Unauthenticated packet forwarding attempts are rejected by backends.

## Verification
Inspect proxy configuration and backend connection verification:
```bash
./bin/network doctor
```
Verifies that `forwarding_secret` exists with mode `0600` and backend configurations are valid.
