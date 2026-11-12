# NanoLimbo auth/limbo runtime

This private backend is the holding server used by VeloAuth. It is not a password database and must never be exposed directly. VeloAuth owns Premium verification, registration, password hashing, session state and transfer to Lobby/Survival.

The runtime artifact is NanoLimbo `1.12.0`; no JAR is committed. See `configs/plugins/components.yml` and `configs/plugins/components.lock.yml`; stage it through the reviewed deployment process.
