# ADR 0003: Hybrid player identity strategy

## Status

Accepted

## Decision

Premium identity is the Mojang UUID after a verified Premium handshake. Non-Premium identity is the deterministic offline UUID associated with the account record. Strict mixed mode reserves Premium nicknames and rejects an offline client attempting to claim an unowned Premium nickname.

VeloAuth maintains the Premium UUID separately from historical offline identity during supported migrations. A username rename is reconciled by UUID rather than creating a new account.

## Migration

Before enabling authentication in production, copy the auth database and all backend player data into staging. Inventory existing offline accounts, map permissions/economy/plugin data by UUID, and test the exact plugin set. Do not bulk-rewrite UUIDs in production. A Premium migration requires a restorable backup, a conflict report and an explicit operator decision for every collision.

## Trade-off

Premium and Non-Premium identities cannot share one UUID namespace without migration risk. Strict mode prioritizes ownership protection and fail-closed Premium verification over allowing an offline player to reserve every possible Premium name.
