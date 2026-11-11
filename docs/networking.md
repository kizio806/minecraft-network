# Networking

Publiczny jest wyłącznie TCP 25565 do Velocity oraz ograniczony dostęp administracyjny SSH. Lobby, Auth i Survival komunikują się po prywatnej sieci Compose. Forwarding gracza musi używać Velocity modern forwarding i unikalnego sekretu; backendy nie mogą przyjmować ruchu bezpośrednio z Internetu.

