# Backup

Backup obejmuje PostgreSQL, konfiguracje oraz kompletne dane `servers/*` wraz ze światem Survival. Redis jest cache'em i nie jest źródłem prawdy, ale jego dump może być pomocny przy diagnostyce.

Minimalna polityka lokalna: codzienny backup przyrostowy/logiczny, cotygodniowy pełny snapshot, retencja ustalona przed produkcją oraz okresowy test odtworzenia na izolowanym katalogu. Backup nie może być jedyną kopią danych.

Przywracanie: zatrzymaj backendy, zabezpiecz bieżący stan, przywróć najpierw bazę i konfigurację, następnie świat, uruchom usługi i wykonaj healthcheck. Każde przywrócenie zapisuj w dzienniku operacyjnym.

