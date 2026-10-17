# Sicherheitsmodell & Datenschutzversprechen

Gately läuft am Netzwerkrand und verwaltet DNS-Anbieter-Schlüssel, private Zertifikatsschlüssel und weitere Zugangsdaten. Sicherheit und Transparenz sind daher die wichtigsten Entwurfsziele.

## Datenschutzversprechen

- **Standardmäßig keine ausgehenden Verbindungen**: keine Verbindungen zu Projektservern, keine Übermittlung von Geräteinformationen, kein Konfigurations-Push aus der Ferne.
- **Sichtbare, änderbare ausgehende Adressen**: Abfrage der öffentlichen IP, STUN-Server und ähnliche Adressen werden in der Oberfläche angezeigt und lassen sich ändern oder abschalten.
- **Selbst verwaltete Anmeldung über Drittanbieter**: Die Anmeldung über GitHub, Google, QQ und OIDC nutzt OAuth-Apps, die Sie selbst registrieren, ohne Relay dazwischen.
- **Selbst verwaltete Upgrades**: Das Programm ersetzt seine Binärdatei nie im Hintergrund; Sie entscheiden, wann Sie upgraden.

## Datenschutz

- Die Konfiguration ist einfaches JSON mit Dateirechten 0600 und lässt sich daher prüfen und zurückrollen.
- Sensible Felder werden in der API-Ausgabe maskiert.
- Das cloudflared-Token wird über eine Umgebungsvariable übergeben und erscheint nicht in der Prozessliste; Token und Schlüssel von frp und Tailscale werden mit Rechten 0600 gespeichert.

## Authentifizierung und Zugriffskontrolle

- bcrypt-Passwort-Hashing und HMAC-signierte Sitzungstoken;
- Zwei-Faktor-Authentifizierung (TOTP, gegen die Testvektoren aus RFC 6238 verifiziert);
- Ratenbegrenzung bei der Anmeldung;
- Sicherer Einstiegspfad, Zugriff nur aus dem LAN und HTTPS für die Verwaltungsoberfläche;
- Multi-Token-API: Jedes Token hat Lese-/Schreibzugriff nur auf die bei der Erstellung gewählten Module, kann ein Ablaufdatum haben und speichert die Quell-IP seiner letzten Nutzung. Endpunkte für Konto, Zwei-Faktor-Authentifizierung, Token-Verwaltung sowie Sicherung, Wiederherstellung, Neustart und Systemeinstellungen akzeptieren nur Anmeldesitzungen. Siehe [API-Token & OpenAPI](/de/docs/reference/api);
- Zugriffskontrolle für Sites: automatische Sperren bei Authentifizierungsfehlern, GeoIP-Länderfilter und Pfadregeln. Siehe [Web-Dienste](/de/docs/modules/web);
- Konfigurationssicherungen unterstützen Verschlüsselung (scrypt + AES-GCM). Siehe [Konfigurationssicherung](/de/docs/modules/backup).

## Standardwerte für Funktionen mit hohem Risiko

Das Terminal mit lokaler Shell ist standardmäßig deaktiviert. Das SSH-Terminal verwendet eine Trust-on-first-use-Richtlinie und speichert die Fingerabdrücke der Host-Schlüssel.

## Sicherheitsprobleme melden

Melden Sie Schwachstellen bitte vertraulich über die Security-Seite des Repositorys und legen Sie Details nicht in öffentlichen Issues offen: <https://github.com/wangwanjie/gately/security>

Weitere praktische Empfehlungen finden Sie im [Härtungsleitfaden](/de/docs/guide/hardening).
