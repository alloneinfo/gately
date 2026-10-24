# WAF-Firewall

Eine Web Application Firewall auf Basis von [Coraza](https://coraza.io/) und OWASP CRS v4, die sich Sites in den [Web-Diensten](/de/docs/modules/web) zuordnen lässt.

## Funktionen

- Integriertes OWASP-CRS-v4-Regelwerk, dessen Regeldateien einzeln aktiviert werden können;
- Eigene SecLang-Regeln;
- Regelgruppen lassen sich umsortieren; Treffer-Logs enthalten Client-IP, Domain, Pfad und Regel-ID und lassen sich nach Domain, IP oder Stichwörtern filtern;
- Blockiermodus und reiner Erkennungsmodus;
- Einstellbares Prüflimit für den Request-Body: Größere Anfragen werden abgelehnt oder nur im vorderen Teil geprüft; die Body-Prüfung lässt sich pro Pfad überspringen;
- WebDAV-Kompatibilitätsmodus: erlaubt die Methoden, Header und beliebigen Upload-Inhaltstypen, die Cloud-Speicher und WebDAV verwenden;
- Mehrere Instanzen, auf die Sites bei Bedarf verweisen.

## Empfehlungen

1. Starten Sie bei einer neuen Site mit „Nur erkennen“ und beobachten Sie die Logs eine Weile, um Fehlalarme auszuschließen;
2. Passen Sie bei Fehlalarmen die Regeldateien an oder fügen Sie eigene Ausnahmen hinzu;
3. Wechseln Sie danach auf „Blockieren“.

## Reihenfolge der Regelgruppen

Die Regelgruppen einer Instanz werden von oben nach unten geladen; mit den Schaltflächen nach oben und nach unten lässt sich die Reihenfolge ändern. Innerhalb derselben Phase laufen frühere Regeln zuerst:

- Sperrregeln (deny usw.) wirken an jeder Position;
- Laufzeit-Freigaben oder -Ausnahmen (`allow`, `ctl:ruleRemoveById`, `ctl:ruleRemoveTargetById` usw.) müssen über der OWASP-CRS-Gruppe stehen, sonst kann CRS die Anfrage vorher sperren;
- Konfigurationsdirektiven wie `SecRuleRemoveById` und `SecRuleUpdateTargetById` wirken nur auf bereits geladene Regeln und müssen daher unter der CRS-Gruppe stehen; darüber bleiben sie wirkungslos.

## Sites mit großen Uploads

Standardmäßig prüft die WAF Request-Bodys bis 12,5 MB; größere Anfragen werden abgelehnt (413). Cloud-Speicher, WebDAV oder Dateimanager können deshalb keine großen Dateien hochladen. In der „Prüfung des Request-Bodys“ einer Instanz lässt sich das anpassen:

- **Prüflimit**: 0 bedeutet den Standardwert 12,5 MB, höchstens 1024 MB;
- **Bei Überschreitung**: „Anfrage ablehnen“ verhält sich wie bisher; „Nur den Anfang prüfen“ prüft nur den Teil bis zum Limit und leitet den Rest unverändert an das Backend weiter – passend für Sites mit großen Uploads;
- **WebDAV- und Upload-kompatibel**: OWASP CRS erlaubt standardmäßig nur GET, HEAD, POST und OPTIONS und blockiert die Header `Lock-Token`, `If` und `Content-Range` sowie Inhaltstypen außerhalb seiner Erlaubnisliste. WebDAV-Clients können deshalb weder Verzeichnisse auflisten noch hochladen oder sperren. Eingeschaltet erlaubt die Option PUT, DELETE, PROPFIND, PROPPATCH, MKCOL, COPY, MOVE, LOCK, UNLOCK und weitere Methoden sowie diese Header und schränkt den Inhaltstyp von Uploads nicht mehr ein; Request-Bodys von PUT und `application/octet-stream` gelten als Dateiinhalt und werden nicht mehr als Formularparameter geprüft. Prüfungen auf Injection, Path Traversal und andere Angriffe bleiben aktiv;
- **Pfade ohne Body-Prüfung**: ein Pfadpräfix pro Zeile (etwa `/api/upload` oder `/remote.php/dav/`). Bei passenden Anfragen entfällt die Body-Prüfung; Header, URL und Parameter werden weiterhin geprüft.

Empfohlen ist eine eigene Instanz für Cloud-Speicher und WebDAV-Sites mit eingeschaltetem „WebDAV- und Upload-kompatibel“ und „Nur den Anfang prüfen“; bei Bedarf kommen die Upload-Endpunkte unter die ausgenommenen Pfade. Alle anderen Sites behalten die Standardeinstellungen.

curl, rclone und ähnliche Clients senden beim Hochladen größerer Inhalte den Header `Expect: 100-continue`, den CRS 4 standardmäßig blockiert. Alle Gately-Instanzen lassen diesen Header zu (Gately verarbeitet 100-continue selbst), eine zusätzliche Einstellung ist nicht nötig.

## Verifizierung

Die WAF wird mit echten Angriffs-Payloads getestet und deckt die Blockierung von Angriffen wie Path Traversal ab.
