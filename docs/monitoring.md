# Monitoring

Prometheus zbiera metryki scrape targets; Grafana prezentuje dashboardy; Loki przechowuje scentralizowane logi; Uptime Kuma wykonuje probe'y dostępności. Docelowo należy dodać node-exporter, cAdvisor, metryki JVM/Purpur i alerty dla RAM, dysku, TPS, opóźnień, restartów kontenerów, backupu oraz wygasania certyfikatów.

Monitoring nie zastępuje healthchecków ani logów audytowych. Panele pozostają prywatne, a retencja i limit dysku muszą być określone przed uruchomieniem produkcyjnym.

