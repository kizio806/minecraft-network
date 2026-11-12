# Deployment na Ubuntu Server LTS

1. Na czystym hoście uruchom jako root `scripts/install-host.sh`. Skrypt sprawdza Ubuntu, tworzy użytkownika i instaluje Docker; nie usuwa reguł SSH/firewalla.
2. Sprawdź aktywny port SSH (`ss -ltnp | grep ssh`) przed ręcznym dodaniem reguł UFW. Nie wykonuj zmian firewalla bez zachowania dostępu administracyjnego.
   Po weryfikacji można użyć `SSH_PORT=<port> ENABLE_UFW=1 ./scripts/configure-firewall.sh`; skrypt nie usuwa istniejących reguł.
3. Skopiuj repozytorium do `/opt/minecraft`, ustaw właściciela na użytkownika deploymentowego i przełącz się na niego.
4. Skopiuj `.env.example` do `.env`, ustaw `MC_UID`/`MC_GID` zgodnie z użytkownikiem deploymentowym oraz przypięte obrazy i wersje. Nie dodawaj haseł do Git; ustaw `.env` na `0600`.
5. Zbuduj CLI z wersją: `go build -trimpath -ldflags='-X github.com/kizio806/minecraft-network/internal/networkcli.Version=0.1.0' -o network ./cmd/network`. Zainstaluj sam executable przez `install -o root -g root -m 0755 network /usr/local/bin/network`; odinstalowanie usuwa wyłącznie `/usr/local/bin/network`.
6. Uruchom `network install`. Przygotowanie jest idempotentne i nie nadpisuje istniejących sekretów.
7. Uruchom `network plugins sync --dry-run`, zaakceptuj plan, następnie `network plugins sync --apply`, `network start` i `network health`. JAR-y pluginów są zarządzane przez Plugin Manager i nie są przechowywane w Git.
   Auth backend uruchamia dokładnie zweryfikowany NanoLimbo `1.12.0` z lokalnego pliku; nie pobiera zależności podczas startu.
7. Skonfiguruj lokalny dostęp do Grafany/Uptime Kuma oraz monitory proxy/backendów.
8. Przed produkcją wykonaj backup i test restore na izolowanym środowisku.

Pierwszy deployment wykonuj testowo. Nie otwieraj portów baz danych ani paneli monitoringu do Internetu bez osobnej decyzji bezpieczeństwa.

## Local staging on Arch Linux

Local development is performed on Arch Linux and must not run the Ubuntu host
bootstrap scripts. Use a separate staging copy outside the repository, separate
mode-0600 secrets and the fixed Compose project name
`minecraft-network-staging`. Published test ports must bind to `127.0.0.1`.

The staging sequence is:

```bash
go run ./cmd/plugin-manager --manifest /path/to/staging/configs/plugins/manifest.yml \
  --lock /path/to/staging/configs/plugins/lock.yml sync --apply
go run ./cmd/plugin-manager --manifest /path/to/staging/configs/plugins/manifest.yml \
  --lock /path/to/staging/configs/plugins/lock.yml components --apply
docker compose --project-name minecraft-network-staging \
  --env-file /path/to/staging/.env -f /path/to/staging/docker-compose.yml \
  up -d postgres auth lobby survival proxy
```

Before starting, verify the exact project name, staging-only paths and that no
production secret is loaded. Cleanup must use the same explicit project:

```bash
docker compose --project-name minecraft-network-staging \
  --env-file /path/to/staging/.env -f /path/to/staging/docker-compose.yml \
  down -v --remove-orphans
```

The first local run created the expected staging resources, but runtime
verification stopped before Velocity startup because `/var/lib/docker` had no
free space and PostgreSQL failed with `No space left on device`. The staging
resources were removed. This is not evidence of a successful runtime deployment;
rerun after freeing Docker storage.

