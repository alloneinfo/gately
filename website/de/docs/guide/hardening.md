# Härtungsleitfaden

Die Verwaltungsoberfläche steuert DNS, Zertifikate und Ports. Arbeiten Sie die folgende Checkliste ab.

## Verwaltungsoberfläche

- [ ] Administratorpasswort nach der ersten Anmeldung ändern
- [ ] Zwei-Faktor-Authentifizierung (TOTP) aktivieren
- [ ] Einen sicheren Einstiegspfad festlegen, sodass die Verwaltungsoberfläche nur über einen eigenen Pfad erreichbar ist
- [ ] „Zugriff nur aus dem LAN“ aktivieren und für die Fernverwaltung ein VPN (etwa [Tailscale](/de/docs/modules/tunnels)) verwenden
- [ ] HTTPS für die Verwaltungsoberfläche aktivieren, wenn Sie über nicht vertrauenswürdige Netzwerke zugreifen
- [ ] Automatisierungsskripten und der [CLI / MCP](/de/docs/guide/cli-mcp) [API-Token](/de/docs/reference/api) mit minimalen Rechten und Ablaufdatum geben und das Administratorpasswort nicht weitergeben
- [ ] Eine Verschlüsselungs-Passphrase für [Konfigurationssicherungen](/de/docs/modules/backup) setzen und geplante Sicherungen aktivieren
- [ ] In der [Benachrichtigungszentrale](/de/docs/modules/notify) Push-Benachrichtigungen für Ereignisse wie Anmeldungen, Zertifikate und fehlgeschlagene Sicherungen einrichten

## Öffentlich erreichbare Dienste

- Wenn Sie LAN-Dienste über [Web-Dienste](/de/docs/modules/web) veröffentlichen, aktivieren Sie HTTPS und kombinieren Sie es mit Basic Auth, Web-Authentifizierung oder Anmeldung über Drittanbieter;
- Aktivieren Sie die [WAF](/de/docs/modules/waf) auf öffentlichen Sites: Starten Sie im Modus „nur erkennen“, um Fehlalarme zu beobachten, und wechseln Sie dann auf „blockieren“;
- Beschränken Sie Quellen mit IP-Allow- und Blocklisten, GeoIP-Länderfilter und automatischen Sperren bei Authentifizierungsfehlern.

## Docker

Die Einbindung von `docker.sock` entspricht Root-Rechten auf dem Host. Entfernen Sie die Einbindung, wenn Sie die [Docker-Verwaltung](/de/docs/modules/docker) nicht brauchen.

## Web-Terminal

Die lokale Shell ist standardmäßig deaktiviert. Aktivieren Sie sie nur, wenn sie wirklich nötig ist, und stellen Sie sicher, dass die Zwei-Faktor-Authentifizierung für die Verwaltungsoberfläche eingeschaltet ist.

## Konfigurationssicherung

Legen Sie das Konfigurationsverzeichnis in einem privaten Git-Repository ab oder sichern Sie es regelmäßig. `ssl.json` enthält private Zertifikatsschlüssel, bewahren Sie die Datei daher sorgfältig auf.
