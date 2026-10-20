# FAQ

## Muss Gately sich mit den Servern des Projekts verbinden?

Nein. Gately verbindet sich standardmäßig mit keinem Projektserver und übermittelt keine Geräteinformationen. Ausgehende Adressen wie die Abfrage der öffentlichen IP nutzen öffentliche Drittanbieter-Dienste und lassen sich in den jeweiligen Aufgaben ändern.

## Wo finde ich das Initialpasswort?

Es wird beim ersten Start einmalig ins Log geschrieben. Führen Sie bei Docker-Bereitstellungen `docker logs gately 2>&1 | grep -i "initial password"` aus.

## Was tun, wenn ich das Administratorpasswort vergessen habe?

Beenden Sie Gately, löschen Sie `account.json` im Konfigurationsverzeichnis und starten Sie es erneut. Das Programm legt das Konto `admin` neu an und schreibt ein neues Initialpasswort ins Log. Prüfen Sie nach der Anmeldung die Sicherheitseinstellungen, etwa die Zwei-Faktor-Authentifizierung.

## Unterstützt Gately fnOS?

Ja. Jedes Release enthält Offline-Pakete für fnOS (`.fpk`), die Sie manuell über das App Center installieren. Gately ist in das einheitliche fnOS-Gateway eingebunden, öffnet sich über das Desktop-Symbol und funktioniert auch über Ihre Remote-Domain von fnOS. Aktivieren Sie „Mit fnOS anmelden“ unter „Einstellungen → Zugriffskontrolle“, dann gelangen bei fnOS angemeldete Administratoren direkt hinein, ohne Passwort und Bestätigungscode. Siehe [Native fnOS-App](/de/docs/guide/install#fnos).

## Warum wird `--network host` empfohlen?

Portweiterleitung, STUN und Web-Dienste lauschen direkt auf Ports des Hosts. Beim Bridge-Netzwerk müssen Sie jeden verwendeten Port selbst veröffentlichen.

## Kann es parallel zu Lucky laufen?

Ja. Der Verwaltungsport ist standardmäßig 16680 und kollidiert nicht mit dem Port 16601 von Lucky. Importierte Regeln sind standardmäßig deaktiviert, damit sie nicht mit einem noch laufenden Lucky um Ports und UPnP-Zuordnungen des Routers konkurrieren.

## Warum muss ich die Binärdateien von Tailscale, frp und cloudflared selbst bereitstellen?

Diese drei Module betreiben die offiziellen externen Programme. Gately liefert sie nicht mit, sodass Sie stets die neueste Upstream-Version nutzen können. Wohin Sie sie legen, steht unter [Konfiguration & Laufzeitoptionen](/de/docs/guide/configuration).

## Meine STUN-Regel ist aktiviert, aber ich komme von außen nicht durch

Liegt der WAN-Anschluss Ihres Routers hinter Carrier-Grade-NAT (100.64.0.0/10), können externe Hosts den durchgereichten Port nicht erreichen. Das ist eine Grenze des Netzwerks selbst. Nutzen Sie stattdessen [Tailscale / frp / Cloudflare Tunnel](/de/docs/modules/tunnels).

## Wie melde ich Probleme oder wirke mit?

Eröffnen Sie ein Issue unter [GitHub Issues](https://github.com/wangwanjie/gately/issues). Die Entwicklungskonventionen stehen unter [Mitwirken](/de/docs/guide/contributing).
