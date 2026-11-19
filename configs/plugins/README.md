# Plugin configuration

Plugin JARs and datapack archives are intentionally not tracked directly in Git. All active plugins are declared in [manifest.yml](manifest.yml) and pinned in [lock.yml](lock.yml). Standalone components (NanoLimbo) and 100% server-side dimension and exploration datapacks (Structory, Structory: Towers, Dungeons and Taverns, Incendium, Nullscape) are declared in [components.yml](components.yml) and pinned in [components.lock.yml](components.lock.yml).

`network plugins sync --apply` refuses to install an unreviewed or mutable artifact. The Go Plugin Manager validates the manifest, lockfile, compatibility, checksum and JAR descriptor before publication.

