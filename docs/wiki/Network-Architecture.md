# Network Architecture

This document describes the architectural topology, container orchestration, and communication protocols governing the **Minecraft Network**.

---

## 🏛️ Topology

```mermaid
flowchart TD
    Internet([Public Internet / Players]) -->|25565/TCP| Proxy[Velocity Modern Forwarding Proxy]

    subgraph Internal ["Internal Docker Bridge Network (Isolated)"]
        Proxy -->|Modern Forwarding Protocol| Lobby[Lobby Node (Paper)]
        Proxy -->|Modern Forwarding Protocol| Survival[Survival Node (Purpur)]
        Proxy -->|Modern Forwarding Protocol| Events[Event Stage (Folia)]

        Lobby <--> Redis[(Redis Cache)]
        Survival <--> Redis
        Events <--> Redis

        Lobby <--> DB[(PostgreSQL Cluster)]
        Survival <--> DB
        Events <--> DB
    end
```

---

## 🔒 Layered Isolation Invariants

1. **Edge Ingress Protection:** Only port `25565` on the Velocity container is mapped to the host's public IP address.
2. **Backend Port Isolation:** Backend server ports (`25566`, `25567`, etc.) are never published to the public network; they bind exclusively to the internal Docker bridge.
3. **Player Forwarding Secrets:** Velocity's modern forwarding secret is generated dynamically and injected securely into all backend instances at initialization time.
4. **Data Persistence Separation:** Databases and Redis instances use segregated volumes with non-root ownership.
