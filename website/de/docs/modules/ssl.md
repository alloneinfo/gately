# SSL-Zertifikate

Stellen Sie Zertifikate automatisch aus und erneuern Sie sie, damit die [Web-Dienste](/de/docs/modules/web) sie nutzen oder Sie sie in andere Software exportieren können.

## Funktionen

- **ACME DNS-01** mit automatischer Ausstellung und Erneuerung, das die DNS-Anbieter aus [Dynamischem DNS](/de/docs/modules/ddns) wiederverwendet, ohne dass die Ports 80 / 443 geöffnet sein müssen;
- **ACME HTTP-01**: braucht keinen DNS-Anbieter, setzt aber voraus, dass Port 80 jeder Domain aus dem Internet erreichbar ist; Wildcards werden nicht unterstützt;
- Wildcard-Zertifikate und EAB (zum Beispiel ZeroSSL);
- Ein vorhandenes Zertifikat hochladen oder aus Dateien lesen;
- In ein Verzeichnis abbilden: Nach einer Änderung des Zertifikats werden die Dateien in das angegebene Verzeichnis geschrieben und ein Skript ausgeführt, sodass andere Dienste neu laden können.

## HTTP-01-Validierung

Wählen Sie im Zertifikat unter „Validierungsmethode“ HTTP-01. Bei der Ausstellung ruft die CA `http://<Domain>/.well-known/acme-challenge/<token>` auf:

- Lauscht ein [Web-Dienst](/de/docs/modules/web) bereits auf Port 80, beantwortet er die Anfrage direkt, ohne bestehende Sites zu beeinträchtigen;
- Ist Port 80 frei, lauscht Gately während der Ausstellung vorübergehend selbst darauf und schließt den Port danach sofort;
- Leitet der Router den öffentlichen Port 80 auf einen anderen Port dieses Hosts weiter, tragen Sie diesen Port unter „Temporärer Port“ ein.

Viele private Breitbandanschlüsse sperren eingehende Verbindungen auf Port 80; verwenden Sie in diesem Fall DNS-01.

## Zertifikatszuordnung

Ist ein Zertifikat aktiviert, schreibt Gately `.crt`, `.key`, `.pem` und `_issuerCertificate.crt` in das Zuordnungsverzeichnis und führt das Änderungsskript aus.

## Sicherheitshinweis

Private Schlüssel werden in `ssl.json` im Konfigurationsverzeichnis gespeichert. Bewahren Sie diese Datei sicher auf und stellen Sie sicher, dass das Repository privat ist, wenn Sie sie unter Versionsverwaltung stellen.

## Migration von Lucky

Bereits ausgestellte Zertifikate werden unverändert importiert und vor ihrem Ablauf nicht erneut beantragt. ZeroSSL registriert mit den EAB-Zugangsdaten aus Lucky ein neues ACME-Konto. Details finden Sie in der [Migrationsanleitung](/de/docs/guide/migration).
