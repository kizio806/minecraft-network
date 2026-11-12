# Bezpieczeństwo

## Host i użytkownicy

- Usługi aplikacyjne działają jako UID 1000 i mają `no-new-privileges`; cAdvisor i Alloy mają opisane wyjątki wynikające z odczytu metryk hosta/logów Dockera.
- Hostowy użytkownik `minecraft` nie powinien mieć powłoki administracyjnej; operacje wymagające `sudo` muszą być jawne i audytowane.
- Katalogi runtime nie mogą być zapisywalne przez osoby nieuprawnione.

## SSH

- Wyłącz logowanie root i hasłami; używaj kluczy oraz ogranicz użytkowników przez `AllowUsers`.
- Ustaw aktualny port tylko jako element redukcji szumu, nie jako kontrolę bezpieczeństwa.
- Włącz aktualizacje bezpieczeństwa i Fail2Ban dla SSH.

## Sieć

- UFW: domyślnie deny incoming, allow outgoing; otwórz tylko SSH z zaufanych adresów i TCP 25565.
- PostgreSQL, Redis, Prometheus, Grafana, Loki i Uptime Kuma nie powinny być publiczne.
- Włącz `no-new-privileges`, ogranicz capabilities w obrazach, stosuj sieci Compose zgodnie z przeznaczeniem i utrzymuj limity CPU/RAM/PID.

## Sekrety

Forwarding secret, hasła i tokeny są dostarczane jako pliki Docker secrets przygotowywane przez `network install`. `.env` zawiera tylko ustawienia niesekretne i musi mieć uprawnienia `0600`. CLI nie umieszcza sekretów w argumentach ani logach. Nie umieszczaj sekretów w konfiguracji, issue ani logach. Po wycieku rotuj je i unieważnij stare.

Node Exporter i cAdvisor mają odczyt systemowych ścieżek hosta wymagany przez monitoring; są to wyjątki od minimalnego mount scope i powinny działać wyłącznie na dedykowanym hoście. Grafana i Uptime Kuma są domyślnie bindowane do `127.0.0.1`; nie wystawiaj ich bez dodatkowej warstwy dostępu.

Backupy są szyfrowane age. Recipient i prywatna identity są przechowywane poza repozytorium i archiwami, w plikach `0600`; CLI nigdy nie wypisuje ich zawartości. Błędny lub brakujący klucz kończy restore błędem przed rozpakowaniem danych.

## Authentication

- VeloAuth is the only authentication policy owner.
- Premium verification is performed through the official session flow; a name lookup or client-provided claim is never sufficient.
- `allow-cracked-on-premium-nicks=false` reserves unresolved Premium names and prevents first-connection impersonation.
- Mojang/API uncertainty must fail closed for identities requiring Premium verification; it must not silently become unrestricted offline access.
- Non-Premium passwords are handled by VeloAuth's BCrypt implementation and are never stored in plaintext.
- Registration/login timeouts, brute-force protection and per-IP admission limits remain enabled in the generated VeloAuth configuration; review the generated version before production.
- Unauthenticated players remain in NanoLimbo and cannot use backend commands, chat or gameplay until VeloAuth authorizes the session.
- `premium.bypass-auth-server=false` keeps the safe auth flow for every Java connection.
- Premium UUID and historical offline identity are not rewritten automatically; collisions require staging and an explicit migration decision.

## Threat model

| Threat | Boundary/mitigation |
|---|---|
| Premium nickname impersonation | VeloAuth strict Premium detection, fail-closed resolution and protected names |
| Non-Premium takeover | BCrypt password verification, rate limits, disconnect session invalidation |
| Direct backend access | No published backend ports; private Docker networks and forwarding secret |
| Forwarding spoofing | Modern forwarding with a separate Docker Secret |
| Auth database compromise | Private persistent SQLite volume, restricted filesystem/backup permissions, no plaintext passwords |
| Permissions database compromise | Private Docker `data_net` isolation, least-privilege `luckperms_user`, separate `luckperms_password` secret, no superuser exposure |
| Brute force/enumeration | VeloAuth throttling, registration/login timeouts and non-revealing auth flow |
| Malicious plugin/config | Manifest, review, SHA-256 gate, GitHub review and no mutable plugin downloads |
| VPS compromise | SSH keys, UFW, Fail2Ban, least privilege and encrypted external backups |
| Backup theft | Restricted backup directory, encrypted external copy and restore audit |

