# Konventionen der Web-Terminal-API

Alle Endpunkte liegen unter `/api/webterminal/` und erfordern eine Anmeldung. Fehler werden immer als `{"error": "..."}` zurückgegeben.

## 1. Konfigurationsstruktur (die webterminal-Datei in `conf`)

```jsonc
{
  "settings": {
    "allowLocalShell": false,
    "maxSessions": 20,
    "idleTimeout": 0,          // Minuten; eine Sitzung ohne Ein- oder Ausgabe über diese Zeit wird automatisch beendet; 0 bedeutet unbegrenzt
    "bufferSize": 256,         // KB; Wiedergabepuffer der Ausgabe pro Sitzung, wird beim Wiederverbinden abgespielt
    "heartbeatInterval": 30,   // Sekunden; WebSocket-Heartbeat (Server-Ping / Frontend-Ping)
    "sessionKeepAlive": 0      // Minuten; wie lange eine Sitzung nach dem Trennen des Browsers erhalten bleibt; 0 beendet sie sofort beim Trennen
  },
  "connections": [Connection],
  "shortcuts": [{ "name": "Festplattenbelegung anzeigen", "command": "df -h", "sendEnter": true }],
  "splitLayout": null,         // vom Frontend angepasstes Layout der geteilten Ansicht; das Backend speichert und liefert es unverändert (json.RawMessage)
  "security": {
    "agreementAccepted": false,
    "twoFAEnable": false,
    "twoFAKey": ""             // TOTP-Geheimnis; in der API-Ausgabe maskiert
  },
  "credentials": [Credential]  // SSH-Anmeldevorlagen
}
```

Connection (`type` ist `ssh` / `telnet` / `local`):

```jsonc
{
  "key": "", "name": "", "type": "ssh", "remark": "", "enable": true,
  "quickAccessDirs": ["/etc", "/var/log"],      // SFTP-Schnellzugriffsverzeichnisse
  "ssh": { "host", "port", "username", "authType": "password|key|credential",
           "credentialKey": "",  // Key der Anmeldevorlage bei authType=credential; username darf dann leer sein und fällt auf den Standard der Vorlage zurück
           "password", "privateKey",
           "keyFile": "",        // Pfad zu einer privaten Schlüsseldatei auf dem Server selbst; verwendet bei authType=key und leerem privateKey
           "passphrase", "hostKey", "keepAlive", "workDir": "", "timeout": 15,
           "proxy": Proxy },
  "telnet": { "host", "port": 23, "username", "password", "timeout": 15, "proxy": Proxy },
  "local": { "shell", "workDir", "env": {"K":"V"} }
}
Proxy = { "enabled": false, "type": "socks5|http", "host", "port", "username", "password" }
```

Credential (Anmeldevorlage, kann von mehreren SSH-Verbindungen referenziert werden):

```jsonc
{ "key": "", "name": "", "authType": "password|key", "username": "", // optional, verwendet, wenn die Verbindung keinen Benutzernamen hat
  "password", "privateKey", "keyFile", "passphrase", "remark": "", "enable": true }
```

Passwörter, private Schlüssel, Passphrasen privater Schlüssel und Proxy-Passwörter werden in der Ausgabe mit `crud.Mask` maskiert und beim Aktualisieren mit `crud.KeepSecret` beibehalten.

## 2. Verbindungsverwaltung

| Methode | Pfad | Beschreibung |
|---|---|---|
| GET/POST | `/connections` | Liste / anlegen |
| PUT/DELETE | `/connections/{key}` | Ändern / löschen |
| PUT | `/connections-order` | Sortieren `{"keys":[...]}` |
| POST | `/connections/{key}/test` | SSH- / Telnet-Verbindung testen |
| POST | `/connections/test` | Eine nicht gespeicherte Verbindung testen (der Body ist eine Connection; gespeicherte Geheimnisse werden übernommen, indem das Feld `key` auf die ursprüngliche Verbindung zeigt) |
| DELETE | `/connections/{key}/hostkey` | Host-Fingerabdruck zurücksetzen |
| PUT | `/connections/{key}/quickaccess` | Body `{"dirs":["/a","/b"]}` |
| GET/POST | `/credentials` | Anmeldevorlagen auflisten / anlegen |
| PUT/DELETE | `/credentials/{key}` | Ändern / löschen (liefert 409 mit den Verbindungsnamen, solange noch referenziert) |
| PUT | `/credentials-order` | Vorlagen sortieren `{"keys":[...]}` |
| GET | `/shells` | Liste der auf diesem Rechner verfügbaren Shells `["/bin/bash", ...]` (liest /etc/shells und filtert auf vorhandene) |

Ergänzende Hinweise (maßgeblich ist die Implementierung):
- `POST /connections/test` und `POST /connections/{key}/test` liefern 200 mit `{ok, error?, serverVersion?, newHostKey?, latencyMs}`; der Typ `local` liefert 400. Bei Telnet wird nur die TCP-Erreichbarkeit geprüft (auch über den Proxy).
- `PUT /connections/{key}/quickaccess` liefert `{ok, dirs}` (Leerraum und Duplikate entfernt). Wird eine Verbindung ohne `quickAccessDirs` bearbeitet, bleibt der alte Wert erhalten.
- `ssh.timeout` / `telnet.timeout` akzeptieren 1 bis 120 Sekunden, Standard 15. `ssh.workDir` wird nach der Anmeldung per `cd` angewendet.
- Bei Verbindungen mit `authType=credential` werden beim Speichern das eigene Passwort, der private Schlüssel, die Schlüsseldatei und die Passphrase geleert; die Zugangsdaten werden beim Verbinden aus der Vorlage aufgelöst. Das Ändern einer Vorlage trennt gepoolte SSH-Verbindungen, die sie referenzieren.
- Das automatische Ausfüllen beim Telnet-Login wird jeweils einmal versucht, und nur wenn Eingabeaufforderungen wie `login:` / `username:` / `password:` erkannt werden.

## 3. Terminal-Sitzungen (persistente Sitzungen, die sich nach einer Trennung wiederverbinden lassen)

- `GET /connections/{key}/ws?cols=&rows=`: legt eine neue Sitzung an und wechselt zu WebSocket.
- `GET /sessions/{id}/ws?cols=&rows=`: verbindet sich mit einer bestehenden Sitzung, spielt zuerst den Puffer ab und leitet dann in Echtzeit weiter. Mehrere Clients können gleichzeitig mit derselben Sitzung verbunden sein (Eingabe wird geteilt, Ausgabe an alle gesendet).

WebSocket-Protokoll:
- Server → Browser: **Binärframes** enthalten die Terminalausgabe; **Textframes** enthalten JSON:
  `{"t":"s","d":"<sessionId>"}` Sitzung hergestellt (wird nach dem Anlegen und nach dem Verbinden jeweils einmal gesendet);
  `{"t":"n","d":"Hinweis"}` Hinweis; `{"t":"x","d":"Grund"}` Sitzung beendet; `{"t":"p"}` Heartbeat-Antwort.
- Browser → Server: Text-JSON: `{"t":"i","d":"Eingabe"}`, `{"t":"r","c":cols,"r":rows}`, `{"t":"p"}` Heartbeat.
- Wenn der Browser-WebSocket getrennt wird: Bei `sessionKeepAlive > 0` bleibt die Sitzung erhalten (detached) und wird nach Ablauf der Zeit geschlossen, wenn sich niemand verbindet; bei 0 wird sie sofort geschlossen.
- Beendet sich der Sitzungsprozess selbst, wird eine `x`-Nachricht gesendet, der WebSocket geschlossen und die Sitzung aus der Sitzungstabelle entfernt.

- Das Anlegen einer Sitzung über `maxSessions` hinaus liefert HTTP 429; ein Verbindungsfehler wird über einen `n`-Hinweisframe gemeldet, danach wird der WebSocket geschlossen.
- `bytesIn` ist die Anzahl der Bytes aus Browser-Eingaben, `bytesOut` die Anzahl der Bytes der Terminalausgabe und `attached` die Anzahl der aktuell verbundenen Clients (eine Zahl); Zeitfelder sind RFC3339.

REST:

| Methode | Pfad | Beschreibung |
|---|---|---|
| GET | `/sessions` | `[{id, connKey, connName, type, remark, createdAt, lastActive, attached, cols, rows, bytesIn, bytesOut}]` |
| GET | `/sessions/{id}` | Eine einzelne Sitzung |
| PUT | `/sessions/{id}/remark` | `{"remark":""}` |
| GET | `/sessions/{id}/stats` | `{bytesIn, bytesOut, durationSec, idleSec, attached, cols, rows}` |
| DELETE | `/sessions/{id}` | Sitzung beenden |

## 4. Einstellungen, Schnellbefehle und geteilte Ansicht

| Methode | Pfad | Beschreibung |
|---|---|---|
| GET/PUT | `/settings` | `{settings, activeSessions}` / der Body ist Settings |
| GET/PUT | `/globalshortcuts` | Array von Schnellbefehlen |
| GET/PUT/DELETE | `/splitlayout` | Beliebiges JSON-Layout |

## 5. Sicherheit (Nutzungsvereinbarung + Zwei-Faktor-Prüfung)

| Methode | Pfad | Beschreibung |
|---|---|---|
| GET | `/security` | `{agreementAccepted, twoFAEnable, verified}`; `verified` gibt an, dass das aktuelle Anmeldetoken die Zwei-Faktor-Prüfung bestanden hat |
| POST | `/security/agreement` | Vereinbarung akzeptieren |
| POST | `/security/2fa/setup` | Ein vorläufiges Geheimnis erzeugen `{secret, url}` (nicht gespeichert) |
| POST | `/security/2fa/enable` | `{secret, code}` aktiviert 2FA, sobald die Prüfung bestanden ist |
| POST | `/security/2fa/disable` | `{code}` deaktiviert 2FA, sobald die Prüfung bestanden ist |
| POST | `/security/verify2fa` | `{code}`; nach bestandener Prüfung gilt das aktuelle Anmeldetoken 30 Minuten lang als verifiziert |
| GET | `/security/check2fa` | `{need: bool}`: ob noch eine Prüfung nötig ist |

Nach 5 aufeinanderfolgenden falschen 2FA-Codes wird das Anmeldetoken für 1 Minute gesperrt (HTTP 429). `2fa/setup` liefert 400, wenn 2FA bereits aktiv ist (zuerst deaktivieren). Nach erfolgreichem `2fa/enable` gilt das aktuelle Token als verifiziert.
Zugangsprüfung: Das Aktivieren der lokalen Shell, das Anlegen einer Sitzung, das Verbinden mit einer Sitzung und alle SFTP-Endpunkte liefern 403 `{"error":"...","needAgreement":true}`, solange die Vereinbarung nicht akzeptiert ist,
und 403 `{"error":"...","need2fa":true}`, wenn 2FA aktiv ist und das aktuelle Token nicht verifiziert wurde. „Aktivieren der lokalen Shell“ meint hier ein `PUT /settings`, das `allowLocalShell` von false auf true ändert. Auf der Go-Seite rufen all diese Stellen `m.requireAccess(w, r) bool` auf.

## 6. SFTP-Dateiverwaltung

Der Pfadparameter `{key}` ist der **Key der Verbindung**. SSH-Verbindungen nutzen SFTP (jede Anfrage öffnet einen SFTP-Subsystemkanal auf einer wiederverwendeten SSH-Verbindung); `local`-Verbindungen arbeiten direkt auf dem Dateisystem des Servers (über dieselben Endpunkte).
Telnet und andere Typen liefern 400; eine nicht vorhandene Verbindung liefert 404. Das Backend verwendet pro Verbindung eine SSH-Verbindung wieder und schließt sie nach 5 Minuten Leerlauf (`m.acquireSSH`).
Eine `local`-Verbindung setzt voraus, dass „Lokale Shell erlauben“ in den Einstellungen aktiviert ist, sonst liefert sie 403 (lokale Dateiberechtigungen entsprechen einer lokalen Shell). Ein Fehler beim SSH-Verbindungsaufbau liefert 502.

Allgemeine Konventionen:
- Jedes `path` / `from` / `to` / `dest` muss ein absoluter Pfad sein, der mit `/` beginnt (ist `path` bei `list` leer, wird das Standardverzeichnis verwendet: bei SSH das Anmeldeverzeichnis, bei local das workDir der Verbindung oder das Home-Verzeichnis des Benutzers). Relative Pfade liefern 400.
- Fehler sind immer `{"error":"Meldung"}`: nicht gefunden 404, keine Berechtigung 403, zu groß 413, nicht unterstützter Typ 415, Ziel existiert bereits 409 (in diesem Abschnitt ergänzt), ungültige Parameter 400, alles andere 500.
- Erfolgreiche schreibende Endpunkte liefern `{"ok":true, ...}`.

| Methode | Pfad | Beschreibung |
|---|---|---|
| GET | `/sftp/{key}/list?path=` | `{path, parent, entries:[{name, path, isDir, isLink, size, mode, modeStr, modTime, owner?}]}`; Verzeichnisse zuerst, dann nach Namen ohne Beachtung der Groß-/Kleinschreibung sortiert. `mode` ist ein oktaler Berechtigungsstring wie `"755"`, `modeStr` hat die Form `drwxr-xr-x`, `modTime` ist RFC3339, und `owner` ist bei local der Benutzername und bei SSH die numerische uid. Symbolische Links haben `isLink=true` sowie `isDir=true`, wenn sie auf ein Verzeichnis zeigen |
| GET | `/sftp/{key}/read?path=&download=1` | Ohne download: Textvorschau, liefert `{path, name, size, modTime, content}`; über 1MB liefert 413, binäre Inhalte (mit NUL oder kein UTF-8) liefern 415. Mit download=1: Streaming-Download (`application/octet-stream` + `Content-Disposition: attachment`) mit Unterstützung für `Range` und `If-Modified-Since` |
| POST | `/sftp/{key}/write` | `{path, content}` speichert Text (überschreibt; ein Verzeichnis als Ziel liefert 400) → `{ok, path, size}` |
| POST | `/sftp/{key}/upload` | Multipart-Streaming-Schreiben: `path` (das Zielverzeichnis, das bereits existieren und **vor files stehen muss**), `files` (mehrfach möglich) und optional `relativePaths` (einer pro Datei; wird in der Reihenfolge des Auftretens den folgenden files zugeordnet und muss vor der jeweiligen Datei stehen; ein JSON-Array-String wird ebenfalls akzeptiert). Ohne `relativePaths` wird der Dateiname unverändert verwendet (er kann einen Unterpfad enthalten, wenn ein Browser ein Verzeichnis hochlädt). Gleichnamige Dateien werden überschrieben und fehlende Zwischenverzeichnisse automatisch angelegt; Pfade mit `..` liefern 400, sodass nie etwas außerhalb des Zielverzeichnisses geschrieben wird → `{ok, count, files:[absolute Pfade]}` |
| POST | `/sftp/{key}/mkdir` | `{path}` legt ein Verzeichnis an (einschließlich fehlender übergeordneter Verzeichnisse); 409, wenn es bereits existiert |
| POST | `/sftp/{key}/touch` | `{path}` legt eine leere Datei an (einschließlich fehlender übergeordneter Verzeichnisse); 409, wenn sie bereits existiert |
| POST | `/sftp/{key}/rename` | `{from, to}`; 409, wenn `to` bereits existiert, 400 beim Verschieben eines Verzeichnisses in sich selbst, 403 beim Umbenennen des Wurzelverzeichnisses; bei local über Gerätegrenzen hinweg wird automatisch auf Kopieren und anschließendes Löschen ausgewichen |
| POST | `/sftp/{key}/copy` | `{from:[...], to}`; `to` ist das Ziel**verzeichnis**, und jedes Element wird nach `to/ursprünglicher-Name` kopiert (rekursiv; symbolische Links werden als Links kopiert; Sonderdateien wie Geräte und Pipes werden ignoriert). Bei einer Namenskollision wird die Kopie **automatisch umbenannt** in `name copy.ext`, `name copy 2.ext` usw., ohne Fehler; das Kopieren eines Verzeichnisses in sich selbst liefert 400 → `{ok, paths:[neue Pfade]}` |
| POST | `/sftp/{key}/chmod` | `{path, mode:"755", recursive}`; `mode` ist ein oktaler String von 0 bis 7777; rekursiv werden symbolische Links nicht verfolgt, und Dateien und Verzeichnisse erhalten alle dieselbe Berechtigung |
| DELETE | `/sftp/{key}/remove` | `{paths:[...]}` (JSON-Body; rekursives Löschen; bei symbolischen Links wird nur der Link entfernt); ist ein Pfad `/`, wird 403 geliefert und nichts gelöscht → `{ok, count}` |
| POST | `/sftp/{key}/compress` | `{paths:[...], dest, format:"zip|tar.gz"}` (format ist standardmäßig zip; `tgz` wird als tar.gz behandelt). `dest` ist der vollständige Pfad des Ausgabearchivs; ist `dest` ein bestehendes Verzeichnis oder endet es auf `/`, wird die Datei darin automatisch benannt (bei einem Element nach dessen Namen, bei mehreren nach dem Namen des übergeordneten Verzeichnisses, plus `.zip` / `.tar.gz`). Jedes Element wird unter seinem eigenen Namen zu einem Eintrag auf oberster Ebene des Archivs; gleichnamige Elemente liefern 400, eine bestehende Ausgabedatei liefert 409, und die Ausgabedatei wird nie in sich selbst gepackt → `{ok, path}` |
| POST | `/sftp/{key}/decompress` | `{path, dest}` unterstützt zip / tar / tar.gz / tgz (erkannt am Suffix; alles andere liefert 415). Ist `dest` leer, wird in das Verzeichnis des Archivs entpackt und es bei Bedarf angelegt; gleichnamige Dateien werden überschrieben. Ein Archiv mit einem `..`-Eintrag wird vollständig abgelehnt (400, Schutz vor Zip-Slip); Einträge, die keine regulären Dateien sind, etwa symbolische Links und Hardlinks, werden übersprungen; von den Dateiberechtigungen bleiben nur die rwx-Bits erhalten → `{ok, dest, count}` |
| GET | `/sftp/{key}/preview-archive?path=` | Liste der Archiveinträge `[{name,size,isDir}]` (höchstens 10000 Einträge; `name` ohne abschließendes `/`) |

Komprimieren und Entpacken erledigt Go vollständig über die Dateisystemschnittstelle (os für local, SFTP-Lesen und -Schreiben für SSH), ohne Abhängigkeit von Befehlen auf der Gegenseite.

## 7. Sonstiges

- Logs: Modul-Logs erscheinen bereits auf der globalen Log-Seite, ein eigener Endpunkt ist daher nicht nötig.
- Lucky-Import: ergänzt telnet, proxy, keyFile, workDir, timeout, env, quickAccessDirs, sessionConfig, globalShortcuts
  und securityConfig (Nutzungsvereinbarung und 2FA-Geheimnis werden mit übernommen).
