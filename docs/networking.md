# Networking

Publiczny jest wyłącznie TCP 25565 do Velocity oraz ograniczony dostęp administracyjny SSH. `proxy_net` służy do ruchu proxy-backend, `data_net` jest wewnętrzną siecią danych łączącą Proxy, Lobby, Survival i PostgreSQL (oraz opcjonalny Redis), a `monitoring` jest prywatna. Lobby, Survival i PostgreSQL nie publikują portów; PostgreSQL jest dostępny wyłącznie wewnątrz `data_net` dla LuckPerms. Redis uruchamia się tylko z uzasadnionym profilem.

Forwarding używa modern forwarding i sekretu `/run/secrets/forwarding_secret`. Proxy ma `online-mode=false`, ponieważ VeloAuth wybiera per-connection tryb Premium/Non-Premium; backendy mają `online-mode=false` oraz Paper Velocity forwarding z tym samym sekretem. VeloAuth pozostaje jedynym komponentem decydującym o autoryzacji, a bezpośrednie połączenie do backendów z Internetu nie jest możliwe. Auth/plugin secret nie jest współdzielony z forwarding secret.

## Ports

| Port | Scope | Purpose |
|---:|---|---|
| TCP 25565 | public | Velocity player entry point |
| TCP SSH | public, restricted | administration |
| TCP 3000 | localhost | Grafana |
| TCP 3001 | localhost | Uptime Kuma |
| TCP 9090/3100 | Docker internal | Prometheus/Loki |
| TCP 25565 (backend names) | Docker internal | Lobby, Survival, NanoLimbo |
| TCP 5432 | Docker internal `data_net` | PostgreSQL (LuckPerms shared storage) |
| TCP 6379 | optional `data_net` | Redis (`cache` profile) |

## DNS and Cloudflare

Initial deployment needs only `play.example.com` as DNS-only (`A`/`AAAA`) pointing at the VPS. Standard Cloudflare orange-cloud HTTP proxying does not proxy ordinary Minecraft Java TCP traffic. Cloudflare Spectrum or another supported TCP product would be a separate paid/networking decision; it is not required for the baseline. `status.example.com` is intentionally deferred until a real status service exists.

UFW allows only the verified SSH port and TCP 25565. Docker-internal ports are not firewall entry points.

