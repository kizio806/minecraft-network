# Context: Network CLI Reference

## Purpose
Document the command-line interface, global flags, subcommands, and exit codes for the unified `./bin/network` binary.

## Scope
Applies to `./bin/network` and its interaction with Docker Compose, secret management, plugin synchronization, and disaster recovery.

## Rules / Information

### 1. Global Flags

| Flag | Argument | Description | Default |
| :--- | :--- | :--- | :--- |
| `--project-dir` | `PATH` | Absolute or relative path to the repository root directory. | Current working directory |
| `--output` | `table\|json`| Output formatting for tabular commands (`status`). | `table` |
| `--yes` | none | Automatic confirmation for destructive actions (`restore`, `update`). | `false` |
| `--dry-run` | none | Simulation mode without applying changes to disk or containers. | `false` |
| `--follow` | none | Stream live container log output (`logs`). | `false` |
| `--monitoring` | none | Include containers with `profiles: [monitoring]` during lifecycle operations. | `false` |
| `--lines` | `N` | Number of tail log lines to show (`logs`). | `100` |

### 2. Subcommands

| Command | Usage | Description |
| :--- | :--- | :--- |
| **`doctor`** | `network doctor` | Complete system audit (Docker, Compose, Manifest, Lock, Secrets, `.env`). |
| **`health`** | `network health` | Inspects live Docker container healthchecks. Fails if any service is unhealthy. |
| **`status`** | `network status [--output json]` | Lists all managed containers, states, and health. |
| **`start`** | `network start [--monitoring] [service]` | Generates runtime configs and starts containers (`docker compose up -d`). |
| **`stop`** | `network stop [--monitoring] [service]` | Stops containers safely. |
| **`restart`** | `network restart [--monitoring] [service]` | Regenerates runtime configs and restarts containers. |
| **`console`** | `network console <service>` | Attaches interactive TTY console to `proxy`, `lobby`, or `survival` (`Ctrl+P, Ctrl+Q` to detach). |
| **`logs`** | `network logs [service] [--follow] [--lines N]`| Reads container logs. |
| **`install`** | `network install` | Synchronizes plugins with SHA-512 check, generates secrets, renders configs. |
| **`backup`** | `network backup [list]` | Creates encrypted `age` archive of PostgreSQL databases and server files. |
| **`restore`** | `network restore <id\|path> --yes` | Decrypts `age` archive, verifies SHA-256, and restores state. |
| **`plugins`** | `network plugins validate\|sync\|report` | Interface to `plugin-manager`. |
| **`config`** | `network config validate\|show` | Validates configuration files against schemas. |
| **`update`** | `network update check\|--apply` | Checks for new image digests and creates pre-update backup. |

### 3. Exit Codes

- `0`: Success — command completed without errors.
- `2`: Validation error / invalid command-line arguments.
- `3`: Missing configuration, missing secret file, or invalid `.env`.
- `4`: Filesystem error, permission violation, or active operation lock contention (`.network-operation.lock`).
- `5`: Docker or Docker Compose execution error.

## Verification
Validate CLI functionality and version response:
```bash
./bin/network --help
./bin/network doctor
```
Both commands must execute and report exit code 0.
