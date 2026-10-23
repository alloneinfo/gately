# Datei- und Betriebstools

## WebDAV

Konfiguration unter „WebDAV“ in der Seitenleistengruppe „Netzwerkspeicher“.

| Feld | Standard | Beschreibung |
| --- | --- | --- |
| `enable` | Aus | Dienst aktivieren; bei aktiviertem Dienst ist mindestens ein Benutzer erforderlich |
| `listenIp` | Leer | Listen-Adresse; leer lassen, um auf allen Adressen zu lauschen |
| `port` | 18800 | Listen-Port |
| `tls` | Aus | HTTPS aktivieren; das Zertifikat liefert das Modul [SSL-Zertifikate](/de/docs/modules/ssl) per SNI |
| `users` | Keine | Mehrere Benutzer. Jeder hat `name`, `password`, `root` (ein absoluter Verzeichnispfad) und `readOnly` oder bindet über `mounts` mehrere Verzeichnisse ein |

Jeder Benutzer kann auf ein Verzeichnis abgebildet werden oder mehrere Verzeichnisse einbinden:

- **Einzelnes Verzeichnis**: `root` und `readOnly` setzen. Nach der Anmeldung ist das Wurzelverzeichnis dieses Verzeichnis.
- **Mehrere Verzeichnisse**: Klicken Sie in der Oberfläche auf „Mehrere Verzeichnisse einbinden“ oder tragen Sie mehrere Einträge in `mounts` ein. Jeder Eintrag hat `name` (Anzeigename, darf kein `/` enthalten und muss pro Benutzer eindeutig sein), `path` (einen absoluten Pfad) und `readOnly`. Nach der Anmeldung listet das Wurzelverzeichnis die Verzeichnisse nach Namen auf. Das Wurzelverzeichnis selbst ist nicht beschreibbar, und Dateien lassen sich nicht zwischen verschiedenen Verzeichnissen verschieben.

Bei der Migration von Lucky wird die Einbindungsliste eines Benutzers in mehrere Verzeichnisse umgewandelt, und Benutzer, die in Lucky mehrfach mit gleichem Namen und Passwort vorkommen, werden zu einem Benutzer zusammengeführt.

## FTP

Konfiguration unter „FTP-Dienst“ in der Gruppe „Netzwerkspeicher“. FTP und FTPS werden unterstützt.

| Feld | Standard | Beschreibung |
| --- | --- | --- |
| `enable` | Aus | Dienst aktivieren; bei aktiviertem Dienst ist mindestens ein Benutzer erforderlich |
| `listenIp` | Leer | Listen-Adresse; leer lassen, um auf allen Adressen zu lauschen |
| `port` | 2121 | Listen-Port |
| `passiveStart` / `passiveEnd` | 0 | Bereich der Datenports im passiven Modus. Sind beide 0, vergibt das System die Ports zufällig; sind sie gesetzt, muss 1 ≤ start ≤ end ≤ 65535 gelten, und der Bereich muss in Firewall und Portzuordnungen freigegeben sein |
| `publicHost` | Leer | Im passiven Modus nach außen gemeldete IP. Hinter NAT tragen Sie die öffentliche oder weitergeleitete IP ein. Es werden nur IP-Adressen unterstützt; leer lassen, um automatisch die Verbindungsadresse zu verwenden |
| `tlsMode` | 0 | 0 erlaubt Klartext und FTPS, 1 verlangt explizites FTPS (`AUTH TLS`), 2 ist implizites FTPS (ab dem Verbindungsaufbau verschlüsselt). Bei einem Wert ungleich 0 liefert das Modul [SSL-Zertifikate](/de/docs/modules/ssl) das Zertifikat per SNI |
| `idleTimeout` | 900 | Sekunden Leerlauf, bevor eine Verbindung geschlossen wird |
| `connectionTimeout` | 30 | Timeout in Sekunden für den Aufbau einer Datenverbindung |
| `disableActiveMode` | Aus | Nur passiven Modus verwenden, was für NAT-Umgebungen besser geeignet ist |
| `users` | Keine | Mehrere Benutzer mit denselben Feldern wie bei WebDAV: `name`, `password`, `root`, `readOnly` oder `mounts`; Benutzernamen müssen eindeutig sein |

Dienstparameter und Benutzer lassen sich aus der FTP-Konfiguration von Lucky importieren (siehe [Migration von Lucky](/de/docs/guide/migration)).

## Speicher

Fügen Sie Einbindungen unter „Speicher“ in der Gruppe „Netzwerkspeicher“ hinzu und verwalten Sie Dateien online auf der Seite.

| Feld | Beschreibung |
| --- | --- |
| `name`, `remark`, `enable` | Name, Bemerkung und Aktivierungsschalter |
| `type` | `local` (lokales Verzeichnis), `webdav` (entferntes WebDAV) oder `smb` (SMB-Freigabe) |
| `path` | Lokales Verzeichnis, muss ein vorhandener absoluter Pfad sein (für `local`) |
| `url`, `username`, `password` | Entfernte Adresse und Konto; das Passwort wird in der Oberfläche maskiert (für `webdav` und `smb`). Eine WebDAV-Adresse beginnt mit `http://` oder `https://`; eine SMB-Adresse hat die Form `smb://host[:port]/freigabe[/unterverzeichnis]`, unterstützt wird SMB2/3, ohne Benutzername wird als Gast verbunden |
| `skipVerify` | Prüfung des HTTPS-Zertifikats überspringen (für `webdav`) |
| `domain` | Domäne oder Arbeitsgruppe für die SMB-Anmeldung, meist leer (für `smb`) |
| `readOnly` | Schreibgeschützt einbinden; der Dateimanager lehnt jeden Schreibvorgang ab |

Lokale Verzeichnisse und WebDAV-Speicher lassen sich aus der Speicherverwaltung von Lucky importieren. Alibaba Cloud Drive und Systemeinbindungen werden nicht migriert.

Der Online-Dateimanager unterstützt das Durchsuchen von Verzeichnissen, das Herunterladen von Dateien, Streaming-Uploads (die ganze Datei wird nicht in den Arbeitsspeicher geladen), das Anlegen von Verzeichnissen, Umbenennen und Löschen. Eine Einbindung kann als Zielort einer Sicherungsaufgabe dienen: Sicherungsdateien werden in das angegebene Verzeichnis der Einbindung geschrieben, und alte Dateien werden gemäß der Aufbewahrungsrichtlinie bereinigt.

## Wake-on-LAN

Verwalten Sie Geräte unter „Wake-on-LAN“ und senden Sie ihnen Magic Packets (6 Bytes `0xFF`, gefolgt von der 16-fach wiederholten MAC).

| Feld | Standard | Beschreibung |
| --- | --- | --- |
| `name` | Keiner | Gerätename, Pflicht |
| `macs` | Keine | MAC-Adressen, eine pro Zeile, mindestens eine; für ein Gerät können Sie mehrere eintragen |
| `broadcastIps` | Leer | IPv4-Broadcast-Adressen. Ist das Feld leer, werden Pakete an `255.255.255.255` und die Broadcast-Adresse jeder aktivierten Netzwerkschnittstelle gesendet |
| `port` | 9 | UDP-Zielport |
| `repeat` | 3 | Wie oft jede MAC gesendet wird, bis zu 20 |
| `checkAddr` | Leer | Adresse für die Online-Prüfung in der Form `host:port`, zum Beispiel `192.168.1.10:3389`; leer lassen, um die Prüfung zu überspringen |
| `remark` | Leer | Bemerkung |

Klicken Sie in der Geräteliste auf „Aufwecken“, um das Magic Packet zu senden; die Seite zeigt anhand von `checkAddr`, ob das Gerät online ist.

## Geplante Aufgaben

Führen Sie Befehle aus oder senden Sie HTTP-Anfragen nach einem Cron-Zeitplan.

- `spec` unterstützt Standardausdrücke mit 5 Feldern, ein optionales Sekundenfeld (6 Felder, etwa `*/30 * * * * *`) sowie Deskriptoren wie `@every 10m` und `@daily`;
- Bei `type` gleich `command` läuft `command` über `sh -c` (unter Windows eine temporäre `.bat`), und die Umgebung erhält `taskName` sowie eine gleichnamige Variable in Großbuchstaben mit dem Präfix `GATELY_`;
- Bei `type` gleich `http` wird eine Anfrage gemäß der `http`-Konfiguration gesendet, mit den Feldern `url`, `method` (GET, POST, PUT, PATCH, DELETE), `headers` (jeweils als `Key: Value`), `body`, `successContent` (der Antwort-Body muss einen seiner Einträge enthalten, damit es als Erfolg zählt; ist das Feld leer, wird nur ein 2xx-Status geprüft), `timeout` und `insecureSkipVerify`;
- Aufgaben lassen sich aktivieren oder deaktivieren, die Seite zeigt die nächste Laufzeit, und eine Aufgabe kann von Hand sofort ausgeführt werden;
- Ist der vorherige Lauf derselben Aufgabe noch nicht beendet, wird der neue Auslöser übersprungen;
- Jede Aufgabe behält ihre letzten 20 Laufprotokolle mit Startzeit, Dauer, Erfolg oder Fehler, ob manuell ausgelöst, sowie der Ausgabe (eine Zusammenfassung von bis zu 4000 Zeichen);
- Das Erstellungsformular validiert und parst den Ausdruck.

## Übersicht und Systemprotokolle

- Die Seite „Übersicht“ zeigt Host-Informationen: CPU-Modell, Auslastung und Last, Arbeitsspeicher, Datenträger, Adressen der Netzwerkschnittstellen und Live-Raten, öffentliche IPv4 und IPv6 (zum Kopieren anklicken), den Dienststatus jedes Moduls sowie für den Gately-Prozess Version, Commit, Build-Zeit, Startzeit, PID, CPU, resident Memory, Go-Heap und Anzahl der Goroutinen;
- Die Seite „Systemprotokolle“ sammelt die Logs aller Module, einschließlich der Laufzeitprotokolle von Modulen wie Docker, FTP und geplanten Aufgaben.

## Designs

Die Oberfläche bietet mehrere Designs: Emerald, Ocean, Fresh, Sakura, Sunset, Dark, Midnight, Punk sowie „System folgen“ (Dark, wenn das System dunkel ist, sonst Emerald). Die Wahl des Designs wird lokal im Browser gespeichert.
