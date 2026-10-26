# Tailscale / frp / cloudflared

Diese drei Module binden die offiziellen Programme als verwaltete Prozesse ein: automatischer Neustart, in der Oberfläche gesammelte Logs und Konfiguration in der Oberfläche. Gately liefert diese Programme nicht mit; sie lassen sich auf der Seite online installieren, aktualisieren und zurücksetzen.

## Online installieren, aktualisieren und zurücksetzen {#install}

Die Seiten frp, Cloudflare Tunnel, Tailscale und [File Browser](/de/docs/modules/tools#filebrowser) haben unten jeweils eine Karte „Online installieren“:

- **Herunterladen und installieren / Auf neueste Version aktualisieren**: lädt die neueste stabile Version vom Upstream-Projekt, prüft sie anhand der dort erfassten SHA-256-Prüfsumme, stellt sicher, dass sie auf diesem Rechner läuft, installiert sie nach `bin/` im Konfigurationsverzeichnis und startet danach die Prozesse neu, die das Programm nutzen. Ist bereits die neueste Version installiert, wird nichts erneut heruntergeladen;
- **Version installieren**: Versionsnummer eingeben (etwa `v0.65.0` für frp, `2025.8.0` für cloudflared, `1.90.2` für Tailscale), auch für ein Downgrade;
- **Download-Proxy**: pro Programm einstellbar, unterstützt http, https und socks5, wird beim Start der Installation gespeichert; leer gilt der Update-Proxy aus den „Einstellungen“;
- **Frühere Versionen und Zurücksetzen**: Versionen, die durch ein Update oder Zurücksetzen ersetzt wurden, liegen unter `bin/.versions/`, höchstens 3. Macht eine neue Version Probleme, klicken Sie in der Liste auf „Zurücksetzen“; die aktuelle Version bleibt ebenfalls erhalten, sodass Sie später wieder zurückwechseln können;
- **Deinstallieren**: löscht das in `bin/` installierte Programm und alle früheren Versionen, die Moduleinstellungen bleiben erhalten. Prozesse, die das Programm nutzen, starten danach neu und verwenden ein gleichnamiges Programm aus `PATH`, falls vorhanden; sonst können sie nicht laufen.

| Programm | Quelle | Installierte Dateien |
| --- | --- | --- |
| frp | GitHub fatedier/frp | `frpc`, `frps` |
| cloudflared | GitHub cloudflare/cloudflared | `cloudflared` |
| Tailscale | statische Pakete von pkgs.tailscale.com, nur Linux | `tailscale`, `tailscaled` |
| FileBrowser Quantum | GitHub gtsteffaniak/filebrowser | `filebrowser` |

Für Tailscale gibt es unter macOS und Windows keine direkt lauffähigen Pakete. Verwenden Sie dort den offiziellen Installer und tragen Sie den Pfad in den Einstellungen ein. Die Karte weist darauf hin, wenn die aktuelle Plattform nicht unterstützt wird.

## Standardeinstellungen wiederherstellen {#reset}

„Standardeinstellungen wiederherstellen“ im Seitenkopf ist vom Deinstallieren getrennt: Es stoppt alle Prozesse des Moduls, setzt die Einstellungen (einschließlich Programmpfad und Download-Proxy) auf die Standardwerte zurück und löscht die Dateien, die Gately für das Modul erzeugt hat. Installierte Programme bleiben erhalten.

| Modul | Was gelöscht wird |
| --- | --- |
| frp | Client-, Server- und alle Proxy-Einstellungen, die unter `frp/` erzeugten Konfigurationsdateien |
| Cloudflare Tunnel | Alle Tunnel und Programmeinstellungen |
| Tailscale | Alle Einstellungen, der Anmeldestatus im Standard-Statusverzeichnis; läuft ein von Gately verwalteter tailscaled, meldet er sich vorher ab. Eigene Statusverzeichnisse und ein vorhandener System-tailscaled bleiben unverändert |
| File Browser | Alle Instanzen und Programmeinstellungen, Benutzer, Freigabelinks und Caches jeder Instanz; Dateien in den freigegebenen Verzeichnissen bleiben unverändert |

Die alte Konfiguration bleibt im [Konfigurationsverlauf](/de/docs/guide/audit) und lässt sich nach einem Versehen wiederherstellen. Für eine vollständige Entfernung deinstallieren Sie zuerst das Programm und stellen dann die Standardeinstellungen wieder her. Die API-Endpunkte sind `DELETE /api/{module}/install` (Deinstallieren) und `POST /api/{module}/reset` (Standardeinstellungen wiederherstellen).

## Speicherort der Programme

Die ausführbaren Dateien werden in dieser Reihenfolge gesucht:

1. dem in den Moduleinstellungen angegebenen Pfad;
2. dem Unterverzeichnis `bin/` des Konfigurationsverzeichnisses (dorthin installiert die Online-Installation);
3. `PATH`.

Ist ein Pfad eingetragen, wird die online installierte Version nicht verwendet; die Karte weist darauf hin. Sie können die Binärdatei auch selbst in das Konfigurationsverzeichnis legen, bei Docker-Bereitstellungen zum Beispiel `./gately-data/bin/frpc`. Tailscale benötigt sowohl `tailscale` als auch `tailscaled`; nutzen Sie einen vorhandenen System-tailscaled, sollte der installierte Befehl `tailscale` möglichst dieselbe Version wie der Daemon haben.

Token und Schlüssel werden mit Rechten 0600 gespeichert. Das cloudflared-Token wird über eine Umgebungsvariable übergeben und erscheint daher nicht in der Prozessliste.

## Tailscale

- Verwaltet `tailscaled` (Userspace-Networking, containerfreundlich) oder bindet sich an einen vorhandenen System-Daemon an;
- Auth-Key oder Anmeldelink, Headscale wird unterstützt;
- Subnetz-Routen und Exit-Nodes;
- Knotenliste und SOCKS5-Proxy;
- Fester WireGuard-UDP-Port, der das Zulassen direkter Verbindungen durch eine Firewall erleichtert;
- Zusätzliche tailscaled-Argumente und Umgebungsvariablen. Nutzt iptables des Hosts das nf_tables-Backend, fügen Sie `TS_DEBUG_FIREWALL_MODE=nftables` zu den Umgebungsvariablen hinzu, sonst funktioniert die Subnetz-Weiterleitung nicht.

## frp

- **frpc-Client**: Proxys vom Typ tcp / udp / http / https / stcp / xtcp; Hinzufügen oder Entfernen wirkt sofort;
- **frps-Server**: Port-Allowlist, Subdomains, Dashboard;
- **frps-Lauschadressen**: „Lauschadresse“ legt fest, wo der frpc-Verbindungsport lauscht, „Proxy-Lauschadresse“, wo die HTTP- / HTTPS-Ports und die Remote-Ports der tcp- / udp-Proxys lauschen; leer bedeutet alle Adressen. Um frps hinter einen Reverse Proxy wie Gately zu stellen, die Lauschadresse auf `127.0.0.1` setzen, im Webdienst eine Site per Reverse Proxy auf `127.0.0.1:<Bind-Port>` leiten und als frpc-Transportprotokoll websocket oder wss zu dieser Site wählen; die Proxy-Lauschadresse leer lassen, damit die Remote-Ports der tcp-Proxys weiter von außen erreichbar sind. Beim Import einer frps-Konfiguration ohne `proxyBindAddr` wird wie bei frps `bindAddr` übernommen;
- **Bestehende Konfiguration importieren**: „Konfiguration importieren“ in der Seitenkopfzeile unterstützt ältere `frpc.ini` / `frps.ini` (einschließlich des Abschnitts `[common]`) und `toml` ab frp 0.52. Sie können eine Datei wählen oder den Inhalt einfügen. Sehen Sie sich zuerst das geparste Ergebnis und die ignorierten Felder in der Vorschau an und bestätigen Sie dann den Import. Beim Client können Sie wählen, ob angehängt (Proxys mit gleichem Namen bleiben unverändert) oder alle Proxys ersetzt werden. Der Import ändert den Aktivierungsstatus nicht und lässt sich auch über die API mit `POST /api/frp/import` aufrufen.

## Cloudflare Tunnel

- Token-Tunnel und temporäre trycloudflare-Tunnel;
- Mehrere Tunnel mit automatischem Neustart;
- Zeigt die öffentliche Adresse an.

## Die richtige Wahl

| Szenario | Empfehlung |
| --- | --- |
| Ein privates Netzwerk über mehrere Geräte aufbauen | Tailscale |
| Sie besitzen einen öffentlichen Server und wollen interne Dienste darüber nach außen abbilden | frp |
| Keine öffentliche IP und kein Server, aber ein Web-Dienst soll schnell erreichbar sein | Cloudflare Tunnel |
| Ihr Provider stellt ein NAT bereit, das Hole-Punching zulässt | [STUN-Traversal](/de/docs/modules/stun) |
