# Native Go CLI architecture

`network` is the primary administrative interface. Operational business logic is implemented in Go; Bash is retained only for host bootstrap and firewall preparation.

```text
network
  ├── configuration and validation
  ├── approved Docker Compose executor
  ├── lifecycle, status, logs and health
  ├── native archive backup and restore
  ├── native update orchestration
  └── existing internal/pluginmanager package
```

## Safety contract

- Commands use `exec.CommandContext`; no shell wrapper is used for ordinary orchestration.
- Compose service names are allowlisted and the Compose file is validated before mutation.
- Mutating operations use an advisory operation lock.
- Restore requires `--yes`; update requires `--yes` or `--dry-run`.
- Backups are checksum-protected, staged atomically and never remove Docker volumes.
- Plugin operations call the existing Plugin Manager package and do not create another manifest or lockfile.
- JSON output is reserved for command data; errors go to stderr.
- Exit codes: `0` success, `2` usage, `3` configuration/precondition, `4` local dependency/lock, `5` service failure, `8` backup/restore failure.

## Commands

`network doctor`, `config validate|show`, `install`, `start|stop|restart [service]`, `status`, `logs [service]`, `health`, `backup [list]`, `restore <backup> --yes`, `update check|--dry-run|--apply`, and `plugins` commands.

The binary accepts `--project-dir`, `--output table|json`, `--yes`, `--dry-run`, `--follow` and `--lines`. Go is not required on the production host after installation.

`network update check` reports locked plugin state, configured image references and comparison with `.env.example`. It explicitly reports that upstream registry inspection is not implemented; it never pulls or deploys images. Status and health JSON use `schema_version: 1`, typed `services` rows and `healthy` for health; unknown or missing service states are failures, never healthy.
