# ADR 0009: Docker network boundaries

## Status

Accepted

## Decision

`proxy_net` carries Proxy, NanoLimbo, Lobby, Survival and the Minecraft exporter. `data_net` is internal and reserved for backend/application data services. `monitoring` is internal; monitoring UIs bind to localhost only. No backend, database or monitoring port is public.
