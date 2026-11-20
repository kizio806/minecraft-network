# Lobby Bootstrap & Administration

The repository automatically provisions LuckPerms groups, permissions, tracks, and WorldGuard regions via declarative SQL seeding (`seed-luckperms.sql`) and configuration templates during installation.

---

## 1. Automated Ranks & Permissions

The PostgreSQL `luckperms` database is automatically seeded on first launch with the 5-tier clean rank hierarchy:
`default -> helper -> moderator -> administrator -> owner`

### Assigning Owner Privileges
To promote the server owner, assign the `owner` group explicitly by player UUID (never by mutable username) via the server console:

```bash
./bin/network console lobby
# Inside console:
lp user <player-uuid> parent set owner
```

---

## 2. WorldGuard Region Protection

The lobby spawn protection is pre-configured via the declarative template in `configs/lobby/plugins/worldguard/worlds/lobby/regions.yml`. The `__global__` region automatically enforces:
- `invincible: allow` and `godmode: allow`
- `pvp: deny`, `mob-spawning: deny`, `hunger-drain: deny`
- `block-break: deny`, `block-place: deny`, `item-drop: deny`, `item-pickup: deny`
- `fall-damage: deny`
