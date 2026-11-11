# `network` CLI

Planowany interfejs operatorski:

```text
network install       # przygotowanie hosta i katalogów
network start|stop    # uruchomienie lub zatrzymanie Compose
network restart       # kontrolowany restart
network status|logs   # stan i logi usług
network backup        # backup lokalny
network restore       # przywrócenie po potwierdzeniu
network update        # aktualizacja według runbooka
network health        # healthcheck usług i hosta
```

Na tym etapie skrypty `scripts/*.sh` są referencyjnymi punktami wejścia. Komenda `restore` pozostaje celowo niezaimplementowana do czasu ustalenia polityki potwierdzeń i lokalizacji archiwów.

