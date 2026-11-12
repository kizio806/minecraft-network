# ADR 0001: Hybrid Premium and Non-Premium authentication

## Status

Accepted

## Decision

Use one Velocity-native VeloAuth installation in strict mixed mode with a private NanoLimbo auth/limbo backend. Velocity runs with `online-mode=false`, while VeloAuth can force the official Mojang handshake for identities it has positively resolved as Premium. Backend servers remain offline-mode and accept only modern Velocity forwarding over the private `proxy_net`.

VeloAuth owns registration, password verification, Premium resolution and session state; NanoLimbo only provides the private holding server. No separate cracked-mode Paper Auth backend or competing AuthMe/FastLogin stack is installed.

## Security model

VeloAuth is configured with `allow-cracked-on-premium-nicks=false`, so a name found in Mojang's premium namespace is not silently claimable by an offline client. Failed or unavailable Premium resolution must not be treated as an unrestricted offline login. Password accounts use the plugin's BCrypt implementation and the database registration operation is ownership-safe.

`bypass-auth-server=false` keeps every Java connection in the authentication flow until the plugin authorizes it. The Velocity forwarding secret remains separate from plugin/database secrets.

## Alternatives rejected

- AuthMe with `online-mode=true` authenticates Premium accounts but does not admit Non-Premium clients.
- AuthMeVelocity's offline-proxy mode requires legacy forwarding, which is not selected for this network because modern forwarding is required and compatible with VeloAuth.
- FastLogin is an integration component, not the single source of registration/session policy; adding it would create competing Premium detection paths.

## Operational constraint

VeloAuth 1.6.2 requires Java 25 and Velocity 4.2.x. NanoLimbo v1.12.0 supports Minecraft 1.21.11. Neither artifact is committed as a binary. Their source, release and compatibility data are recorded in the manifests and lockfiles; artifact checksums are verified by the native Go Plugin Manager before staging.
