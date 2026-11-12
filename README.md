# Minecraft Network

Infrastruktura prywatnego Minecraft SMP dla około 20 graczy. Projekt jest przygotowany pod Ubuntu Server LTS, Docker Compose, Velocity 4.2.x, Purpur 1.21.11, VeloAuth i monitoring. Obsługiwane są zarówno oficjalne konta Premium, jak i konta Non-Premium.

## Architektura

Ruch graczy trafia wyłącznie do Velocity Proxy z VeloAuth. VeloAuth prowadzi zintegrowany auth/limbo flow, weryfikuje Premium przez Mojang i obsługuje BCrypt dla Non-Premium. Backendy są niedostępne z hosta. Szczegóły znajdują się w [docs/architecture.md](docs/architecture.md).

## Wymagania

- Ubuntu Server LTS, dostęp administracyjny tylko do instalacji
- Docker Engine i Docker Compose Plugin
- DNS lub adres IP serwera oraz otwarty port TCP 25565
- Java 25 dla Proxy/VeloAuth i Lobby; Survival pozostaje na dotychczasowym przypiętym runtime Java 21
- przypięte obrazy i kontrolowany proces wydawniczy pluginów

## Instalacja

```bash
cp .env.example .env
# Zweryfikuj lub zatwierdź wszystkie przypięte digesty obrazów przed wdrożeniem.
go build -trimpath -ldflags='-X github.com/kizio806/minecraft-network/internal/networkcli.Version=0.1.0' -o network ./cmd/network
./network install
./network start
./network health
```

Instalacja produkcyjna jest opisana krok po kroku w [docs/deployment.md](docs/deployment.md). Przed startem trzeba uzupełnić lokalne sekrety i zaakceptować EULA zgodnie z prawem/licencją Mojang.

## CLI

Głównym interfejsem jest natywny program Go: `network install|start|stop|restart|status|logs|backup|restore|update|health`. Skrypty Bash są ograniczone do bootstrapu hosta i przygotowania firewalla.

## Roadmapa

1. Test restore i smoke testy na odizolowanym hoście.
2. Zatwierdzenie digestów obrazów oraz artefaktu VeloAuth z checksumą.
3. Testy Premium, Non-Premium, rename i konfliktów UUID na stagingu.
4. Konfiguracja alertów i dashboardów operacyjnych.

