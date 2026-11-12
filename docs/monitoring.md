# Monitoring

Prometheus zbiera metryki Prometheusa, node-exporter, cAdvisor i `mc-monitor`. Exporter odpytuje Proxy, Lobby i Survival po prywatnej sieci i dostarcza faktyczne metryki dostępności, wersji i liczby graczy. Grafana ma automatycznie provisionowane datasource Prometheus/Loki, Loki przechowuje logi na trwałym volume, a Grafana Alloy czyta logi kontenerów przez read-only socket Dockera. Uptime Kuma jest dostępny lokalnie i służy do probe'ów administracyjnych.

TPS/MSPT i metryki JVM/Purpur nie są deklarowane jako gotowe, ponieważ `mc-monitor` ich nie dostarcza. Po zatwierdzeniu pluginu metrics albo endpointu JVM należy dodać osobny scrape target i dashboard.

Monitoring jest ograniczony do localhost przez `MONITORING_BIND=127.0.0.1`. Prometheus, Loki, Grafana i Uptime Kuma nie mają publicznych portów. Dane Prometheus, Loki, Grafana, Alloy i Uptime Kuma są przechowywane w volume; retencja Prometheusa wynosi 15 dni.

Każdy obraz Compose jest przypięty do wersji oraz digestu `sha256`. Aktualizacja obrazu wymaga przeglądu release notes, weryfikacji digestu dla właściwej architektury, zmiany konfiguracji i przejścia walidacji CI; tag bez digestu jest odrzucany przez workflow. Digesty są przechowywane w `docker/docker-compose.yml` lub `.env.example` i aktualizowane jawnie.

Redis ma profil Compose `cache` i nie startuje domyślnie, ponieważ obecny szkielet nie ma pluginu, który go potrzebuje. Włączenie: `docker compose --profile cache up -d redis`.

NanoLimbo jest prywatnym auth/limbo backendem wymaganym przez VeloAuth; nie ma publikowanego portu hosta. VeloAuth pozostaje właścicielem rejestracji, haseł, Premium verification i transferu do Lobby/Survival.

Alloy i cAdvisor są wyjątkami bezpieczeństwa: Alloy ma read-only dostęp do `/var/run/docker.sock`, a cAdvisor do read-only filesystemów hosta. Są potrzebne odpowiednio do logów kontenerów i metryk zasobów kontenerów.
