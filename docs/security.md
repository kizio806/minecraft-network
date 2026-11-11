# Bezpieczeństwo

## Host i użytkownicy

- Usługi działają w kontenerach bez trybu privileged i bez montowania socketu Dockera.
- Hostowy użytkownik `minecraft` nie powinien mieć powłoki administracyjnej; operacje wymagające `sudo` muszą być jawne i audytowane.
- Katalogi runtime nie mogą być zapisywalne przez osoby nieuprawnione.

## SSH

- Wyłącz logowanie root i hasłami; używaj kluczy oraz ogranicz użytkowników przez `AllowUsers`.
- Ustaw aktualny port tylko jako element redukcji szumu, nie jako kontrolę bezpieczeństwa.
- Włącz aktualizacje bezpieczeństwa i Fail2Ban dla SSH.

## Sieć

- UFW: domyślnie deny incoming, allow outgoing; otwórz tylko SSH z zaufanych adresów i TCP 25565.
- PostgreSQL, Redis, Prometheus, Grafana, Loki i Uptime Kuma nie powinny być publiczne.
- Włącz `no-new-privileges`, ogranicz capabilities i stosuj sieci Compose zgodnie z przeznaczeniem.

## Sekrety

Forwarding secret, hasła i tokeny są dostarczane przez `.env`/menedżer sekretów. Nie umieszczaj ich w konfiguracji, issue ani logach. Po wycieku rotuj je i unieważnij stare.

