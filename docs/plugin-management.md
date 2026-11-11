# Zarządzanie pluginami

Każdy plugin otrzymuje przypiętą wersję, źródło pobrania, checksumę SHA-256, listę zależności, zakres kompatybilnych wersji i plan rollbacku. Pluginy trafiają do kontrolowanego katalogu dopiero po przeglądzie licencji, uprawnień i logów.

Konfiguracje pluginów są wersjonowane oddzielnie od danych świata. Aktualizacje przechodzą przez środowisko testowe, backup, smoke test proxy/backendów i obserwację po wdrożeniu. W pierwszym commicie nie ma żadnych pluginów ani JAR-ów.

