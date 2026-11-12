# Plugin configuration

Plugin JARs are intentionally not included. The selected authentication dependency is recorded in [manifest.yml](manifest.yml): VeloAuth 1.6.2, Java 25, Velocity 4.2.x and Minecraft 1.21.11. Lobby uses FancyNpcs 2.12.1 with a pinned Modrinth version ID and SHA-256/SHA-512 checksums. Citizens and ProtocolLib are deliberately absent from the active set. Each approved plugin must have a pinned version, source URL, checksum, compatibility note and rollback procedure.

`network plugins sync --apply` refuses to install an unreviewed or mutable artifact. The Go Plugin Manager validates the manifest, lockfile, compatibility, checksum and JAR descriptor before publication.

