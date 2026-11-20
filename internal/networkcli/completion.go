package networkcli

import (
	"fmt"
	"strings"
)

const bashCompletionScript = `# bash completion for network CLI
_network_completion() {
    local cur prev words cword
    _init_completion -n : || return

    local commands="init doctor config install start stop restart console status logs health backup restore update plugins completion"
    local global_flags="--project-dir --output --yes --dry-run --follow --monitoring --lines --help --version"
    local all_services="proxy auth lobby survival postgres discord-bot node-exporter prometheus grafana uptime-kuma minecraft-exporter"
    local console_services="proxy lobby survival"

    local cmd=""
    local i=1
    while [[ $i -lt $cword ]]; do
        local w="${words[i]}"
        case "$w" in
            --project-dir|--output|--lines)
                ((i++))
                ;;
            --*)
                ;;
            *)
                cmd="$w"
                break
                ;;
        esac
        ((i++))
    done

    if [[ -z "$cmd" ]]; then
        if [[ "$cur" == -* ]]; then
            COMPREPLY=($(compgen -W "$global_flags" -- "$cur"))
        else
            COMPREPLY=($(compgen -W "$commands" -- "$cur"))
        fi
        return 0
    fi

    case "$cmd" in
        config)
            COMPREPLY=($(compgen -W "validate show" -- "$cur"))
            ;;
        backup)
            COMPREPLY=($(compgen -W "list" -- "$cur"))
            ;;
        restore)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=($(compgen -W "--yes" -- "$cur"))
            else
                COMPREPLY=($(compgen -f -X '!*.tar.gz.age' -- "$cur"))
            fi
            ;;
        update)
            COMPREPLY=($(compgen -W "check --dry-run --apply" -- "$cur"))
            ;;
        plugins)
            COMPREPLY=($(compgen -W "validate list plan verify sync components rollback --dry-run" -- "$cur"))
            ;;
        completion)
            COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur"))
            ;;
        console)
            COMPREPLY=($(compgen -W "$console_services" -- "$cur"))
            ;;
        start|stop|restart)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=($(compgen -W "--monitoring" -- "$cur"))
            else
                COMPREPLY=($(compgen -W "$all_services" -- "$cur"))
            fi
            ;;
        logs)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=($(compgen -W "--follow --lines" -- "$cur"))
            else
                COMPREPLY=($(compgen -W "$all_services" -- "$cur"))
            fi
            ;;
        init)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=($(compgen -W "--non-interactive -y --yes --name --domain --locale --theme --max-players" -- "$cur"))
            fi
            ;;
    esac
}

complete -F _network_completion network
complete -F _network_completion ./bin/network
`

const zshCompletionScript = `#compdef network ./bin/network

_network() {
    local -a commands
    commands=(
        'init:Initialize network configuration via interactive TUI wizard'
        'doctor:Execute comprehensive environment and permission diagnostics'
        'config:Validate or inspect active network configuration'
        'install:Synchronize plugins, render templates, and generate secrets'
        'start:Start containerized network services'
        'stop:Stop running network services'
        'restart:Restart network services with re-rendered templates'
        'console:Attach interactive console to a server'
        'status:Display real-time container status and resource usage'
        'logs:Tail service logs'
        'health:Verify network readiness and service ports'
        'backup:Create or list encrypted age backups'
        'restore:Restore network state from an age-encrypted backup'
        'update:Check or apply updates for pinned dependencies'
        'plugins:Manage declarative plugin supply chain'
        'completion:Generate shell autocompletion script'
    )

    local -a console_services
    console_services=('proxy' 'lobby' 'survival')

    local -a all_services
    all_services=('proxy' 'auth' 'lobby' 'survival' 'postgres' 'discord-bot' 'node-exporter' 'prometheus' 'grafana' 'uptime-kuma' 'minecraft-exporter')

    _arguments -C \
        '--project-dir[Specify custom project root path]:path:_files -/' \
        '--output[Output format]:format:(table json)' \
        '--yes[Bypass confirmation prompts]' \
        '--dry-run[Simulate operations without writing]' \
        '--follow[Follow log streaming output]' \
        '--monitoring[Include monitoring stack services]' \
        '--lines[Number of log lines to inspect]:number:' \
        '1: :->command' \
        '*:: :->args'

    case $state in
        command)
            _describe -t commands 'network commands' commands
            ;;
        args)
            case $line[1] in
                console)
                    _values 'console service' $console_services
                    ;;
                start|stop|restart)
                    _values 'service' $all_services
                    ;;
                logs)
                    _values 'service' $all_services
                    ;;
                config)
                    _values 'config command' 'validate' 'show'
                    ;;
                backup)
                    _values 'backup command' 'list'
                    ;;
                restore)
                    _files -g '*.tar.gz.age'
                    ;;
                update)
                    _values 'update command' 'check' '--dry-run' '--apply'
                    ;;
                plugins)
                    _values 'plugins command' 'validate' 'list' 'plan' 'verify' 'sync' 'components' 'rollback'
                    ;;
                completion)
                    _values 'shell' 'bash' 'zsh' 'fish'
                    ;;
            esac
            ;;
    esac
}

_network "$@"
`

const fishCompletionScript = `# fish completion for network CLI

set -l commands init doctor config install start stop restart console status logs health backup restore update plugins completion
set -l all_services proxy auth lobby survival postgres discord-bot node-exporter prometheus grafana uptime-kuma minecraft-exporter
set -l console_services proxy lobby survival

complete -c network -l project-dir -d "Project root directory" -r
complete -c network -l output -d "Output format" -xa "table json"
complete -c network -l yes -d "Bypass interactive confirmation"
complete -c network -l dry-run -d "Simulate execution without modifications"
complete -c network -l follow -d "Follow log streaming output"
complete -c network -l monitoring -d "Include monitoring services"
complete -c network -l lines -d "Number of log lines to tail" -r

complete -c network -n "not __fish_seen_subcommand_from $commands" -a init -d "Initialize network configuration"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a doctor -d "Run system diagnostic audit"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a config -d "Validate or inspect configuration"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a install -d "Synchronize plugins and render configs"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a start -d "Launch network services"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a stop -d "Stop network services"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a restart -d "Restart network services"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a console -d "Attach to server console"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a status -d "Show container health and status"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a logs -d "Tail container logs"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a health -d "Verify service readiness"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a backup -d "Create encrypted backup"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a restore -d "Restore from encrypted backup"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a update -d "Check or apply updates"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a plugins -d "Plugin supply chain operations"
complete -c network -n "not __fish_seen_subcommand_from $commands" -a completion -d "Generate shell completion script"

complete -c network -n "__fish_seen_subcommand_from config" -a "validate show"
complete -c network -n "__fish_seen_subcommand_from backup" -a "list"
complete -c network -n "__fish_seen_subcommand_from update" -a "check --dry-run --apply"
complete -c network -n "__fish_seen_subcommand_from plugins" -a "validate list plan verify sync components rollback"
complete -c network -n "__fish_seen_subcommand_from completion" -a "bash zsh fish"
complete -c network -n "__fish_seen_subcommand_from console" -a "$console_services"
complete -c network -n "__fish_seen_subcommand_from start stop restart logs" -a "$all_services"

complete -c ./bin/network -w network
`

func (a App) completion(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(a.Out, `Usage: network completion [bash|zsh|fish]

Generate shell autocompletion script for bash, zsh, or fish.

Installation:
  Bash:
    source <(./bin/network completion bash)
    # Persist:
    ./bin/network completion bash | sudo tee /etc/bash_completion.d/network > /dev/null

  Zsh:
    source <(./bin/network completion zsh)
    # Persist:
    ./bin/network completion zsh > "${fpath[1]}/_network"

  Fish:
    ./bin/network completion fish | source
    # Persist:
    ./bin/network completion fish > ~/.config/fish/completions/network.fish`)
		return nil
	}

	shell := strings.ToLower(args[0])
	switch shell {
	case "bash":
		fmt.Fprint(a.Out, bashCompletionScript)
		return nil
	case "zsh":
		fmt.Fprint(a.Out, zshCompletionScript)
		return nil
	case "fish":
		fmt.Fprint(a.Out, fishCompletionScript)
		return nil
	default:
		return &ExitError{2, fmt.Errorf("unsupported shell %q (supported: bash, zsh, fish)", shell)}
	}
}
