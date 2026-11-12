# Backup i restore

`network backup` zatrzymuje Proxy i backendy, archiwizuje konfigurację, dane serwerów i sekrety oraz wykonuje `pg_dump` bazy PostgreSQL (w tym bazy `luckperms`). Następnie szyfruje archiwum w formacie age i tworzy checksumę SHA-256 szyfrowanego pliku.

Lokalizacja to `BACKUP_DIR`; dla produkcji powinna znajdować się na osobnym, chronionym filesystemie. Redis jest opcjonalnym cache'em i nie jest źródłem prawdy; jego backup wymaga osobnego dumpa dopiero wtedy, gdy plugin użyje go jako danych trwałych.

```bash
network backup
network restore /var/backups/minecraft-network/minecraft-network-<timestamp>.tar.gz.age --yes
```

Restore wymaga jawnego `--yes`, sprawdza checksumę, odszyfrowuje archiwum przez klucz z `BACKUP_AGE_IDENTITY_FILE`, sprawdza bezpieczeństwo ścieżek, zatrzymuje backendy, przywraca pliki i bazę, a następnie uruchamia usługi. Klucze age pozostają poza repozytorium i archiwami; oba pliki kluczy muszą mieć `0600`.

Wygeneruj parę kluczy narzędziem `age-keygen` poza repozytorium. Publiczny recipient umieść w `BACKUP_AGE_RECIPIENT_FILE`, a prywatną identity na chronionym hoście w `BACKUP_AGE_IDENTITY_FILE`. Rotacja wymaga zachowania starej identity do odczytu istniejących backupów, utworzenia nowej pary i wykonania nowego backupu. Utrata identity oznacza brak możliwości odtworzenia kopii.

Backup zawiera sekrety i runtime JAR-y potrzebne do odtworzenia deploymentu, dlatego katalog backupów musi mieć ograniczony dostęp i dodatkową ochronę storage. Test restore należy wykonywać cyklicznie na izolowanym hoście. JAR-y nie są przechowywane w Git, ale są częścią lokalnego backupu runtime.

Produkcja wymaga drugiej, zewnętrznej kopii na innym hostcie lub w object storage. Kopia zewnętrzna powinna być szyfrowana, mieć osobne ograniczone credentials, retencję zgodną z polityką operacyjną i regularny test odtworzenia. Dostawca zewnętrzny pozostaje decyzją wdrożeniową, nie jest obecnie dodawany do Compose.
