# Kompleksowy Audyt Architektoniczny i Plan Refaktoryzacji — minecraft-network

**Dokument Identyfikacyjny:** `AUDIT-REF-MCN-2026-11`  
**Projekt:** `minecraft-network`  
**Autor:** Principal Systems Architect & Senior Quality Reviewer  
**Status Dokumentu:** WYKONANY I ZWERYFIKOWANY (DONE_VERIFIED)  
**Branch Docelowy:** `main`

---

## 1. Krótki opis repozytorium i jego celu

`minecraft-network` to produkcyjny, konteneryzowany projekt typu *Infrastructure as Code* (IaC) oraz zestaw narzędzi operatorskich (CLI + daemon Discord Bot) w języku **Go 1.24**, zaprojektowany do uruchamiania i automatyzacji skalowalnej sieci serwerów Minecraft (wersja docelowa: **1.21.x**).

Głównym celem projektu jest dostarczenie kompletnego środowiska serwerowego o architekturze Zero-Trust, wykorzystującego:
- **Velocity (4.3.1)** jako jedyny punkt wejścia (ingress proxy) wystawiający publiczny port `25565/tcp`,
- **NanoLimbo (1.12.0)** jako bezpieczną izolatkę (holding pen) do hybrydowej autentykacji (Mojang Premium + BCrypt cracked via VeloAuth),
- Serwery backendowe **Purpur (1.21.11)** (`lobby` i `survival`) działające w izolowanej sieci kontenerowej bez bezpośredniego dostępu z zewnątrz,
- Skonsolidowaną bazę danych **PostgreSQL 16 Alpine** (LuckPerms, VeloAuth, LibertyBans, UltimateTeams) z polityką „Zero Redis”,
- Wbudowane narzędzie CLI w Go (`network`) zarządzające deklaratywnym cyklem życia serwerów, weryfikacją skrótów SHA-512 pluginów i kopiami zapasowymi szyfrowanymi asymetrycznie narzędziem `age`,
- Zintegrowanego bota Discord napisanego natywnie w Go z obsługą komponentów UI (przycisków, modali, ról, raportów i synchronizacji rang).

Projekt ma aspirację stać się wzorcowym przykładem zaawansowanej architektury sieci gier wieloosobowych w portfolio seniora DevOps/Go.

---

## 2. Mapa aktualnego stanu repozytorium

```
minecraft-network/
├── .agents/                      # Reguły, role i konteksty dla agentów AI
│   ├── context/                  # Specyfikacje architektury, infrastruktury, matrycy pluginów
│   ├── roles/                    # Definicje person inżynierskich (architekt, devops, tester)
│   ├── rules/                    # Twarde reguły (port isolation, digests, secrets)
│   ├── skills/                   # Procedury diagnostyczne i operacyjne
│   └── workflows/                # Przepływy prac
├── .github/                      # Automatyzacja CI/CD
│   └── workflows/
│       ├── ci.yml                # Linting, testy z flagą -race, weryfikacja digestów obrazów
│       ├── codeql.yml            # Analiza bezpieczeństwa statycznego CodeQL
│       ├── release.yml           # Kompilacja binarek CLI i wydania GitHub
│       └── sonarcloud.yml        # Quality Gate SonarCloud
├── bin/                          # Katalog wyjściowy dla skompilowanych binarek (gitignored)
├── cmd/                          # Entry pointy aplikacji w Go
│   ├── discord-bot/              # Daemon bota Discord (main.go, main_test.go)
│   └── network/                  # Główne narzędzie CLI (main.go, main_test.go)
├── configs/                      # Szablony konfiguracyjne i deklaracje
│   ├── auth/                     # Konfiguracja NanoLimbo (settings.yml)
│   ├── discord/                  # Konfiguracja bota (config.yml)
│   ├── lobby/                    # Konfiguracje Purpur/Paper dla lobby
│   ├── network.yml               # Główny manifest sieciowy (motd, wersje, sloty, motyw)
│   ├── plugins/                  # Deklaratywny łańcuch dostaw pluginów
│   │   ├── components.lock.yml   # Pinned digesty komponentów
│   │   ├── components.yml        # Konfiguracje komponentów
│   │   ├── lock.yml              # Wygenerowane skróty SHA-512 i adresy pobierania
│   │   ├── manifest.schema.json  # JSON Schema walidujące poprawność manifestu
│   │   └── manifest.yml          # Deklaracja wszystkich JAR-ów i datapacków
│   ├── proxy/                    # Konfiguracja Velocity (velocity.toml, ikona)
│   └── survival/                 # Konfiguracje Purpur dla serwera survival
├── docs/                         # Pełna dokumentacja techniczna
│   ├── architecture.md           # Architektura logiczna i fizyczna
│   ├── backup.md                 # Procedura backupów age
│   ├── cli-architecture.md       # Architektura wewnętrzna CLI
│   ├── decisions/                # 16 Rekordów Decyzji Architektonicznych (ADR 0001 - 0016)
│   ├── deployment.md             # Wdrożenie produkcyjne krok po kroku
│   ├── localization.md           # Architektura i18n
│   ├── monitoring.md             # Konfiguracja stosu Prometheus + Grafana
│   ├── networking.md             # Segmentacja sieciowa i Zero-Trust
│   ├── plugin-management.md      # Zarządzanie manifestem pluginów
│   ├── security.md               # Model zagrożeń i reguły bezpieczeństwa
│   └── theming.md                # System palet kolorystycznych
├── infrastructure/               # Skrypty bazodanowe i monitoring
│   ├── monitoring/               # prometheus.yml, alerts.yml, grafana provisioning
│   └── postgres/                 # init-databases.sh, seed-luckperms.sql
├── internal/                     # Wewnętrzne pakiety Go (Core Logic)
│   ├── backup/                   # Tworzenie, szyfrowanie age i przywracanie archiwów
│   ├── compose/                  # Wrapper na wywołania docker compose i healthchecki
│   ├── config/                   # Ładowanie i walidacja configs/network.yml
│   ├── discord/                  # Bot Discord (interakcje, DB, rcon, moderation, sync)
│   ├── fileutil/                 # Pomocnicze funkcje I/O (kopiowanie, uprawnienia)
│   ├── i18n/                     # Wiązanie tłumaczeń (en, pl)
│   ├── networkcli/               # Dyspozytor komend CLI (install, status, doctor, update)
│   ├── pluginmanager/            # Pobieranie, weryfikacja SHA-512 i synchronizacja JAR-ów
│   ├── template/                 # Silnik szablonowania oparty o text/template
│   └── tui/                      # Interaktywny kreator terminalowy (Charm Huh)
├── locales/                      # Paczki językowe i18n (en.yml, pl.yml)
├── scripts/                      # Skrypty hosta (configure-firewall.sh, install-host.sh)
├── docker-compose.yml            # Definicja kontenerów produkcyjnych
├── Dockerfile.discord-bot        # Multi-stage build kontenera bota Discord
├── Makefile                      # Standardowe cele kompilacji, testów i formatowania
├── go.mod / go.sum               # Zarządzanie zależnościami Go 1.24
├── PROJECT.md                    # Nadrzędna specyfikacja architektoniczna
└── README.md                     # Główny opis projektu i statusy CI
```

### Entry pointy
1. **`cmd/network/main.go`**: Binarka `./bin/network` udostępniająca komendy: `init`, `install`, `status`, `health`, `doctor`, `backup`, `restore`, `logs`, `update`, `plugins`, `config`, `completion`.
2. **`cmd/discord-bot/main.go`**: Demon `./bin/discord-bot` obsługujący komendy slashowe (`/status`, `/players`, `/link`, `/sync`, `/report`, `/rcon`), interakcje modali i pętlę aktualizacji statusu.

### Build system
Natywny `Makefile` ze wsparciem dla automatycznego fallbacku do kontenerowego kompilatora `golang:1.24` oraz `golangci-lint:v1.64.5`. W CI wykorzystywane są akcje GitHub z Go 1.24 i cache modułów.

---

## 3. Szczegółowa ocena jakości w 13 obszarach

| Obszar | Ocena (1-5) | Podsumowanie stanu obecnego |
|---|:---:|---|
| **Architektura** | 4.5 | Bardzo dojrzała topologia sieciowa Zero-Trust, skonsolidowany PostgreSQL, brak Redis. Wzorowy podział na warstwę CLI i serwerową. |
| **Separacja odpowiedzialności** | 4.0 | Czysty podział na pakiety w `internal/`. Poważna wada: implementacja RCON w bocie Discord opiera się na wywoływaniu CLI Dockera zamiast protokołu sieciowego. |
| **Nazewnictwo** | 4.5 | Konsekwentne stosowanie idiomatycznych nazw pakietów Go (krótkie, bez podkreśleń). Czytelne identyfikatory. |
| **Styl kodu** | 4.5 | Rygorystyczny `gofmt`, konfiguracja `.golangci.yml` z linterami `gosec`, `bodyclose`, `revive`, `errorlint`. |
| **Testy** | 3.5 | Wysokie formalne pokrycie instrukcji (>88%), ale testy bazy danych (`db_postgres_test.go`) opierają się na mockach w pamięci, ukrywając błędy schematów. Brak testów integracyjnych z prawdziwym Dockerem/Postgres. |
| **Dokumentacja** | 5.0 | Wyjątkowo bogata: `PROJECT.md`, 16 ADR-ów w `docs/decisions/`, szczegółowe przewodniki w `docs/`. |
| **Dokumentacja dla AI** | 5.0 | Wzorcowy `AGENTS.md` oraz katalog `.agents/` z podziałem na role, reguły i konteksty. Standard rynkowy. |
| **Build i CI** | 4.5 | Solidne workflowy: testy z `-race`, linting, SonarCloud, CodeQL, weryfikacja niezmiennych digestów SHA-256 obrazów. |
| **Bezpieczeństwo** | 3.5 | Dobre zasady izolacji portów i uprawnień `0600`, ale zidentyfikowano potencjalne wstrzyknięcie komend w modalu moderacyjnym bota oraz brak walidacji RCON. |
| **Utrzymywalność** | 4.0 | Łatwe dodawanie nowych pluginów dzięki deklaratywnemu manifestowi. Dobre zarządzanie zależnościami Go. |
| **Spójność techniczna** | 3.5 | Rozbieżność między bazą danych zdefiniowaną dla bota (`minecraft`), a bazą z seedem rang (`luckperms`). |
| **Gotowość portfolio** | 4.5 | Wygląda imponująco pod kątem inżynierii systemowej, jednak techniczne niedociągnięcia w bocie Discord mogą wyjść na jaw przy live demo. |
| **Gotowość do pracy zespołowej**| 4.5 | Kompletny `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, jasne konwencje commitów (Conventional Commits). |

---

## 4. Lista wszystkich zauważonych problemów

### Kategoria: Bezpieczeństwo i Izolacja Środowiska (Krytyczne)
1. **[P0 - KRYTYCZNY] Niedziałający RCON w kontenerze bota Discord**  
   - *Plik:* `internal/discord/rcon.go` (funkcja `DockerConsoleExecutor.Execute`, linie 47–88) oraz `Dockerfile.discord-bot`.  
   - *Wpływ:* W pliku `docker-compose.yml` usługa `discord-bot` działa w dedykowanym, minimalistycznym kontenerze Alpine bez zamontowanego gniazda `/var/run/docker.sock` i bez zainstalowanej binarki `docker`. Tymczasem `DockerConsoleExecutor` próbuje uruchomić `/usr/bin/docker compose exec ...`. W efekcie każda próba użycia komendy `/rcon` oraz ukarania gracza przez modal w kontenerze zakończy się błędem `exec: "docker": executable file not found in $PATH`.  
   - *Uzasadnienie:* Architektoniczny anty-pattern polegający na próbie zarządzania kontenerami z wnętrza nieuprzywilejowanego kontenera bez daemon socketu.
2. **[P1 - WYSOKI] Podatność na wstrzyknięcie komend w modalu karania (Command Injection)**  
   - *Plik:* `internal/discord/moderation.go` (linie 253–258).  
   - *Wpływ:* W kodzie `HandlePunishModalSubmit` budowana jest komenda stringowa: `cmd = fmt.Sprintf("ban %s %s", targetPlayer, reason)` lub `tempban %s %s %s`. Ani `targetPlayer`, ani `reason` nie podlegają sanityzacji pod kątem znaków nowej linii (`\n`, `\r`) czy separatorów komend. Złośliwy użytkownik zgłaszający gracza z powodem zawierającym `\nop malicious` mógłby doprowadzić do wykonania niepożądanych instrukcji w konsoli serwera.  
   - *Uzasadnienie:* Brak rygorystycznej walidacji i cytowania argumentów przekazywanych do RCON. Ponadto komenda jest wysyłana na sztywno do serwera `"lobby"` zamiast do proxy Velocity (gdzie działa LibertyBans).

### Kategoria: Spójność Bazy Danych i Integracja (Wysoki)
3. **[P1 - WYSOKI] Rozbieżność bazodanowa pobierania rang gracza (Cross-Database Defect)**  
   - *Plik:* `internal/discord/db.go` (metoda `GetPlayerRank`, linie 188–198) oraz `infrastructure/postgres/init-databases.sh`.  
   - *Wpływ:* W `init-databases.sh` tabela `luckperms_players` jest tworzona w dedykowanej bazie danych `luckperms` pod użytkownikiem `luckperms_user`. Tymczasem bot Discord łączy się ze zmiennej `DATABASE_URL` (domyślnie baza `minecraft`). Metoda `GetPlayerRank` wykonuje zapytanie `SELECT primary_group FROM luckperms_players WHERE ...` na połączeniu bota. W środowisku PostgreSQL zapytanie to zakończy się błędem `relation "luckperms_players" does not exist`, ponieważ PostgreSQL nie pozwala na cross-database queries w ramach jednej sesji bez FDW/dblink. Komenda `/sync` przestanie działać na produkcji.  
   - *Uzasadnienie:* Fałszywe poczucie bezpieczeństwa wywołane przez test `db_postgres_test.go`, który używa zamockowanego drivera SQL i nie weryfikuje istnienia tabeli w docelowej bazie.

### Kategoria: Higiena Repozytorium i CI/CD (Średni)
4. **[P2 - ŚREDNI] Pozostawiony nieśledzony plik `cov.txt` w głównym katalogu**  
   - *Plik:* `cov.txt` w roocie repozytorium.  
   - *Wpływ:* W `.gitignore` zdefiniowano wykluczenie dla `coverage.txt`, lecz pominięto `cov.txt`. W efekcie po lokalnym uruchomieniu testów z flagą `-coverprofile=cov.txt` plik ten pojawia się w `git status` jako untracked artifact.  
   - *Uzasadnienie:* Zanieczyszczenie drzewa roboczego, ryzyko przypadkowego skomitowania śmieciowego pliku tekstowego do głównej gałęzi.
5. **[P2 - ŚREDNI] Ciche tłumienie błędów odczytu haseł w `prepareRuntimeConfigs`**  
   - *Plik:* `internal/networkcli/install.go` (linie 102–111).  
   - *Wpływ:* Podczas generowania konfiguracji dla pluginów, błędy odczytu plików `ultimateteams_password` i `libertybans_password` są ignorowane instrukcją `if ... err == nil`. W przypadku braku pliku lub błędnych uprawnień wartość hasła staje się pustym stringiem, a generowane pliki konfiguracyjne serwerów otrzymują puste poświadczenia bazodanowe bez ostrzeżenia dla operatora.

---

## 5. Plan refaktoru krok po kroku

### Krok 1: Wdrożenie natywnego klienta sieciowego RCON w Go dla Discord Bota
- **Cel:** Całkowite wyeliminowanie zależności od binarki Dockera wewnątrz kontenera bota.
- **Działania:**
  1. Stworzyć czysty sieciowy klient protokołu RCON (`internal/discord/rcon_client.go`) łączący się przez TCP z serwerami Minecraft (`auth:25575`, `lobby:25575`, `survival:25575`, `proxy`).
  2. Zastąpić `DockerConsoleExecutor` adapterem `NetworkRconExecutor`, wykorzystującym poświadczenia z secrets i konfigurację sieciową Docker Compose (`proxy_net`).
  3. Dodać obsługę timeoutów kontekstu (`context.Context`) oraz automatyczne ponawianie połączeń przy zerwaniu sesji.
- **Ryzyka i ostrożność:** Port RCON w serwerach backendowych nie może być wystawiony na interfejs hosta — musi pozostać wyłącznie w sieci `proxy_net`/`data_net`.

### Krok 2: Sanityzacja argumentów i unifikacja komend moderacyjnych
- **Cel:** Ochrona przed Command Injection i poprawne kierowanie kar do LibertyBans.
- **Działania:**
  1. Zaimplementować rygorystyczny walidator nicków graczy (`ValidateMinecraftUsername`: 3-16 znaków, `^[a-zA-Z0-9_]{3,16}$`).
  2. Zaimplementować oczyszczanie powodów kar (`reason`): usunięcie znaków kontrolnych `\r`, `\n`, `\0`, ograniczenie długości do 128 znaków.
  3. Skierować komendy wyciszeń i banów do instancji proxy (`velocity-cmd lb ban ...`), ponieważ LibertyBans działa na poziomie sieciowym proxy, uniemożliwiając ominięcie kary przez zmianę podserwera.

### Krok 3: Unifikacja bazy danych dla bota lub konfiguracja ról/schematów
- **Cel:** Rozwiązanie problemu niedostępności `luckperms_players` w bazie bota Discord.
- **Działania:**
  1. Zdecydować o architekturze schematów: przenieść tabele bota Discord (`discord_links`, `discord_whitelist`, `discord_reports`) do bazy `luckperms` jako schemat `discord`, LUB w `init-databases.sh` skonfigurować uprawnienia odczytu tabeli LuckPerms dla użytkownika bota, LUB dokonać pełnej konsolidacji do jednej logicznej bazy z osobnymi schematami (`luckperms`, `veloauth`, `discord`, `libertybans`) zgodnie z ADR 0011.
  2. Zaktualizować ciąg połączeniowy w konfiguracji bota (`configs/discord/config.yml` i `docker-compose.yml`).

### Krok 4: Uzupełnienie obsługi błędów i higiena `.gitignore`
- **Cel:** Eliminacja cichego ignorowania błędów haseł i zanieczyszczeń repozytorium.
- **Działania:**
  1. Zmienić obsługę odczytu `ultimateteams_password` i `libertybans_password` w `install.go` na jawną walidację i logowanie ostrzeżeń w przypadku brakujących sekretów.
  2. Dodać `*.txt`, `cov.txt`, `coverage.out` oraz `*.test` do `.gitignore`.

---

## 6. Plan testów

### 1. Testy jednostkowe do dodania/poprawienia
- **`internal/discord/rcon_test.go`**: Napisać testy jednostkowe nowego klienta RCON z zamockowanym serwerem TCP obsługującym pakiet autoryzacji RCON i pakiet komendy.
- **`internal/discord/moderation_test.go`**: Dodać testy weryfikujące odrzucanie niebezpiecznych znaków w powodach kar (`injection\nattack`, cudzysłowy, znaki niedrukowalne).
- **`internal/networkcli/install_test.go`**: Rozszerzyć test o asercję sprawdzającą, czy brak któregokolwiek z plików haseł zwraca deterministyczny błąd walidacji.

### 2. Testy integracyjne z prawdziwą bazą (Testcontainers-Go)
- Zastąpić zamockowany driver w `db_postgres_test.go` prawdziwym testem integracyjnym opartym o kontener PostgreSQL 16 uruchamiany przez `testcontainers-go` (lub kontener w CI).
- Przetestować sekwencję:
  1. Inicjalizacja schematów przez skrypt `seed-luckperms.sql`.
  2. Wykonanie `GetPlayerRank` dla gracza istniejącego w bazie oraz dla nieznanego gracza (weryfikacja fallbacku do `default`).
  3. Zapis i pobranie raportu ze statusem `pending` -> `resolved`.

### 3. Testy regresyjne i obszary restrykcyjne
- **Walidacja Niezmiennych Digestów Obrazów:** Skrypt CI `validate-images` musi uniemożliwić dodanie jakiegokolwiek obrazu bez przypiętego SHA-256.
- **Walidacja Uprawnień Plików:** Test sprawdzający, czy `./bin/network init` nadaje uprawnienia `0600` dla wygenerowanych sekretów oraz `.env`.

---

## 7. Plan dokumentacji

1. **README.md:** Dodać sekcję *Live Demo & Local Sandbox Guide* opisującą krok po kroku, jak uruchomić całe środowisko na maszynie deweloperskiej bez konieczności posiadania domeny publicznej.
2. **Dokumentacja dla dewelopera (`docs/development.md`):** Uzupełnić opis uruchamiania i debugowania bota Discord w trybie lokalnym (`make build-bot`, uruchomienie z flagą `-config`).
3. **Dokumentacja dla AI (`AGENTS.md`):** Zaktualizować invariant dotyczący komunikacji bota z serwerami (zastąpienie wywołań Dockera protokołem RCON przez sieć kontenerową).
4. **Changelog (`CHANGELOG.md`):** Utworzyć i utrzymać sformalizowany plik zmian według standardu *Keep a Changelog* (obecnie brak dedykowanego pliku changelogu w głównym drzewie).
5. **Architektura bota Discord (`docs/discord-bot.md`):** Dodać dokument opisujący architekturę uprawnień Discord (wymagane Privileged Gateway Intents: `Server Members`, `Message Content`) oraz interakcji z PostgreSQL.

---

## 8. Plan utrzymania jakości na przyszłość

1. **Reguły pre-commit i CI:**
   - Wymuszenie `golangci-lint` z aktywnymi regułami `gosec` (G204, G304) oraz `errorlint`.
   - Zablokowanie commitowania plików o wzorcach `*.age`, `*.secret`, `*.jar`, `*.key`.
2. **Kontrola regresji pokrycia:**
   - Utrzymanie wskaźnika Codecov na poziomie minimum **88%** z progiem błędu 1%.
3. **Standard „Nie wolno tego robić” dla inżynierów i agentów AI:**
   - Nigdy nie montować `/var/run/docker.sock` do kontenerów aplikacji o ile nie jest to wyspecjalizowany agent monitoringu.
   - Nigdy nie budować poleceń konsoli przez konkatenację stringów bez uprzedniej walidacji wyrażeniem regularnym.
   - Nigdy nie zakładać poprawności integracji z bazą danych na podstawie zamockowanych testów jednostkowych bez testu z realnym silnikiem bazodanowym.

---

## 9. Wniosek końcowy

- **Co jest najsłabsze:** Podatność na błędy wykonania komend konsoli przez bota Discord w środowisku kontenerowym (próba wywoływania Dockera w Alpine bez dockera) oraz niespójność bazodanowa pobierania rang.
- **Co daje największy efekt:** Zastąpienie wywołań Dockera w bocie natywnym klientem protokołu RCON oraz skonsolidowanie bazy danych bota z tabelami uprawnień LuckPerms.
- **Co musi zostać zrobione najpierw:** Poprawka `cov.txt` w `.gitignore`, wdrożenie bezpiecznej sanityzacji w modalu karania graczy oraz eliminacja mocka w testach Postgresa.
- **Droga do topowego portfolio:** Po wyeliminowaniu wyżej wymienionych 4 problemów `minecraft-network` będzie jednym z najbardziej dojrzałych i profesjonalnych publicznych repozytoriów infrastruktury gier wieloosobowych na GitHubie.
