# Web-Dienste

Stellen Sie Reverse Proxy, Weiterleitungen, statische Dateien und festen Text pro Domain bereit. Das ist der einheitliche Web-Einstiegspunkt von Gately. Menüpunkt: Seitenleiste „Web → Web-Dienste“.

## Aufbau: Listener, Sites und Standard-Site

Die Konfiguration hat drei Ebenen:

1. Listener: ein Listen-Port. Zu den Feldern gehören Netzwerk (`tcp` Dual-Stack, `tcp4`, `tcp6`), Listen-Adresse (leer für alle Adressen), Port, HTTPS-Schalter, minimale TLS-Version (1.0 / 1.1 / 1.2 / 1.3) und ob das PROXY-Protokoll akzeptiert wird.
2. Site: eine Site unter einem Listener, die über die Domain zugeordnet wird. Domains können eine exakte Domain, `*.example.com` (trifft Subdomains beliebiger Tiefe) oder `*` (trifft alles) sein. Die Zuordnungspriorität lautet: exakte Domain, Wildcard, dann `*`.
3. Standard-Site: wird verwendet, wenn keine Site passt. Ist sie nicht aktiviert, wird 404 zurückgegeben.

## Site-Typen

| Typ | Beschreibung |
| --- | --- |
| Reverse Proxy | Leitet an ein oder mehrere Backends weiter. Backend-Adressen haben die Form `http(s)://host:port`, WebSocket wird unterstützt |
| Weiterleitung | Leitet mit Statuscode 301 / 302 / 307 / 308 (Standard 302) auf eine vollständige URL um, wahlweise unter Beibehaltung von ursprünglichem Pfad und Query-String |
| Statische Dateien | Das Verzeichnis muss ein absoluter Pfad sein. Ist „Verzeichnisauflistung“ aus, liefert eine Verzeichnisanfrage nur dann eine Antwort, wenn `index.html` existiert |
| Fester Text | Eigener Inhalt, `Content-Type` (Standard `text/plain; charset=utf-8`) und Statuscode (100-599, Standard 200) |

Weitere Optionen für Reverse-Proxy-Sites:

- Backend-Host verwenden: Der Host-Header wird auf die Backend-Adresse geändert; ist die Option aus, bleibt der Host der Client-Anfrage erhalten;
- Prüfung des Backend-Zertifikats überspringen, Keep-Alive deaktivieren;
- PROXY-Protokoll an das Backend senden (v2-Header, reicht die echte Client-IP durch, setzt Unterstützung durch das Backend voraus);
- `X-Forwarded-For` / `X-Forwarded-Proto` / `X-Forwarded-Host` hinzufügen.

Sites jedes Typs können ein Zugriffslog, Cross-Origin (CORS) sowie eigene Request- und Response-Header festlegen (je Zeile ein `Key: Value`; ein leerer Value entfernt den Header) und auf eine Instanz der [WAF-Firewall](/de/docs/modules/waf) verweisen (existiert die Instanz nicht oder ist sie deaktiviert, werden Anfragen durchgelassen).

## Transport und Protokolle

- HTTPS: nutzt ein Zertifikat aus dem Modul [SSL-Zertifikate](/de/docs/modules/ssl), das die Domain abdeckt;
- HTTP/3 (QUIC): wird in einem Listener mit aktiviertem HTTPS eingeschaltet; HTTP/3 wird dann zusätzlich per UDP auf derselben Portnummer bereitgestellt und Browsern über den Antwort-Header `Alt-Svc` angekündigt. Der UDP-Port muss in Firewall und Portweiterleitung freigegeben sein;
- WebSocket: wird vom Reverse Proxy automatisch unterstützt;
- PROXY-Protokoll: Ein Listener kann PROXY-Header akzeptieren, die vorgelagerte Komponenten (wie haproxy und frp) senden. Er vertraut nur den IPs oder Subnetzen unter „Globale Einstellungen → Vertrauenswürdige PROXY-Quellen“; bleibt die Liste leer, wird keiner Quelle vertraut. Der Reverse Proxy kann PROXY-Header an das Backend senden.

## Authentifizierung

Konfigurieren Sie sie auf der Seite „Sicherheit“ des Site-Editors:

- Web-Authentifizierung: Der Browser zeigt eine Anmeldeseite;
- Basic-Authentifizierung: HTTP Basic;
- Benutzer: je Zeile ein `username:password`, das Passwort kann ein bcrypt-Hash sein. Die Basic-Authentifizierung braucht bei Aktivierung mindestens einen Benutzer; die Web-Authentifizierung braucht bei Aktivierung mindestens einen Benutzer oder eine Identität der Anmeldung über Drittanbieter;
- Anmeldung über Drittanbieter: GitHub, Google, QQ und OIDC werden unterstützt, mit Ihren eigenen OAuth-Apps. Erlaubte Identitäten stehen je Zeile in der Form `provider:account`, zum Beispiel `github:octocat` und `google:me@gmail.com`. Ein `#` am Zeilenende fügt einen Kommentar hinzu, und `github:*` bedeutet jedes Konto dieses Anbieters.

Die Anbieter für die Anmeldung über Drittanbieter werden in den „Globalen Einstellungen“ konfiguriert: Tragen Sie Ihre eigene Client-ID / Secret ein und aktivieren Sie den Anbieter, und legen Sie außerdem eine Callback-Adresse fest (sie muss von einem Listener von Gately bedient werden, zum Beispiel `https://auth.example.com`). Die bei GitHub / Google / QQ zu registrierende Callback-URL lautet:

```text
https://auth.example.com/__gately_auth/oauth/callback
```

Die Anmeldung über Drittanbieter für die Verwaltungsoberfläche selbst wird separat unter „Einstellungen → Konto & Sicherheit“ konfiguriert.

## Zugriffskontrolle

### IP-Allowlist und -Blocklist

„IP-Zugriffskontrolle“ auf der Seite „Sicherheit“ der Site kann aus, Allowlist oder Blocklist sein. Die Liste nimmt je Zeile eine IP oder ein Subnetz auf (etwa `192.168.1.0/24`).

### Pfadregeln

Überschreiben Sie unter „Pfadregeln“ auf der Seite „Sicherheit“ die Authentifizierungs- und IP-Einstellungen der Site nach Pfadpräfix:

| Feld | Werte |
| --- | --- |
| `prefix` | Ein Pfadpräfix, das mit `/` beginnt, etwa `/admin` |
| `auth` | Von der Site erben (leer), `off` (keine Authentifizierung), `on` (Authentifizierung erforderlich) |
| `ipMode` / `ipList` | Von der Site erben (leer), `whitelist` oder `blacklist`, mit einer Liste von IPs oder Subnetzen |

Zuordnungsregeln:

- Das längste Präfix gewinnt;
- Die Zuordnung folgt den Grenzen von Pfadsegmenten: `/admin` trifft `/admin` und `/admin/x`, aber nicht `/administrator`;
- Der Anfragepfad wird zuerst normalisiert (`.`, `..` und doppelte Schrägstriche werden entfernt) und dann abgeglichen, daher wird `/public/../admin` als `/admin` behandelt;
- Für dasselbe Präfix lässt sich nur eine Regel konfigurieren.

Die Option „Authentifizierungsbereich“ (`authOptional`): Ist sie angekreuzt, gilt die Authentifizierung nur für Pfade, die in den Regeln als „Erforderlich“ markiert sind, alle anderen Pfade sind öffentlich. Für diese Option muss zuerst die Basic- oder Web-Authentifizierung aktiviert sein.

### Temporäre Freigabelinks

Ist die Basic- oder Web-Authentifizierung aktiviert, können Sie für eine Site Freigabelinks erstellen, damit Besucher ohne Konto zugreifen können:

| Feld | Beschreibung |
| --- | --- |
| Name | Eine Bezeichnung, etwa „Für Alice“ |
| Ablauf | Leer lassen für keinen Ablauf |
| Nutzungen `maxUses` | Maximale Anzahl, wie oft der Link eingelöst werden kann; 0 bedeutet unbegrenzt. Der Zähler wird nach einem Neustart zurückgesetzt |
| Pfadbeschränkung `pathPrefix` | Ist das Feld nicht leer, kann der Link nur auf Pfade unter diesem Präfix zugreifen; es muss mit `/` beginnen |

Ein Besucher ruft `https://domain/beliebiger-pfad?_gs=token` auf, um eine Sitzung einzulösen. Nach erfolgreicher Einlösung setzt der Server ein Cookie für den anmeldefreien Zugang und leitet auf die Adresse ohne den Parameter `_gs` weiter. Die Sitzung gilt höchstens 24 Stunden und nie über den Ablauf des Links hinaus. Ein ungültiges Token wird wie ein normaler Besuch behandelt und verrät den Grund des Fehlschlags nicht. Bereits ausgestellte Sitzungen werden sofort ungültig, sobald der Link gelöscht wird. Das Token wird nach dem Speichern erzeugt (mindestens 16 Zeichen), und der vollständige Link lässt sich in der Oberfläche kopieren.

### Ratenbegrenzung bei Authentifizierungsfehlern und automatische Sperre

- Nach 10 fehlgeschlagenen Authentifizierungen von derselben Quell-IP innerhalb von 15 Minuten werden weitere Versuche eine Zeit lang abgelehnt;
- Tragen Sie unter „Globale Einstellungen → Automatische Sperre → Sperre bei Authentifizierungsfehlern“ eine Anzahl Minuten ein (0 bis 43200); die Quell-IP wird gesperrt, wenn sie 10 Fehlschläge erreicht. 0 bedeutet keine Sperre, nur Ratenbegrenzung;
- Löst eine automatische Sperre aus, wird ein Ereignis `AuthBanned` veröffentlicht, das sich über die [Benachrichtigungszentrale](/de/docs/modules/notify) versenden lässt;
- IPv6-Adressen werden pro /64-Subnetz gezählt und gesperrt.

### Sperrliste

Die Sperrliste in den „Globalen Einstellungen“ zeigt die aktuell gesperrten IPs, den Grund (wiederholte Authentifizierungsfehler oder manuelle Sperre) und den Ablaufzeitpunkt, und jeder Eintrag lässt sich mit „Entsperren“ aufheben. Sie können auch manuell sperren: Geben Sie eine IP und eine Anzahl Stunden ein (1 bis 720, also höchstens 30 Tage); die API verwendet standardmäßig 24 Stunden.

### GeoIP-Länderfilter

Der „Länderfilter“ einer Site hat drei Modi: aus, nur erlauben und verweigern. Geben Sie zweistellige ISO-Ländercodes kommagetrennt ein, etwa `CN, HK, US`. Bei Aktivierung ist mindestens ein Land erforderlich.

- Die Filterung hängt von einer GeoIP-Datenbank ab, die Sie zuerst unter „Globale Einstellungen → GeoIP-Datenbank“ importieren müssen: Laden Sie eine `.mmdb`-Datei hoch oder geben Sie eine Download-URL ein. MaxMind GeoLite2, DB-IP und andere Länder- oder Städte-Datenbanken werden unterstützt; die Upload-Grenze beträgt 129 MB;
- Ist keine Datenbank geladen, hat die Filterung keine Wirkung (alle Besucher passieren) und die Oberfläche zeigt einen Hinweis;
- Adressen privater Netze sind immer erlaubt;
- Im Modus „Verweigern“ werden Adressen erlaubt, deren Land sich in der Datenbank nicht finden lässt.

## Erweiterungen für den Reverse Proxy

Diese Optionen finden Sie auf der Seite „Traffic“ des Site-Editors.

### Backends, Gewichtung und Backup

Die Backend-Liste nimmt je Zeile eine Adresse auf, optional gefolgt von Parametern:

```text
http://10.0.0.2:8080 weight=3
http://10.0.0.3:8080 backup
```

- `weight=N`: Gewichtung, eine ganze Zahl von 1 bis 100, Standard 1;
- `backup`: ein Standby-Backend, das nur verwendet wird, wenn alle Nicht-Backup-Backends nicht verfügbar sind. Sind alle Backends nicht verfügbar, werden alle Nicht-Backup-Backends erneut versucht.

### Strategie der Lastverteilung

| Option in der Oberfläche | `lbPolicy` | Beschreibung |
| --- | --- | --- |
| Round Robin (gewichtet) | Leer | Standard |
| Zufällig (gewichtet) | `random` | |
| IP-Hash (Sticky Sessions) | `ip_hash` | Dieselbe Client-IP geht immer an dasselbe Backend |
| Wenigste Verbindungen | `least_conn` | Wählt den kleinsten Wert aus Verbindungen / Gewichtung |

### Health-Checks

Der Schalter für aktive Prüfungen ist standardmäßig aus. Felder und Standardwerte nach dem Aktivieren:

| Feld | Standard | Bereich |
| --- | --- | --- |
| Modus `mode` | `http` | `http` (fragt den Prüfpfad an; ein Statuscode unter 400 gilt als gesund) oder `tcp` (testet nur die Verbindung) |
| Pfad `path` | `/` | Beginnt mit `/`, nur im http-Modus |
| Intervall `interval` | 10 Sekunden | 1 bis 3600 |
| Timeout `timeout` | 3 Sekunden | 1 bis 60 |
| Aufeinanderfolgende Fehlschläge `fails` | 3 Fehlschläge markieren es als nicht verfügbar | 1 bis 20 |
| Aufeinanderfolgende Erfolge `passes` | 2 Erfolge stellen es wieder her | 1 bis 20 |

Ist das Senden des PROXY-Protokolls an das Backend aktiviert, wechseln HTTP-Prüfungen automatisch auf TCP.

Passive Prüfungen sind immer aktiv: Bei mehreren Backends wird ein Backend nach 3 aufeinanderfolgenden Verbindungsfehlern für 30 Sekunden pausiert und danach erneut geprüft (ein einzelnes Backend ohne aktive Prüfung wird nie entfernt). Sobald aktive Prüfungen aktiviert sind, entscheiden allein die Prüfergebnisse über die Wiederherstellung. Ändert sich der Zustand eines Backends, werden die Ereignisse „Upstream ausgefallen“ und „Upstream wiederhergestellt“ veröffentlicht, die sich über die [Benachrichtigungszentrale](/de/docs/modules/notify) versenden lassen.

### Kompression

Die Kompression lässt sich für Reverse-Proxy-, Statische-Dateien- und Fester-Text-Sites aktivieren (nicht für Weiterleitungs-Sites). Anhand von `Accept-Encoding` wird `br` bevorzugt, danach `gzip`.

- Nur textartige Inhalte werden komprimiert: `text/*`, JSON, JavaScript, XML, XHTML, RSS / Atom, WebAssembly und SVG; `text/event-stream` ist ausgenommen;
- Antworten, deren deklarierte `Content-Length` unter 256 Bytes liegt, bleiben unverändert;
- Antworten, die bereits ein `Content-Encoding` haben, bleiben unverändert; WebSocket-Upgrade-Anfragen werden direkt durchgereicht;
- Nach der Kompression werden starke ETags in schwache ETags umgewandelt.

### Ratenbegrenzung pro IP

Token-Bucket-Ratenbegrenzung pro Client-IP. Felder:

| Feld | Beschreibung |
| --- | --- |
| `rate` | Erlaubte Anfragen pro Sekunde |
| `burst` | Burst-Kapazität. Ist `rate` gesetzt und `burst` gleich 0, gilt als Standard `rate` abgerundet plus 1 |
| `maxConns` | Maximale gleichzeitige Anfragen pro IP; 0 bedeutet unbegrenzt |

Ist die Ratenbegrenzung aktiviert, muss mindestens eines von `rate` und `maxConns` gesetzt sein.

### Antwort-Cache im Arbeitsspeicher

Gilt nur für Reverse-Proxy-Sites. Der Cache liegt im Arbeitsspeicher und verdrängt nach LRU.

| Feld | Standard | Bereich |
| --- | --- | --- |
| `ttl` Standarddauer | 0 | 0 bis 30 Tage (in Sekunden). Deklariert der Upstream keine Cache-Direktive, werden 200er-Antworten so lange gecacht; 0 bedeutet, dass nur Antworten gecacht werden, die der Upstream ausdrücklich erlaubt |
| `maxMb` Cache-Kapazität | 64 | 1 bis 4096 MB |

Cache-Regeln:

- Nur GET-Anfragen werden gecacht, und die Anfrage darf weder die Header `Authorization`, `Cookie` oder `Range` noch `Cache-Control: no-cache` enthalten;
- Nur Antworten mit Status 200 werden gecacht; eine Antwort wird übersprungen, wenn sie `Set-Cookie`, `Vary: *` oder ein `Cache-Control` von `no-store` / `private` / `no-cache` enthält;
- `s-maxage` wird zuerst berücksichtigt, danach `max-age`;
- Ein einzelner Eintrag ist auf 4 MB und zugleich auf höchstens ein Viertel der Gesamtkapazität begrenzt;
- Ein Treffer liefert die Header `X-Cache: HIT` und `Age`, ein Fehlschlag liefert `X-Cache: MISS`;
- Der Cache-Schlüssel enthält Host, die vollständige Anfrage-URI und `Accept-Encoding`.

### Pfad umschreiben und weiterleiten

Regeln werden der Reihe nach abgeglichen, und beim ersten Treffer endet der Abgleich.

| Feld | Beschreibung |
| --- | --- |
| `pattern` | Ein regulärer Ausdruck, der gegen den Anfragepfad abgeglichen wird, etwa `^/old/(.*)$` |
| `to` | Das Ersetzungsergebnis; mit `$1` verweisen Sie auf Gruppen |
| `redirect` | Aus für ein internes Umschreiben (der umgeschriebene Pfad wird an das Backend gesendet, und das Ziel muss mit `/` beginnen); an für eine Weiterleitung |
| `code` | Statuscode der Weiterleitung 301 / 302 / 307 / 308, Standard 302 |

Weiterleitungen behalten die ursprünglichen Query-Parameter bei.

## Zugriffsstatistik und Monitoring

### Site-Statistik

Klicken Sie bei einem laufenden Listener in der Site-Zeile auf „Statistik“, um das Statistikfenster zu öffnen, in dem Sie die letzte 1 Stunde, 6 Stunden oder 24 Stunden wählen können:

- Zugriffsstatistik im Minutentakt: ein Punkt pro Minute, die letzten 24 Stunden aufbewahrt, mit einer Kurve der Anfragezahl sowie 4xx- / 5xx-Fehlerkurven;
- Verteilung der Statuscodes (2xx / 3xx / 4xx / 5xx) und kumulierter Antwort-Traffic;
- Die Top 10 der Quell-IPs und die Top 10 der Länder (für die Länderstatistik ist eine importierte GeoIP-Datenbank nötig);
- Reverse-Proxy-Sites zeigen Health-Status, Gewichtung, Backup-Kennzeichen und aktuelle Verbindungsanzahl jedes Backends.

Die Statistiken liegen im Arbeitsspeicher und beginnen nach einem Neustart oder nach einer Änderung des Listeners von vorn. Die Ranglisten verfolgen höchstens 5000 verschiedene Quellen.

### Prometheus-Metriken

Der globale Metriken-Endpunkt `GET /api/metrics` fasst die Metriken aller Module zusammen, siehe [Monitoring](/en/docs/reference/monitoring). Der folgende Modul-Endpunkt enthält nur die Metriken der Web-Dienste.

Der Metriken-Endpunkt ist `GET /api/webservice/metrics` im Prometheus-Textformat und erfordert Anmeldedaten oder ein [API-Token](/en/docs/reference/api-reference).

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/webservice/metrics
```

| Metrik | Typ | Beschreibung |
| --- | --- | --- |
| `gately_site_requests_total` | counter | Kumulierte Anfragen einer Site |
| `gately_site_errors_total` | counter | Backend-Fehler einer Site |
| `gately_site_active_requests` | gauge | Laufende Anfragen |
| `gately_site_response_bytes_total` | counter | Kumulierte Antwort-Bytes |
| `gately_site_responses_total` | counter | Antworten, gezählt nach Statusklasse (Label `class`: `2xx` bis `5xx`) |
| `gately_upstream_up` | gauge | Ob ein Backend gesund ist: 1 gesund, 0 nicht verfügbar (mit Label `upstream`) |

Alle Metriken tragen die Labels `listener` (Listener-Name#Listener-Schlüssel) und `site` (Site-Name oder, falls unbenannt, die erste Domain).

## Hinweis zur Bereitstellung

Web-Dienste müssen auf Ports des Hosts wie 80 / 443 lauschen. Bei Docker-Bereitstellungen empfehlen wir `--network host`.
