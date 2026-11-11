# Deployment na Ubuntu Server LTS

1. Utwórz dedykowanego użytkownika `minecraft`, zablokuj logowanie hasłem i przygotuj katalog `/opt/minecraft-network`.
2. Zainstaluj aktualne Docker Engine oraz Compose Plugin z oficjalnego repozytorium.
3. Skopiuj repozytorium do katalogu deploymentu, ustaw właściciela na `minecraft:minecraft` i uprawnienia katalogów na minimum wymagane.
4. Skopiuj `.env.example` do `.env`, ustaw silne lokalne hasło PostgreSQL i nie zapisuj `.env` w Git.
5. Skonfiguruj SSH, UFW i Fail2Ban według [security.md](security.md).
6. Zweryfikuj `docker compose -f docker/docker-compose.yml config`.
7. Wygeneruj i przejrzyj konfigurację Velocity/Purpur, dodaj zatwierdzone pluginy i forwarding secret.
8. Uruchom `docker compose -f docker/docker-compose.yml up -d` i wykonaj `./scripts/healthcheck.sh`.
9. Skonfiguruj dashboardy, alerty i Uptime Kuma dopiero po potwierdzeniu działania usług.

Pierwszy deployment wykonuj testowo. Nie otwieraj portów baz danych ani paneli monitoringu do Internetu bez osobnej decyzji bezpieczeństwa.

