# Minecraft Network

Profesjonalny, lokalny szkielet infrastruktury prywatnego Minecraft SMP dla około 20 graczy. Projekt jest przygotowany pod Ubuntu Server LTS, Docker Compose, Velocity, Purpur 1.21.11, PostgreSQL, Redis i monitoring.

## Architektura

Ruch graczy trafia wyłącznie do Velocity Proxy, który kieruje połączenia do izolowanych backendów: Lobby, Auth i Survival. Baza PostgreSQL przechowuje dane trwałe, Redis zapewnia cache, a Prometheus, Grafana, Loki i Uptime Kuma obserwują usługi. Szczegóły znajdują się w [docs/architecture.md](docs/architecture.md).

## Wymagania

- Ubuntu Server LTS, dostęp administracyjny tylko do instalacji
- Docker Engine i Docker Compose Plugin
- DNS lub adres IP serwera oraz otwarty port TCP 25565
- Java i pluginy dostarczane później przez kontrolowany proces wydawniczy

## Instalacja

```bash
cp .env.example .env
./scripts/install.sh
docker compose -f docker/docker-compose.yml config
docker compose -f docker/docker-compose.yml up -d
```

Instalacja produkcyjna jest opisana krok po kroku w [docs/deployment.md](docs/deployment.md). Przed startem trzeba uzupełnić lokalne sekrety i zaakceptować EULA zgodnie z prawem/licencją Mojang.

## CLI

Docelowe komendy to `network install|start|stop|restart|status|logs|backup|restore|update|health`. Obecnie dostępny jest bezpieczny przykładowy wrapper: [cli/README.md](cli/README.md) oraz skrypty w `scripts/`.

## Roadmapa

1. Utwardzenie hosta Ubuntu i pierwszy deployment testowy.
2. Dodanie zatwierdzonych wersji Velocity/Purpur oraz pluginów z checksumami.
3. Konfiguracja routingu, forwarding secret i trybów Auth/Lobby/Survival.
4. Testy backupu i odtworzenia na osobnym środowisku.
5. Alerty, dashboardy i procedury aktualizacji bez przestoju.

