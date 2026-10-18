# Gately

[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md) | [한국어](README.ko.md) | [Español](README.es.md) | [Français](README.fr.md) | **Deutsch** | [Русский](README.ru.md) | [Português (Brasil)](README.pt-BR.md)

Ein quelloffenes, prüfbares Toolkit für Netzwerk-Gateways – für Heimnetz, NAS, Cloud-Hosts und beliebige Server. DDNS, STUN-Traversal, Portweiterleitung, Reverse-Proxy, Zertifikate, WAF und weitere gängige Funktionen sind in einer einzigen Datei von etwa 22 MB vereint; bestehende Lucky-Konfigurationen lassen sich importieren.

![Gately-Übersicht](docs/images/overview.webp)

![Dynamisches DNS](docs/images/ddns.webp)

![Docker](docs/images/docker.webp)

## Warum Gately

Gately ist vollständig quelloffen, sodass Nutzer und Community Quellcode und Konfiguration direkt prüfen können. Bei Software, die an der Netzwerkgrenze läuft und DNS-Schlüssel sowie private Zertifikatsschlüssel verwaltet, zeigt Open Source, was sie tatsächlich tut. Die Zusagen von Gately:

- Der Quellcode steht unter Apache-2.0 und kann von der Community geprüft werden
- Standardmäßig keine Verbindung zu Servern des Projekts und keine Übermittlung von Geräteinformationen; die Anmeldung über Drittanbieter nutzt Ihre eigenen OAuth-Anwendungen ohne Zwischenstation
- Alle externen Adressen (Abfrage der öffentlichen IP, STUN-Server usw.) sind sichtbar, änderbar und abschaltbar
- Die Konfiguration ist reines JSON und lässt sich zur Prüfung und für Rollbacks direkt in git verwalten

## Funktionen

| Modul | Funktionen |
| --- | --- |
| Dynamisches DNS | Alibaba Cloud, Cloudflare, Tencent Cloud, DNSPod, Huawei Cloud, GoDaddy, Porkbun, dynv6, DuckDNS, benutzerdefinierter Callback; IP per URL/Netzwerkschnittstelle/Befehl ermitteln; IPv4/IPv6; Webhook |
| STUN-Traversal | TCP/UDP, Keepalive, UPnP/NAT-PMP, integrierte Weiterleitung, Webhook und Skript bei Adressänderung |
| Portweiterleitung | TCP/UDP, IPv4/IPv6, Portbereiche, Failover mit mehreren Zielen, Verbindungslimit, Traffic-Statistik |
| SSL-Zertifikate | Automatische Ausstellung und Erneuerung per ACME DNS-01 (Wildcard, EAB), Upload, Dateien einlesen, in ein Verzeichnis abbilden und Skript ausführen |
| Tailscale | Verwaltet tailscaled (Userspace-Netzwerk, container-freundlich) oder nutzt einen vorhandenen System-Daemon; Auth-Key/Login-Link, Headscale, Subnetz-Routen, Exit-Node, Knotenliste und SOCKS5-Proxy, fester Port, zusätzliche tailscaled-Argumente und Umgebungsvariablen |
| frp | frpc-Client (tcp/udp/http/https/stcp/xtcp-Proxys, Änderungen wirken sofort) und frps-Server (Port-Whitelist, Subdomains, Dashboard); Konfigurationen im alten ini- und im neuen toml-Format lassen sich mit einem Klick importieren |
| Cloudflare Tunnel | cloudflared-Token-Tunnel und temporäre trycloudflare-Tunnel, mehrere Tunnel, automatischer Neustart, Anzeige der öffentlichen Adresse |
| Web-Dienste | Reverse-Proxy/Weiterleitung/statische Dateien/Festtext je Domain; HTTPS; WebSocket; PROXY-Protokoll senden und empfangen; Basic-Authentifizierung, Web-Authentifizierung, Anmeldung über GitHub/Google/QQ/OIDC; IP-Black-/Whitelist, Pfadregeln, temporäre Freigabelinks, automatische Sperre nach fehlgeschlagenen Anmeldungen, GeoIP-Länderfilter; Backend-Gewichtung/Backup/Lastverteilungsstrategien, aktive und passive Health-Checks, Kompression, Ratenbegrenzung, Antwort-Cache, Pfad-Rewrite; minutengenaue Zugriffsstatistik und Prometheus-Metriken |
| WAF-Firewall | Coraza + OWASP CRS v4, Regeln dateiweise aktivierbar, eigenes SecLang, Blockieren/nur Erkennen |
| Docker-Verwaltung | Container erstellen und bearbeiten, Logs und Terminal, Compose, Prüfung auf Image-Updates und Upgrade per Klick (bei Fehlern automatischer Rollback), Netzwerke und Volumes |
| Web-Terminal | SSH (Hostschlüssel beim ersten Mal bestätigen), Telnet, lokale Shell (standardmäßig deaktiviert); Proxy, persistente Sitzungen, SFTP-Dateiverwaltung, 2FA und Protokollbestätigung, Schnellbefehle, geteilte Ansicht |
| Netzwerkdiagnose | Ping, Traceroute, Portprüfung, DNS-Abfrage, Geschwindigkeitstest, LAN-Scan |
| Benachrichtigungszentrale | Webhook, Telegram, Bark, WeCom, DingTalk, Feishu, E-Mail; Stufen sowie Regeln für Wildcards und Stummschaltung/Deduplizierung |
| Konfigurationsbackup | Verschlüsselte Backups (scrypt + AES-GCM), Aufbewahrungsrichtlinie, zeitgesteuerte Backups in ein lokales Verzeichnis oder einen Speicher-Mountpunkt |
| Kommandozeile und MCP | `gately ctl` steuert die laufende Instanz, `gately mcp` stellt einen MCP-Dienst über stdio bereit (standardmäßig schreibgeschützt) |
| Wake-on-LAN | Magic-Packet-Broadcast, Online-Erkennung |
| WebDAV | Mehrere Benutzer, eigene Verzeichnisse, schreibgeschützt |
| FTP | Mehrere Benutzer, eigene Verzeichnisse, schreibgeschützt, mit FTPS und passivem Portbereich |
| Speicherverwaltung | Lokale Verzeichnisse und entfernte WebDAV-Mounts, Dateiverwaltung online |
| Geplante Aufgaben | cron-Ausdrücke, Befehle oder HTTP-Anfragen, Ausführungsprotokoll |
| Sicherheit und Einstellungen | Zwei-Faktor-Authentifizierung (TOTP), Ratenbegrenzung bei fehlgeschlagenen Anmeldungen, Sicherheitseinstieg, Zugriff nur aus dem internen Netz, HTTPS für die Verwaltungsoberfläche, mehrere API-Tokens (Lese-/Schreibrechte je Modul), benutzerdefiniertes DNS (mit DoH-Unterstützung), Zeitzone, Konfigurationsbackup und -wiederherstellung |
| Migration | Import von Lucky-`.lkcf`-Dateien oder Konfigurations-ZIPs der gesamten Instanz: Portweiterleitung, DDNS, STUN, Zertifikate, Reverse-Proxy, WAF, Wake-on-LAN, Web-Terminal, WebDAV, Grundeinstellungen |

## Schnellstart

### Docker

Das Image unterstützt amd64, arm64 und arm/v7:

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep -i "initial password"
```

Öffnen Sie `http://<Geräte-IP>:16680`, melden Sie sich mit `admin` und dem Anfangspasswort aus dem Log an und ändern Sie anschließend unter „Einstellungen“ das Passwort und aktivieren Sie die Zwei-Faktor-Authentifizierung. Ein vollständiges Compose-Beispiel finden Sie in [deploy/docker-compose.yml](deploy/docker-compose.yml).

Portweiterleitung, STUN und Web-Dienste müssen Ports des Hosts belegen; empfohlen wird `--network host`. Wird die Docker-Verwaltung nicht benötigt, kann der `docker.sock`-Mount entfallen.

### Binärdatei

Laden Sie von [Releases](https://github.com/wangwanjie/gately/releases) das Archiv für Ihre Plattform herunter (Linux amd64/arm64/arm/mips/mipsle/riscv64, macOS, Windows, FreeBSD), entpacken Sie es und starten Sie:

```bash
./gately -c /etc/gately
```

Parameter: `-c` Konfigurationsverzeichnis (alternativ die Umgebungsvariable `GATELY_CONFIG_DIR`), `-listen` überschreibt die Listen-Adresse der Verwaltungsoberfläche, `-version` gibt die Version aus.

### API-Aufrufe

Erstellen Sie auf der Seite „API-Tokens“ der Verwaltungsoberfläche ein Token (mehrere sind möglich, je Modul mit Lese- oder Lese-/Schreibrechten, mit optionalem Ablaufdatum; der Klartext wird nur einmal nach der Erstellung angezeigt). Damit können Skripte die Verwaltungs-API in großem Umfang aufrufen:

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks
```

Konto, Zwei-Faktor-Authentifizierung, Token-Verwaltung sowie die Schnittstellen für Backup, Wiederherstellung, Neustart und Systemeinstellungen sind nur mit einer Anmeldesitzung zugänglich. Alle Schnittstellen, erforderlichen Berechtigungen und Datenstrukturen finden Sie auf der Seite „API-Dokumentation“ der Verwaltungsoberfläche; mit `make api-docs` lassen sie sich nach [docs/api](docs/api/de) exportieren (`openapi.json` und `API.md`), auf die die Website https://gately.cn bei der Veröffentlichung direkt verweist.

Beim Upgrade von einer älteren Version wird das bisherige globale API-Token automatisch zu einem „Altbestand“-Token (der Wert bleibt gleich, mit Rechten für alle Module). Es empfiehlt sich, auf der Seite „API-Tokens“ die Rechte einzuschränken oder es durch ein neues Token zu ersetzen.

### Tailscale, frp, cloudflared

Diese drei Module verwalten offizielle externe Programme (automatischer Neustart, Logs werden in der Oberfläche gesammelt). Gately selbst enthält sie nicht, sodass stets die neueste Upstream-Version genutzt werden kann. Die Programme werden in dieser Reihenfolge gesucht: der in den Moduleinstellungen angegebene Pfad, `PATH`, das Unterverzeichnis `bin/` im Konfigurationsverzeichnis. Bei Docker-Deployments legen Sie die Binärdatei einfach in das eingebundene Konfigurationsverzeichnis, z. B. `./gately-data/bin/frpc`; für Tailscale müssen `tailscale` und `tailscaled` bereitgestellt werden. Token und Schlüssel in den Konfigurationsdateien werden mit Berechtigung 0600 gespeichert; das cloudflared-Token wird über eine Umgebungsvariable übergeben und taucht nicht in der Prozessliste auf.

### Migration von Lucky

Siehe die [Migrationsanleitung](https://gately.cn/de/docs/guide/migration). Kurzfassung: Wählen Sie unter „Einstellungen → Aus Lucky importieren“ die Datei `lucky_*.lkcf` aus dem Lucky-Konfigurationsverzeichnis, prüfen Sie die Vorschau und importieren Sie. Importierte Regeln sind standardmäßig deaktiviert; deaktivieren Sie Modul für Modul in Lucky und aktivieren Sie es in Gately.

## Entwicklung

Voraussetzungen: Go 1.26+, Node.js 24+, pnpm.

```bash
make web      # Frontend bauen (Ausgabe in web/dist, per go:embed eingebettet)
make build    # bin/gately kompilieren
make test     # go test -race ./...
make lint     # gofmt, go vet, Typprüfung des Frontends
```

Für die Frontend-Entwicklung führen Sie `cd web && pnpm dev` aus; `/api` wird an das lokale `127.0.0.1:16680` weitergeleitet.

Einige Tests werden standardmäßig übersprungen und brauchen zusätzliche Voraussetzungen:

| Test | Aktivierung |
| --- | --- |
| ACME-End-to-End (Pebble) | pebble und pebble-challtestsrv installieren, `GATELY_PEBBLE_BIN` und `GATELY_PEBBLE_SRC` setzen (läuft in der CI automatisch) |
| Echte öffentliche STUN-Server | `GATELY_NET_TEST=1` |
| Import-Prüfung mit echter Lucky-Konfiguration | `GATELY_LUCKY_SAMPLE` usw., siehe `TestImportRealSample` im jeweiligen Modul |

Verzeichnisstruktur und Technologieauswahl siehe [docs/ROADMAP.md](docs/ROADMAP.md).

Der Quellcode der Website ([gately.cn](https://gately.cn)) liegt in `website/` und basiert auf VitePress: `cd website && pnpm install && pnpm dev`; `pnpm deploy:site` baut und synchronisiert auf den Server.

## Beitragsrichtlinien

- Die Kompatibilität mit Lucky beruht auf öffentlich beobachtbarem Verhalten, Schnittstellen und Konfigurationsformaten. Reichen Sie keinen Code ein, der aus Lucky-Binärdateien oder Frontend-Artefakten kopiert wurde.
- Neue Module benötigen: Konfigurationsstruktur und Validierung, Start/Stopp zur Laufzeit, REST-Schnittstelle, Unit-Tests; Module, die sich von Lucky migrieren lassen, implementieren `luckyimport.Importer`, Module, die ihren Status auf der Übersichtsseite anzeigen sollen, implementieren `module.Summarizer`.
- Code-Kommentare werden auf Chinesisch verfasst; Kontrollanweisungen müssen geschweifte Klammern verwenden.

## Community

Fragen, Feedback und Nutzungstipps sind in der Telegram-Gruppe willkommen: [t.me/gately_chat](https://t.me/gately_chat). Fehler bitte in den [GitHub Issues](https://github.com/wangwanjie/gately/issues) melden.

## Star-Verlauf

[![Star History Chart](https://api.star-history.com/svg?repos=wangwanjie/gately&type=Date)](https://star-history.com/#wangwanjie/gately&Date)

## Lizenz

[Apache License 2.0](LICENSE)
