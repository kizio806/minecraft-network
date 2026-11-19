# ADR 0001 - Hybrid Premium and Non-Premium Authentication

## Status
Accepted

## Superseded By
None

## Date
2026-11-12

## Decision
Use a single Velocity-native VeloAuth installation in strict mixed mode backed by a private NanoLimbo void holding server. Velocity runs with `online-mode=false`, while VeloAuth enforces official Mojang handshakes for positively identified Premium accounts. Backend servers operate in offline mode and accept traffic exclusively via Velocity Modern Forwarding over the internal `proxy_net`.

VeloAuth owns registration, password hashing (BCrypt), session validation, and Premium detection. NanoLimbo serves solely as an unauthenticated void holding environment with zero terrain generation.

## Reason
The server requires seamless access for both Premium (official Mojang accounts with automatic login) and Non-Premium players without exposing cracked-mode account takeovers or requiring multiple competing authentication plugins (e.g. AuthMe + FastLogin).

## Alternatives
- **AuthMeReloaded on Paper backend:** Rejected because it requires legacy BungeeCord forwarding or complex bridge plugins, introducing security risks and login latency.
- **FastLogin + AuthMeVelocity:** Rejected due to conflicting Premium detection logic and reliance on legacy IP forwarding without HMAC validation.
- **Pure offline mode without Premium verification:** Rejected because it allows arbitrary account hijacking of legitimate Mojang usernames.

## Impact
- VeloAuth enforces `allow-cracked-on-premium-nicks=false`, preventing offline clients from impersonating official Mojang account owners.
- Unauthenticated players remain confined to NanoLimbo until authorized.
- Single source of truth for player authentication state.

## Verification
Connect to the proxy via Minecraft client:
- A registered Premium account must be authenticated via Mojang session servers without `/login`.
- A Non-Premium account must spawn in NanoLimbo and require `/register` or `/login`.
- Verify proxy health:
  ```bash
  ./bin/network health
  ```
