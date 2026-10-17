# SSL-Zertifikate

Stellen Sie Zertifikate automatisch aus und erneuern Sie sie, damit die [Web-Dienste](/de/docs/modules/web) sie nutzen oder Sie sie in andere Software exportieren können.

## Funktionen

- **ACME DNS-01** mit automatischer Ausstellung und Erneuerung, das die DNS-Anbieter aus [Dynamischem DNS](/de/docs/modules/ddns) wiederverwendet, ohne dass die Ports 80 / 443 geöffnet sein müssen;
- Wildcard-Zertifikate und EAB (zum Beispiel ZeroSSL);
- Ein vorhandenes Zertifikat hochladen oder aus Dateien lesen;
- In ein Verzeichnis abbilden: Nach einer Änderung des Zertifikats werden die Dateien in das angegebene Verzeichnis geschrieben und ein Skript ausgeführt, sodass andere Dienste neu laden können.

## Zertifikatszuordnung

Ist ein Zertifikat aktiviert, schreibt Gately `.crt`, `.key`, `.pem` und `_issuerCertificate.crt` in das Zuordnungsverzeichnis und führt das Änderungsskript aus. Die Dateinamen entsprechen denen von Lucky, und der geschriebene Inhalt ist Byte für Byte identisch mit dem von Lucky.

## Sicherheitshinweis

Private Schlüssel werden in `ssl.json` im Konfigurationsverzeichnis gespeichert. Bewahren Sie diese Datei sicher auf und stellen Sie sicher, dass das Repository privat ist, wenn Sie sie unter Versionsverwaltung stellen.

## Migration von Lucky

Bereits ausgestellte Zertifikate werden unverändert importiert und vor ihrem Ablauf nicht erneut beantragt. ZeroSSL registriert mit den EAB-Zugangsdaten aus Lucky ein neues ACME-Konto. Details finden Sie in der [Migrationsanleitung](/de/docs/guide/migration).
