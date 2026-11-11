# Architektura

```text
Internet
   |
TCP 25565
   v
[Velocity Proxy] -- private Docker network --> [Lobby]
                         |                    [Auth]
                         |                    [Survival]
                         |
                 [PostgreSQL] [Redis]

[Prometheus] --> [Grafana]
[Docker logs] --> [Loki]
[Uptime Kuma] --> service probes
```

Velocity jest jedynym publicznym punktem wejścia. Backendowe serwery nie publikują portów hosta. PostgreSQL i Redis są dostępne tylko w sieci wewnętrznej. Monitoring ma osobną sieć; przed wystawieniem paneli poza localhost należy dodać reverse proxy, TLS i uwierzytelnianie.

Przepływ: klient -> proxy -> wybrany backend -> dane aplikacyjne w PostgreSQL/Redis. Logi i metryki nie powinny zawierać sekretów ani danych wrażliwych.

