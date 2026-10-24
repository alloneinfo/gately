# SSL-Zertifikate

Stellen Sie Zertifikate automatisch aus und erneuern Sie sie, damit die [Web-Dienste](/de/docs/modules/web) sie nutzen oder Sie sie in andere Software exportieren können.

## Funktionen

- **ACME DNS-01** mit automatischer Ausstellung und Erneuerung, das die DNS-Anbieter aus [Dynamischem DNS](/de/docs/modules/ddns) wiederverwendet, ohne dass die Ports 80 / 443 geöffnet sein müssen;
- **ACME HTTP-01**: braucht keinen DNS-Anbieter, setzt aber voraus, dass Port 80 jeder Domain aus dem Internet erreichbar ist; Wildcards werden nicht unterstützt;
- Wildcard-Zertifikate und EAB (zum Beispiel ZeroSSL);
- Ein vorhandenes Zertifikat hochladen oder aus Dateien lesen;
- In ein Verzeichnis abbilden: Nach einer Änderung des Zertifikats werden die Dateien in das angegebene Verzeichnis geschrieben und ein Skript ausgeführt, sodass andere Dienste neu laden können;
- Automatische Bereitstellung: Nach einer Änderung wird das Zertifikat in mehrere lokale Verzeichnisse kopiert, per SSH auf andere Hosts übertragen und dort ein Befehl ausgeführt, oder die Docker-Container, die es nutzen, werden neu gestartet.

## HTTP-01-Validierung

Wählen Sie im Zertifikat unter „Validierungsmethode“ HTTP-01. Bei der Ausstellung ruft die CA `http://<Domain>/.well-known/acme-challenge/<token>` auf:

- Lauscht ein [Web-Dienst](/de/docs/modules/web) bereits auf Port 80, beantwortet er die Anfrage direkt, ohne bestehende Sites zu beeinträchtigen;
- Ist Port 80 frei, lauscht Gately während der Ausstellung vorübergehend selbst darauf und schließt den Port danach sofort;
- Leitet der Router den öffentlichen Port 80 auf einen anderen Port dieses Hosts weiter, tragen Sie diesen Port unter „Temporärer Port“ ein.

Viele private Breitbandanschlüsse sperren eingehende Verbindungen auf Port 80; verwenden Sie in diesem Fall DNS-01.

## Zertifikatszuordnung

Ist ein Zertifikat aktiviert, schreibt Gately `.crt`, `.key`, `.pem` und `_issuerCertificate.crt` in das Zuordnungsverzeichnis und führt das Änderungsskript aus.

## Automatische Bereitstellung

Unter „Automatische Bereitstellung“ eines Zertifikats lassen sich mehrere Bereitstellungsziele hinzufügen, die nach Ausstellung, Verlängerung oder Aktualisierung des Zertifikats automatisch ausgeführt werden:

- **In lokales Verzeichnis kopieren**: Schreibt das Zertifikat in ein Verzeichnis auf diesem Rechner und führt danach lokal den Befehl unter „Nach dem Kopieren ausführen“ aus. Mehrere Verzeichnisse mit jeweils eigenem Befehl sind möglich. Werden die Zertifikatsdateien in einem Verzeichnis gelöscht oder geändert, kopiert die nächste Prüfung sie erneut. Läuft Gately in Docker, liegen Verzeichnis und Befehl im Container;
- **Auf SSH-Host hochladen**: Wählen Sie eine vorhandene SSH-Verbindung aus dem [Web-Terminal](/de/docs/modules/terminal); deren Adresse, Authentifizierung und Proxy werden übernommen. Die Zertifikatsdateien (`<Notiz>.crt`, `.key`, `.pem`, `_issuerCertificate.crt`, wie bei der Zertifikatszuordnung, daher darf die Notiz nicht leer sein) werden in das entfernte Verzeichnis geschrieben, danach führt der entfernte Host den Befehl aus „Nach dem Hochladen ausführen“ aus, etwa `systemctl reload nginx`. Im Befehl stehen `$certName`, `$certDir`, `$certFile` und `$keyFile` zur Verfügung. Jede Datei wird zuerst als temporäre Datei mit den passenden Rechten geschrieben und dann per Umbenennen ersetzt; der private Schlüssel erhält die Rechte 0600;
- **Lokale Container neu starten**: Nach einer Zertifikatsänderung werden die ausgewählten Docker-Container nacheinander neu gestartet. Das eignet sich für Dienste, deren Container das Zuordnungsverzeichnis einbinden und für ein neues Zertifikat neu starten müssen.

„In lokales Verzeichnis kopieren“ und „Auf SSH-Host hochladen“ akzeptieren Dateinamen für Zertifikat (vollständige Kette) und privaten Schlüssel, etwa `fullchain.pem` und `privkey.pem`. Sind beide angegeben, werden nur diese zwei Dateien geschrieben und die Notiz darf leer sein; leer bleiben die Dateinamen der Zertifikatszuordnung. `$certFile` und `$keyFile` im Befehl zeigen auf die tatsächlich geschriebenen Dateien.

Die automatische Bereitstellung vertraut keinem unbekannten Host: Mit der gewählten SSH-Verbindung muss zuerst einmal im Web-Terminal verbunden werden, damit der Host-Schlüssel gespeichert ist; erst dann wird übertragen. Fehlgeschlagene Bereitstellungen werden stündlich wiederholt und lösen das Ereignis „Zertifikat: Bereitstellung fehlgeschlagen“ aus, das Sie in der [Benachrichtigungszentrale](/de/docs/modules/notify) abonnieren können. „Jetzt bereitstellen“ auf der Zertifikatskarte ignoriert das Wiederholungsintervall, stellt sofort auf alle aktivierten Ziele bereit und zeigt das Ergebnis je Ziel an.

Beim Bearbeiten eines Zertifikats werden Bereitstellungen, deren Zieleinstellungen unverändert sind, nicht erneut ausgeführt; nach einer Änderung von entferntem Verzeichnis, Befehl oder Containern wird einmal neu bereitgestellt.

## Sicherheitshinweis

Private Schlüssel werden in `ssl.json` im Konfigurationsverzeichnis gespeichert. Bewahren Sie diese Datei sicher auf und stellen Sie sicher, dass das Repository privat ist, wenn Sie sie unter Versionsverwaltung stellen.
