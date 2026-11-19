# ADR 0003 - Hybrid Player Identity and UUID Strategy

## Status
Accepted

## Superseded By
None

## Date
2026-11-12

## Decision
Premium identity uses the official Mojang UUID after an authenticated Mojang handshake. Non-Premium identity uses the deterministic offline UUID (UUID v3 calculated from username) associated with the local VeloAuth account record. Strict mixed mode reserves Premium usernames and rejects any offline client attempting to register or claim a username owned by an official Mojang account.

## Reason
In a hybrid Minecraft network, allowing offline players to register arbitrary names allows malicious users to claim official administrator or content creator usernames. Furthermore, Minecraft plugins (LuckPerms, CoreProtect) track player data by UUID; using genuine Mojang UUIDs for Premium players ensures inventory and permission persistence across potential future network transitions.

## Alternatives
- **Universal Offline UUIDs for all players:** Rejected because Premium players lose their global skin bindings, Mojang identity verification, and third-party API integration.
- **Permissive mixed mode:** Rejected because it allows an offline player to register a name before the legitimate Premium owner joins, resulting in account takeover.

## Impact
- VeloAuth maintains separate namespaces: official Mojang UUIDs for Premium and deterministic offline UUIDs for Non-Premium.
- Player username renames for Premium accounts are tracked cleanly by Mojang UUID.
- Permissions and statistics in LuckPerms and CoreProtect remain persistent and distinct per UUID.

## Verification
Inspect player UUID mapping inside LuckPerms and VeloAuth:
```bash
./bin/network console survival
# Within console:
lp user <username> info
```
Verify that Premium accounts display their genuine Mojang UUID.
