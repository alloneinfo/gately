# Gately Verwaltungs-API

Die Gately Verwaltungs-API dient der Stapelverwaltung per Skript und Automatisierung.

Authentifizierung: Erstellen Sie auf der Seite „API-Tokens“ der Verwaltungsoberfläche ein Token und senden Sie im Request-Header `Authorization: Bearer gtl_xxxx`.

Berechtigungen: Jedes Token besitzt nur die bei der Erstellung ausgewählten Berechtigungen im Format `<Modul>:read` oder `<Modul>:write`, wobei Schreibrechte Leserechte einschließen. Schnittstellen für Konto, Zwei-Faktor-Authentifizierung, Token-Verwaltung sowie Backup, Wiederherstellung, Neustart und Systemeinstellungen sind nur mit einer Anmeldesitzung nutzbar.

Der Anfragekörper verwendet `Content-Type: application/json`, Fehler werden einheitlich als `{"error": "Beschreibung"}` zurückgegeben.

Website: https://gately.cn

Die maschinenlesbare Beschreibung befindet sich im selben Verzeichnis in `openapi.json` (OpenAPI 3.0).

## Schnellstart

```bash
# Systemstatus anzeigen (erfordert status:read)
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/status

# Alle DDNS-Aufgaben auflisten (erfordert ddns:read)
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks

# Eine DDNS-Aufgabe sofort synchronisieren (erfordert ddns:write, key stammt aus der Liste des vorherigen Schritts)
curl -X POST -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks/<key>/sync
```

## Berechtigungen

| Berechtigungsbereich | Name | Verfügbare Berechtigungen |
| --- | --- | --- |
| `portforward` | Portweiterleitung | `portforward:read`、`portforward:write` |
| `ddns` | Dynamisches DNS | `ddns:read`、`ddns:write` |
| `stun` | STUN-Durchleitung | `stun:read`、`stun:write` |
| `tailscale` | Tailscale-Netzwerk | `tailscale:read`、`tailscale:write` |
| `frp` | frp-Intranet-Durchleitung | `frp:read`、`frp:write` |
| `cloudflared` | Cloudflare-Tunnel | `cloudflared:read`、`cloudflared:write` |
| `ssl` | SSL-Zertifikate | `ssl:read`、`ssl:write` |
| `waf` | WAF-Firewall | `waf:read`、`waf:write` |
| `webservice` | Web-Dienst | `webservice:read`、`webservice:write` |
| `wol` | Wake-on-LAN | `wol:read`、`wol:write` |
| `webterminal` | Web-Terminal | `webterminal:read`、`webterminal:write` |
| `cron` | Geplante Aufgaben | `cron:read`、`cron:write` |
| `docker` | Docker-Verwaltung | `docker:read`、`docker:write` |
| `webdav` | WebDAV | `webdav:read`、`webdav:write` |
| `ftp` | FTP-Dienst | `ftp:read`、`ftp:write` |
| `storage` | Speicherverwaltung | `storage:read`、`storage:write` |
| `notify` | Benachrichtigungszentrale | `notify:read`、`notify:write` |
| `nettools` | Netzwerkdiagnose | `nettools:read`、`nettools:write` |
| `backup` | Konfigurations-Backup | `backup:read`、`backup:write` |
| `status` | Systemstatus | `status:read` |
| `logs` | Systemprotokolle | `logs:read` |
| `metrics` | Metriken (Prometheus) | `metrics:read` |
| `import` | Lucky-Konfigurationsimport | `import:write` |

Wildcard-Schreibweise: `*:read` bedeutet Lesezugriff auf alle Module, `*:write` oder `*` bedeutet Lese- und Schreibzugriff auf alle Module.

## Portweiterleitung

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/portforward/logs` | `portforward:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/portforward/logs` | `portforward:write` | Modulprotokolle leeren |
| GET | `/api/portforward/rules` | `portforward:read` | Portweiterleitungsregeln auflisten |
| POST | `/api/portforward/rules` | `portforward:write` | Portweiterleitungsregeln hinzufügen |
| PUT | `/api/portforward/rules/{key}` | `portforward:write` | Portweiterleitungsregeln ändern |
| DELETE | `/api/portforward/rules/{key}` | `portforward:write` | Portweiterleitungsregeln löschen |
| PUT | `/api/portforward/rules/{key}/enable` | `portforward:write` | Portweiterleitungsregeln aktivieren oder deaktivieren |
| PUT | `/api/portforward/rules-order` | `portforward:write` | Portweiterleitungsregeln neu anordnen |
| GET | `/api/portforward/settings` | `portforward:read` | Globale Portweiterleitungs-Einstellungen lesen |
| PUT | `/api/portforward/settings` | `portforward:write` | Globale Portweiterleitungs-Einstellungen speichern |

## Dynamisches DNS

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/ddns/logs` | `ddns:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/ddns/logs` | `ddns:write` | Modulprotokolle leeren |
| GET | `/api/ddns/tasks` | `ddns:read` | DDNS-Aufgaben auflisten |
| POST | `/api/ddns/tasks` | `ddns:write` | DDNS-Aufgaben hinzufügen |
| PUT | `/api/ddns/tasks/{key}` | `ddns:write` | DDNS-Aufgaben ändern |
| DELETE | `/api/ddns/tasks/{key}` | `ddns:write` | DDNS-Aufgaben löschen |
| PUT | `/api/ddns/tasks/{key}/enable` | `ddns:write` | DDNS-Aufgaben aktivieren oder deaktivieren |
| PUT | `/api/ddns/tasks-order` | `ddns:write` | DDNS-Aufgaben neu anordnen |
| POST | `/api/ddns/tasks/{key}/sync` | `ddns:write` | Eine DDNS-Aufgabe sofort synchronisieren |
| GET | `/api/ddns/providers` | `ddns:read` | Unterstützte DNS-Anbieter und ihre erforderlichen Parameter auflisten |
| GET | `/api/ddns/netinterfaces` | `ddns:read` | Netzwerkschnittstellen und Adressen dieses Rechners auflisten |
| POST | `/api/ddns/iptest` | `ddns:write` | Ermittlung der öffentlichen IP testen |
| GET | `/api/ddns/defaults` | `ddns:read` | DDNS-Standardwerte lesen |
| GET | `/api/ddns/settings` | `ddns:read` | Globale DDNS-Einstellungen lesen |
| PUT | `/api/ddns/settings` | `ddns:write` | Globale DDNS-Einstellungen speichern |

## STUN-Durchleitung

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/stun/logs` | `stun:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/stun/logs` | `stun:write` | Modulprotokolle leeren |
| GET | `/api/stun/rules` | `stun:read` | STUN-Regeln auflisten |
| POST | `/api/stun/rules` | `stun:write` | STUN-Regeln hinzufügen |
| PUT | `/api/stun/rules/{key}` | `stun:write` | STUN-Regeln ändern |
| DELETE | `/api/stun/rules/{key}` | `stun:write` | STUN-Regeln löschen |
| PUT | `/api/stun/rules/{key}/enable` | `stun:write` | STUN-Regeln aktivieren oder deaktivieren |
| PUT | `/api/stun/rules-order` | `stun:write` | STUN-Regeln neu anordnen |
| POST | `/api/stun/rules/{key}/restart` | `stun:write` | Eine STUN-Regel neu starten |
| GET | `/api/stun/netinterfaces` | `stun:read` | Netzwerkschnittstellen und Adressen dieses Rechners auflisten |
| GET | `/api/stun/settings` | `stun:read` | Globale STUN-Einstellungen lesen |
| PUT | `/api/stun/settings` | `stun:write` | Globale STUN-Einstellungen speichern |

## Tailscale-Netzwerk

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/tailscale/logs` | `tailscale:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/tailscale/logs` | `tailscale:write` | Modulprotokolle leeren |
| GET | `/api/tailscale/settings` | `tailscale:read` | Tailscale-Einstellungen lesen |
| PUT | `/api/tailscale/settings` | `tailscale:write` | Tailscale-Einstellungen speichern |
| GET | `/api/tailscale/status` | `tailscale:read` | Tailscale-Verbindungsstatus anzeigen |
| POST | `/api/tailscale/restart` | `tailscale:write` | Tailscale neu starten |
| POST | `/api/tailscale/logout` | `tailscale:write` | Von Tailscale abmelden |

## frp-Intranet-Durchleitung

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/frp/logs` | `frp:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/frp/logs` | `frp:write` | Modulprotokolle leeren |
| GET | `/api/frp/proxies` | `frp:read` | Einträge auflisten |
| POST | `/api/frp/proxies` | `frp:write` | Einträge hinzufügen |
| PUT | `/api/frp/proxies/{key}` | `frp:write` | Einträge ändern |
| DELETE | `/api/frp/proxies/{key}` | `frp:write` | Einträge löschen |
| PUT | `/api/frp/proxies/{key}/enable` | `frp:write` | Einträge aktivieren oder deaktivieren |
| PUT | `/api/frp/proxies-order` | `frp:write` | Einträge neu anordnen |
| POST | `/api/frp/import` | `frp:write` | Vorhandene frpc-/frps-Konfiguration importieren (unterstützt altes ini und neues toml); bei apply false nur Vorschau |
| GET | `/api/frp/client` | `frp:read` | frp-Client-Konfiguration lesen |
| PUT | `/api/frp/client` | `frp:write` | frp-Client-Konfiguration speichern |
| GET | `/api/frp/server` | `frp:read` | frp-Server-Konfiguration lesen |
| PUT | `/api/frp/server` | `frp:write` | frp-Server-Konfiguration speichern |

## Cloudflare-Tunnel

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/cloudflared/logs` | `cloudflared:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/cloudflared/logs` | `cloudflared:write` | Modulprotokolle leeren |
| GET | `/api/cloudflared/tunnels` | `cloudflared:read` | Cloudflare-Tunnel auflisten |
| POST | `/api/cloudflared/tunnels` | `cloudflared:write` | Cloudflare-Tunnel hinzufügen |
| PUT | `/api/cloudflared/tunnels/{key}` | `cloudflared:write` | Cloudflare-Tunnel ändern |
| DELETE | `/api/cloudflared/tunnels/{key}` | `cloudflared:write` | Cloudflare-Tunnel löschen |
| PUT | `/api/cloudflared/tunnels/{key}/enable` | `cloudflared:write` | Cloudflare-Tunnel aktivieren oder deaktivieren |
| PUT | `/api/cloudflared/tunnels-order` | `cloudflared:write` | Cloudflare-Tunnel neu anordnen |
| POST | `/api/cloudflared/tunnels/{key}/restart` | `cloudflared:write` | Einen Cloudflare-Tunnel neu starten |
| GET | `/api/cloudflared/settings` | `cloudflared:read` | Cloudflare-Tunnel-Einstellungen lesen |
| PUT | `/api/cloudflared/settings` | `cloudflared:write` | Cloudflare-Tunnel-Einstellungen speichern |

## SSL-Zertifikate

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/ssl/logs` | `ssl:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/ssl/logs` | `ssl:write` | Modulprotokolle leeren |
| GET | `/api/ssl/certs` | `ssl:read` | Zertifikate auflisten |
| POST | `/api/ssl/certs` | `ssl:write` | Zertifikate hinzufügen |
| PUT | `/api/ssl/certs/{key}` | `ssl:write` | Zertifikate ändern |
| DELETE | `/api/ssl/certs/{key}` | `ssl:write` | Zertifikate löschen |
| PUT | `/api/ssl/certs/{key}/enable` | `ssl:write` | Zertifikate aktivieren oder deaktivieren |
| PUT | `/api/ssl/certs-order` | `ssl:write` | Zertifikate neu anordnen |
| POST | `/api/ssl/certs/{key}/renew` | `ssl:write` | Zertifikat sofort ausstellen oder erneuern |
| GET | `/api/ssl/certs/{key}/download` | `ssl:write` | Zertifikat und privaten Schlüssel herunterladen |
| GET | `/api/ssl/providers` | `ssl:read` | Unterstützte DNS-Validierungsanbieter auflisten |
| GET | `/api/ssl/settings` | `ssl:read` | Globale Zertifikatseinstellungen lesen |
| PUT | `/api/ssl/settings` | `ssl:write` | Globale Zertifikatseinstellungen speichern |

## WAF-Firewall

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/waf/logs` | `waf:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/waf/logs` | `waf:write` | Modulprotokolle leeren |
| GET | `/api/waf/instances` | `waf:read` | WAF-Instanzen auflisten |
| POST | `/api/waf/instances` | `waf:write` | WAF-Instanzen hinzufügen |
| PUT | `/api/waf/instances/{key}` | `waf:write` | WAF-Instanzen ändern |
| DELETE | `/api/waf/instances/{key}` | `waf:write` | WAF-Instanzen löschen |
| PUT | `/api/waf/instances/{key}/enable` | `waf:write` | WAF-Instanzen aktivieren oder deaktivieren |
| PUT | `/api/waf/instances-order` | `waf:write` | WAF-Instanzen neu anordnen |
| GET | `/api/waf/crs` | `waf:read` | OWASP-CRS-Regeldateien auflisten |
| GET | `/api/waf/crs/{file}` | `waf:read` | Eine CRS-Regeldatei lesen |

## Web-Dienst

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/webservice/logs` | `webservice:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/webservice/logs` | `webservice:write` | Modulprotokolle leeren |
| GET | `/api/webservice/listeners` | `webservice:read` | Web-Dienst lauscht auflisten |
| POST | `/api/webservice/listeners` | `webservice:write` | Web-Dienst lauscht hinzufügen |
| PUT | `/api/webservice/listeners/{key}` | `webservice:write` | Web-Dienst lauscht ändern |
| DELETE | `/api/webservice/listeners/{key}` | `webservice:write` | Web-Dienst lauscht löschen |
| PUT | `/api/webservice/listeners/{key}/enable` | `webservice:write` | Web-Dienst lauscht aktivieren oder deaktivieren |
| PUT | `/api/webservice/listeners-order` | `webservice:write` | Web-Dienst lauscht neu anordnen |
| GET | `/api/webservice/settings` | `webservice:read` | Globale Web-Dienst-Einstellungen lesen |
| PUT | `/api/webservice/settings` | `webservice:write` | Globale Web-Dienst-Einstellungen speichern |
| GET | `/api/webservice/stats` | `webservice:read` | Minutenstatistik einer Site (?listener=&site=&minutes=) |
| GET | `/api/webservice/metrics` | `webservice:read` | Metriken im Prometheus-Textformat |
| GET | `/api/webservice/bans` | `webservice:read` | Aktuell gesperrte IPs |
| POST | `/api/webservice/bans` | `webservice:write` | IP manuell sperren |
| DELETE | `/api/webservice/bans` | `webservice:write` | Sperre aufheben (body ist {"key": Adresse}) |
| GET | `/api/webservice/geoip` | `webservice:read` | GeoIP-Datenbankstatus |
| POST | `/api/webservice/geoip/upload` | `webservice:write` | GeoIP-Datenbank hochladen (multipart-Feld file, .mmdb) |
| POST | `/api/webservice/geoip/download` | `webservice:write` | GeoIP-Datenbank (.mmdb) von einer Adresse herunterladen |
| DELETE | `/api/webservice/geoip` | `webservice:write` | GeoIP-Datenbank entfernen |

## Wake-on-LAN

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/wol/logs` | `wol:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/wol/logs` | `wol:write` | Modulprotokolle leeren |
| GET | `/api/wol/devices` | `wol:read` | Gerät aufwecken auflisten |
| POST | `/api/wol/devices` | `wol:write` | Gerät aufwecken hinzufügen |
| PUT | `/api/wol/devices/{key}` | `wol:write` | Gerät aufwecken ändern |
| DELETE | `/api/wol/devices/{key}` | `wol:write` | Gerät aufwecken löschen |
| PUT | `/api/wol/devices/{key}/enable` | `wol:write` | Gerät aufwecken aktivieren oder deaktivieren |
| PUT | `/api/wol/devices-order` | `wol:write` | Gerät aufwecken neu anordnen |
| POST | `/api/wol/devices/{key}/wake` | `wol:write` | Wake-on-LAN-Magic-Packet an ein Gerät senden |
| GET | `/api/wol/status` | `wol:read` | Online-Status eines Geräts anzeigen |

## Web-Terminal

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/webterminal/logs` | `webterminal:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/webterminal/logs` | `webterminal:write` | Modulprotokolle leeren |
| GET | `/api/webterminal/connections` | `webterminal:read` | Terminalverbindungen auflisten |
| POST | `/api/webterminal/connections` | `webterminal:write` | Terminalverbindungen hinzufügen |
| PUT | `/api/webterminal/connections/{key}` | `webterminal:write` | Terminalverbindungen ändern |
| DELETE | `/api/webterminal/connections/{key}` | `webterminal:write` | Terminalverbindungen löschen |
| PUT | `/api/webterminal/connections/{key}/enable` | `webterminal:write` | Terminalverbindungen aktivieren oder deaktivieren |
| PUT | `/api/webterminal/connections-order` | `webterminal:write` | Terminalverbindungen neu anordnen |
| DELETE | `/api/webterminal/connections/{key}/hostkey` | `webterminal:write` | Gespeicherten Host-Schlüssel löschen |
| POST | `/api/webterminal/connections/{key}/test` | `webterminal:write` | Gespeicherte Verbindung testen |
| POST | `/api/webterminal/connections/test` | `webterminal:write` | Temporäre Verbindungsparameter testen |
| PUT | `/api/webterminal/connections/{key}/quickaccess` | `webterminal:write` | Schnellzugriff für Verbindungen festlegen |
| GET | `/api/webterminal/credentials` | `webterminal:read` | Einträge auflisten |
| POST | `/api/webterminal/credentials` | `webterminal:write` | Einträge hinzufügen |
| PUT | `/api/webterminal/credentials/{key}` | `webterminal:write` | Einträge ändern |
| DELETE | `/api/webterminal/credentials/{key}` | `webterminal:write` | Einträge löschen |
| PUT | `/api/webterminal/credentials/{key}/enable` | `webterminal:write` | Einträge aktivieren oder deaktivieren |
| PUT | `/api/webterminal/credentials-order` | `webterminal:write` | Einträge neu anordnen |
| GET | `/api/webterminal/connections/{key}/ws` | `webterminal:write` | Terminalsitzung öffnen (WebSocket) |
| GET | `/api/webterminal/sessions/{id}/ws` | `webterminal:write` | An bestehende Terminalsitzung anbinden (WebSocket) |
| GET | `/api/webterminal/sessions` | `webterminal:read` | Terminalsitzungen auflisten |
| GET | `/api/webterminal/sessions/{id}` | `webterminal:read` | Details einer Terminalsitzung anzeigen |
| PUT | `/api/webterminal/sessions/{id}/remark` | `webterminal:write` | Notiz einer Terminalsitzung ändern |
| GET | `/api/webterminal/sessions/{id}/stats` | `webterminal:read` | Statistik der Terminalsitzungen anzeigen |
| DELETE | `/api/webterminal/sessions/{id}` | `webterminal:write` | Terminalsitzung schließen |
| GET | `/api/webterminal/security` | Nur Anmeldesitzung | Status der Terminal-Sicherheitseinstellungen anzeigen |
| POST | `/api/webterminal/security/agreement` | Nur Anmeldesitzung | Terminal-Nutzungsvereinbarung bestätigen |
| POST | `/api/webterminal/security/2fa/setup` | Nur Anmeldesitzung | Schlüssel für Zwei-Faktor-Authentifizierung des Terminals erzeugen |
| POST | `/api/webterminal/security/2fa/enable` | Nur Anmeldesitzung | Zwei-Faktor-Authentifizierung des Terminals aktivieren |
| POST | `/api/webterminal/security/2fa/disable` | Nur Anmeldesitzung | Zwei-Faktor-Authentifizierung des Terminals deaktivieren |
| POST | `/api/webterminal/security/verify2fa` | Nur Anmeldesitzung | Code der Zwei-Faktor-Authentifizierung des Terminals prüfen |
| GET | `/api/webterminal/security/check2fa` | Nur Anmeldesitzung | Prüfen, ob die Zwei-Faktor-Authentifizierung des Terminals bestanden wurde |
| GET | `/api/webterminal/sftp/{key}/list` | `webterminal:read` | Remote-Verzeichnis auflisten |
| GET | `/api/webterminal/sftp/{key}/read` | `webterminal:write` | Remote-Datei lesen |
| POST | `/api/webterminal/sftp/{key}/write` | `webterminal:write` | Remote-Datei schreiben |
| POST | `/api/webterminal/sftp/{key}/upload` | `webterminal:write` | Datei auf den Remote-Host hochladen |
| POST | `/api/webterminal/sftp/{key}/mkdir` | `webterminal:write` | Remote-Verzeichnis erstellen |
| POST | `/api/webterminal/sftp/{key}/touch` | `webterminal:write` | Leere Remote-Datei erstellen |
| POST | `/api/webterminal/sftp/{key}/rename` | `webterminal:write` | Remote-Datei umbenennen oder verschieben |
| POST | `/api/webterminal/sftp/{key}/copy` | `webterminal:write` | Remote-Datei kopieren |
| POST | `/api/webterminal/sftp/{key}/chmod` | `webterminal:write` | Berechtigungen einer Remote-Datei ändern |
| DELETE | `/api/webterminal/sftp/{key}/remove` | `webterminal:write` | Remote-Datei löschen |
| POST | `/api/webterminal/sftp/{key}/compress` | `webterminal:write` | Datei auf dem Remote-Host komprimieren |
| POST | `/api/webterminal/sftp/{key}/decompress` | `webterminal:write` | Datei auf dem Remote-Host entpacken |
| GET | `/api/webterminal/sftp/{key}/preview-archive` | `webterminal:write` | Inhalt eines Remote-Archivs in der Vorschau anzeigen |
| GET | `/api/webterminal/settings` | `webterminal:read` | Web-Terminal-Einstellungen lesen |
| PUT | `/api/webterminal/settings` | `webterminal:write` | Web-Terminal-Einstellungen speichern |
| GET | `/api/webterminal/shells` | `webterminal:read` | Auf diesem Rechner verfügbare Shells auflisten |
| GET | `/api/webterminal/globalshortcuts` | `webterminal:read` | Globale Schnellbefehle lesen |
| PUT | `/api/webterminal/globalshortcuts` | `webterminal:write` | Globale Schnellbefehle speichern |
| GET | `/api/webterminal/splitlayout` | `webterminal:read` | Split-Layout lesen |
| PUT | `/api/webterminal/splitlayout` | `webterminal:write` | Split-Layout speichern |
| DELETE | `/api/webterminal/splitlayout` | `webterminal:write` | Split-Layout zurücksetzen |

## Geplante Aufgaben

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/cron/logs` | `cron:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/cron/logs` | `cron:write` | Modulprotokolle leeren |
| GET | `/api/cron/tasks` | `cron:read` | Geplante Aufgaben auflisten |
| POST | `/api/cron/tasks` | `cron:write` | Geplante Aufgaben hinzufügen |
| PUT | `/api/cron/tasks/{key}` | `cron:write` | Geplante Aufgaben ändern |
| DELETE | `/api/cron/tasks/{key}` | `cron:write` | Geplante Aufgaben löschen |
| PUT | `/api/cron/tasks/{key}/enable` | `cron:write` | Geplante Aufgaben aktivieren oder deaktivieren |
| PUT | `/api/cron/tasks-order` | `cron:write` | Geplante Aufgaben neu anordnen |
| POST | `/api/cron/tasks/{key}/run` | `cron:write` | Eine geplante Aufgabe sofort ausführen |
| POST | `/api/cron/parse` | `cron:write` | Cron-Ausdruck parsen und Vorschau anzeigen |

## Docker-Verwaltung

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/docker/logs` | `docker:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/docker/logs` | `docker:write` | Modulprotokolle leeren |
| GET | `/api/docker/tasks` | `docker:read` | Hintergrundaufgaben auflisten |
| GET | `/api/docker/tasks/{id}` | `docker:read` | Fortschritt und Ausgabe einer Hintergrundaufgabe anzeigen |
| DELETE | `/api/docker/tasks` | `docker:write` | Beendete Hintergrundaufgaben bereinigen |
| GET | `/api/docker/images` | `docker:read` | Images auflisten |
| POST | `/api/docker/images/pull` | `docker:write` | Image pullen |
| POST | `/api/docker/images/push` | `docker:write` | Image pushen |
| POST | `/api/docker/images/search` | `docker:write` | Images suchen |
| POST | `/api/docker/images/load` | `docker:write` | Image-Datei importieren |
| POST | `/api/docker/images/build` | `docker:write` | Image bauen |
| GET | `/api/docker/images/{id}` | `docker:read` | Image-Details anzeigen |
| POST | `/api/docker/images/{id}/tag` | `docker:write` | Image taggen |
| DELETE | `/api/docker/images/{id}` | `docker:write` | Image löschen |
| GET | `/api/docker/images/{id}/export` | `docker:write` | Image exportieren |
| GET | `/api/docker/networks` | `docker:read` | Netzwerke auflisten |
| POST | `/api/docker/networks` | `docker:write` | Netzwerk erstellen |
| DELETE | `/api/docker/networks/{id}` | `docker:write` | Netzwerk löschen |
| POST | `/api/docker/networks/{id}/connect` | `docker:write` | Container mit einem Netzwerk verbinden |
| POST | `/api/docker/networks/{id}/disconnect` | `docker:write` | Container vom Netzwerk trennen |
| GET | `/api/docker/volumes` | `docker:read` | Volumes auflisten |
| POST | `/api/docker/volumes` | `docker:write` | Volume erstellen |
| DELETE | `/api/docker/volumes/{name}` | `docker:write` | Volume löschen |
| GET | `/api/docker/volumes/{name}/export` | `docker:write` | Volume exportieren |
| POST | `/api/docker/volumes/{name}/import` | `docker:write` | Daten in ein Volume importieren |
| GET | `/api/docker/compose/projects` | `docker:read` | Compose-Projekte auflisten |
| POST | `/api/docker/compose/projects` | `docker:write` | Compose-Projekt erstellen |
| POST | `/api/docker/compose/{project}/up` | `docker:write` | Compose-Projekt starten |
| POST | `/api/docker/compose/{project}/pull` | `docker:write` | Images eines Compose-Projekts pullen |
| POST | `/api/docker/compose/{project}/{action}` | `docker:write` | Aktion auf Compose-Projekt ausführen |
| GET | `/api/docker/compose/{project}/config` | `docker:write` | Compose-Konfigurationsdatei lesen |
| PUT | `/api/docker/compose/{project}/config` | `docker:write` | Compose-Konfigurationsdatei speichern |
| GET | `/api/docker/compose/{project}/logs` | `docker:read` | Protokolle eines Compose-Projekts anzeigen |
| DELETE | `/api/docker/compose/{project}` | `docker:write` | Compose-Projekt löschen |
| GET | `/api/docker/info` | `docker:read` | Docker-Engine-Informationen anzeigen |
| GET | `/api/docker/disk-usage` | `docker:read` | Docker-Speicherbelegung anzeigen |
| POST | `/api/docker/prune` | `docker:write` | Ungenutzte Docker-Ressourcen bereinigen |
| GET | `/api/docker/settings` | `docker:read` | Docker-Moduleinstellungen lesen |
| PUT | `/api/docker/settings` | `docker:write` | Docker-Moduleinstellungen speichern |
| GET | `/api/docker/containers` | `docker:read` | Container auflisten |
| POST | `/api/docker/containers` | `docker:write` | Container erstellen |
| POST | `/api/docker/containers/check-all` | `docker:write` | Alle Container-Images auf Updates prüfen |
| POST | `/api/docker/containers/upgrade-batch` | `docker:write` | Container gesammelt aktualisieren |
| POST | `/api/docker/images/upgrade-check` | `docker:write` | Prüfen, ob für ein Image eine neue Version vorliegt |
| GET | `/api/docker/containers/{id}` | `docker:read` | Container-Details anzeigen |
| DELETE | `/api/docker/containers/{id}` | `docker:write` | Container löschen |
| POST | `/api/docker/containers/{id}/edit` | `docker:write` | Container-Konfiguration ändern und neu erstellen |
| POST | `/api/docker/containers/{id}/rename` | `docker:write` | Container umbenennen |
| POST | `/api/docker/containers/{id}/commit` | `docker:write` | Container als Image committen |
| POST | `/api/docker/containers/{id}/check` | `docker:write` | Prüfen, ob für ein Container-Image Updates vorliegen |
| POST | `/api/docker/containers/{id}/upgrade` | `docker:write` | Container-Image aktualisieren und neu erstellen |
| POST | `/api/docker/containers/{id}/{action}` | `docker:write` | Aktion auf Container ausführen |
| GET | `/api/docker/containers/{id}/logs` | `docker:read` | Container-Protokolle anzeigen |
| GET | `/api/docker/containers/{id}/stats` | `docker:read` | Ressourcenverbrauch des Containers anzeigen |
| GET | `/api/docker/containers/{id}/processes` | `docker:read` | Prozesse im Container anzeigen |
| GET | `/api/docker/containers/{id}/files/list` | `docker:read` | Verzeichnis im Container auflisten |
| GET | `/api/docker/containers/{id}/files/read` | `docker:write` | Datei im Container lesen |
| GET | `/api/docker/containers/{id}/files/download` | `docker:write` | Datei aus dem Container herunterladen |
| POST | `/api/docker/containers/{id}/files/write` | `docker:write` | Datei im Container schreiben |
| POST | `/api/docker/containers/{id}/files/mkdir` | `docker:write` | Verzeichnis im Container erstellen |
| POST | `/api/docker/containers/{id}/files/rename` | `docker:write` | Datei im Container umbenennen |
| POST | `/api/docker/containers/{id}/files/upload` | `docker:write` | Datei in den Container hochladen |
| DELETE | `/api/docker/containers/{id}/files` | `docker:write` | Datei im Container löschen |
| GET | `/api/docker/containers/{id}/exec` | `docker:write` | Terminal im Container öffnen (WebSocket) |

## WebDAV

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/webdav/logs` | `webdav:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/webdav/logs` | `webdav:write` | Modulprotokolle leeren |
| GET | `/api/webdav/settings` | `webdav:read` | WebDAV-Einstellungen lesen |
| PUT | `/api/webdav/settings` | `webdav:write` | WebDAV-Einstellungen speichern |

## FTP-Dienst

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/ftp/logs` | `ftp:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/ftp/logs` | `ftp:write` | Modulprotokolle leeren |
| GET | `/api/ftp/settings` | `ftp:read` | FTP-Einstellungen lesen |
| PUT | `/api/ftp/settings` | `ftp:write` | FTP-Einstellungen speichern |

## Speicherverwaltung

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/storage/logs` | `storage:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/storage/logs` | `storage:write` | Modulprotokolle leeren |
| GET | `/api/storage/mounts` | `storage:read` | Einträge auflisten |
| POST | `/api/storage/mounts` | `storage:write` | Einträge hinzufügen |
| PUT | `/api/storage/mounts/{key}` | `storage:write` | Einträge ändern |
| DELETE | `/api/storage/mounts/{key}` | `storage:write` | Einträge löschen |
| PUT | `/api/storage/mounts/{key}/enable` | `storage:write` | Einträge aktivieren oder deaktivieren |
| PUT | `/api/storage/mounts-order` | `storage:write` | Einträge neu anordnen |
| GET | `/api/storage/files/{key}/list` | `storage:read` | Speicherverzeichnis auflisten |
| GET | `/api/storage/files/{key}/download` | `storage:write` | Speicherdatei herunterladen |
| POST | `/api/storage/files/{key}/upload` | `storage:write` | Datei in den Speicher hochladen |
| POST | `/api/storage/files/{key}/mkdir` | `storage:write` | Speicherverzeichnis erstellen |
| POST | `/api/storage/files/{key}/rename` | `storage:write` | Speicherdatei umbenennen oder verschieben |
| DELETE | `/api/storage/files/{key}` | `storage:write` | Speicherdatei löschen |

## Benachrichtigungszentrale

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/notify/logs` | `notify:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/notify/logs` | `notify:write` | Modulprotokolle leeren |
| GET | `/api/notify/channels` | `notify:read` | Benachrichtigungskanäle auflisten |
| POST | `/api/notify/channels` | `notify:write` | Benachrichtigungskanäle hinzufügen |
| PUT | `/api/notify/channels/{key}` | `notify:write` | Benachrichtigungskanäle ändern |
| DELETE | `/api/notify/channels/{key}` | `notify:write` | Benachrichtigungskanäle löschen |
| PUT | `/api/notify/channels/{key}/enable` | `notify:write` | Benachrichtigungskanäle aktivieren oder deaktivieren |
| PUT | `/api/notify/channels-order` | `notify:write` | Benachrichtigungskanäle neu anordnen |
| GET | `/api/notify/rules` | `notify:read` | Benachrichtigungsregeln auflisten |
| POST | `/api/notify/rules` | `notify:write` | Benachrichtigungsregeln hinzufügen |
| PUT | `/api/notify/rules/{key}` | `notify:write` | Benachrichtigungsregeln ändern |
| DELETE | `/api/notify/rules/{key}` | `notify:write` | Benachrichtigungsregeln löschen |
| PUT | `/api/notify/rules/{key}/enable` | `notify:write` | Benachrichtigungsregeln aktivieren oder deaktivieren |
| PUT | `/api/notify/rules-order` | `notify:write` | Benachrichtigungsregeln neu anordnen |
| GET | `/api/notify/event-types` | `notify:read` | Abonnierbare Ereignistypen |
| GET | `/api/notify/history` | `notify:read` | Letzte Zustellungsprotokolle |
| POST | `/api/notify/channels/{key}/test` | `notify:write` | Testbenachrichtigung an einen Kanal senden |

## Netzwerkdiagnose

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/nettools/logs` | `nettools:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/nettools/logs` | `nettools:write` | Modulprotokolle leeren |
| POST | `/api/nettools/run` | `nettools:write` | Diagnosewerkzeug ausführen und Aufgabe zurückgeben (tool: ping / traceroute / tcp / dns / speed / scan) |
| GET | `/api/nettools/tasks` | `nettools:read` | Letzte Diagnoseaufgaben |
| GET | `/api/nettools/tasks/{id}` | `nettools:read` | Aufgabendetails und Ausgabe |
| DELETE | `/api/nettools/tasks/{id}` | `nettools:write` | Laufende Aufgabe abbrechen oder beendete Aufgabe löschen |
| GET | `/api/nettools/settings` | `nettools:read` | Diagnoseeinstellungen |
| PUT | `/api/nettools/settings` | `nettools:write` | Liste gesperrter Scan-Ziele ändern |
| GET | `/api/nettools/networks` | `nettools:read` | Lokale Netzsegmente, die für den LAN-Scan verwendet werden können |

## Konfigurations-Backup

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/backup/logs` | `backup:read` | Laufzeitprotokolle eines Moduls anzeigen |
| DELETE | `/api/backup/logs` | `backup:write` | Modulprotokolle leeren |
| GET | `/api/backup/targets` | `backup:read` | Backup-Ziele auflisten |
| POST | `/api/backup/targets` | `backup:write` | Backup-Ziele hinzufügen |
| PUT | `/api/backup/targets/{key}` | `backup:write` | Backup-Ziele ändern |
| DELETE | `/api/backup/targets/{key}` | `backup:write` | Backup-Ziele löschen |
| PUT | `/api/backup/targets/{key}/enable` | `backup:write` | Backup-Ziele aktivieren oder deaktivieren |
| PUT | `/api/backup/targets-order` | `backup:write` | Backup-Ziele neu anordnen |
| GET | `/api/backup/settings` | `backup:read` | Backup-Einstellungen |
| PUT | `/api/backup/settings` | `backup:write` | Backup-Einstellungen ändern (Passphrase als Platzhalter übergeben, um sie beizubehalten) |
| POST | `/api/backup/run` | `backup:write` | Sofort ein Backup ausführen |
| GET | `/api/backup/status` | `backup:read` | Ergebnis des letzten Backups und nächster Ausführungszeitpunkt |
| GET | `/api/backup/mounts` | `backup:read` | Speicher-Mountpunkte, die als Backup-Ziel dienen können |
| GET | `/api/backup/local` | Nur Anmeldesitzung | Liste lokaler Backup-Dateien |
| GET | `/api/backup/local/{name}` | Nur Anmeldesitzung | Lokale Backup-Datei herunterladen |

## System

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/session` | Beliebiges gültiges Token | Aktuelle Authentifizierungsidentität anzeigen |
| GET | `/api/status` | `status:read` | System- und Modulstatus anzeigen |
| GET | `/api/logs` | `logs:read` | Systemprotokolle anzeigen |
| GET | `/api/system/settings` | Nur Anmeldesitzung | Globale Einstellungen lesen |
| PUT | `/api/system/settings` | Nur Anmeldesitzung | Globale Einstellungen speichern |
| GET | `/api/system/info` | `status:read` | Systeminformationen anzeigen |
| GET | `/api/system/update` | `status:read` | Nach neuer Version suchen |
| GET | `/api/system/backup` | Nur Anmeldesitzung | Konfigurations-Backup herunterladen |
| GET | `/api/system/diagnostics` | Nur Anmeldesitzung | Anonymisiertes Diagnosepaket exportieren |
| GET | `/api/audit` | Nur Anmeldesitzung | Audit-Protokoll abfragen |
| GET | `/api/metrics` | `metrics:read` | Globale Prometheus-Metriken |
| GET | `/api/history` | Nur Anmeldesitzung | Verlaufsversionen der Konfiguration auflisten |
| GET | `/api/history/{name}/{id}/diff` | Nur Anmeldesitzung | Verlaufsversion mit der aktuellen Konfiguration vergleichen |
| POST | `/api/history/{name}/{id}/restore` | Nur Anmeldesitzung | Konfiguration auf eine Verlaufsversion zurücksetzen |
| POST | `/api/system/restore` | Nur Anmeldesitzung | Konfiguration aus Backup wiederherstellen |
| POST | `/api/system/restart` | Nur Anmeldesitzung | Gately neu starten |
| POST | `/api/system/selfcheck` | Nur Anmeldesitzung | Selbsttest von Netzwerk und Konfiguration |
| GET | `/api/system/onboarding` | Nur Anmeldesitzung | Ob die Ersteinrichtung aussteht |
| POST | `/api/system/onboarding/dismiss` | Nur Anmeldesitzung | Ersteinrichtung abschließen oder überspringen |
| GET | `/api/openapi.json` | Beliebiges gültiges Token | OpenAPI-Beschreibung dieser API abrufen |

## Konfigurationsimport

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| POST | `/api/import/lucky` | `import:write` | Lucky-Konfiguration importieren |

## API-Token

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| GET | `/api/tokens` | Nur Anmeldesitzung | API-Tokens auflisten |
| GET | `/api/tokens/scopes` | Nur Anmeldesitzung | Autorisierbare Berechtigungsbereiche auflisten |
| POST | `/api/tokens` | Nur Anmeldesitzung | API-Token erstellen |
| PUT | `/api/tokens/{id}` | Nur Anmeldesitzung | Name, Berechtigungen, Ablaufzeit und Aktivierungsstatus eines API-Tokens ändern |
| POST | `/api/tokens/{id}/regenerate` | Nur Anmeldesitzung | Token-Klartext neu generieren |
| DELETE | `/api/tokens/{id}` | Nur Anmeldesitzung | API-Token löschen |

## Konto

| Methode | Pfad | Erforderliche Berechtigung | Beschreibung |
| --- | --- | --- | --- |
| PUT | `/api/account` | Nur Anmeldesitzung | Administrator-Benutzername und -Passwort ändern |
| GET | `/api/account/passkeys` | Nur Anmeldesitzung | Registrierte Passkeys auflisten |
| POST | `/api/account/passkeys/begin` | Nur Anmeldesitzung | Passkey-Registrierung starten |
| POST | `/api/account/passkeys` | Nur Anmeldesitzung | Passkey-Registrierung abschließen |
| DELETE | `/api/account/passkeys/{id}` | Nur Anmeldesitzung | Passkey löschen |
| GET | `/api/account/users` | Nur Anmeldesitzung | Schreibgeschützte Konten auflisten |
| POST | `/api/account/users` | Nur Anmeldesitzung | Schreibgeschütztes Konto erstellen |
| PUT | `/api/account/users/{name}` | Nur Anmeldesitzung | Passwort eines schreibgeschützten Kontos zurücksetzen |
| DELETE | `/api/account/users/{name}` | Nur Anmeldesitzung | Schreibgeschütztes Konto löschen |
| GET | `/api/account/totp` | Nur Anmeldesitzung | Status der Zwei-Faktor-Authentifizierung anzeigen |
| POST | `/api/account/totp/setup` | Nur Anmeldesitzung | Schlüssel für Zwei-Faktor-Authentifizierung erzeugen |
| POST | `/api/account/totp/enable` | Nur Anmeldesitzung | Zwei-Faktor-Authentifizierung aktivieren |
| POST | `/api/account/totp/disable` | Nur Anmeldesitzung | Zwei-Faktor-Authentifizierung deaktivieren |

## Datenstrukturen

### backup.LocalFile

| Feld | Typ |
| --- | --- |
| `modTime` | Zeit (RFC 3339) |
| `name` | string |
| `size` | integer |

### backup.Result

| Feld | Typ |
| --- | --- |
| `error` | string |
| `file` | string |
| `ok` | boolean |
| `pruned` | integer |
| `target` | string |

### backup.Settings

| Feld | Typ |
| --- | --- |
| `enable` | boolean |
| `keep` | integer |
| `localEnable` | boolean |
| `password` | string |
| `schedule` | string |

### backup.Status

| Feld | Typ |
| --- | --- |
| `lastRun` | Zeit (RFC 3339) |
| `manual` | boolean |
| `nextRun` | Zeit (RFC 3339) |
| `results` | Array von `backup.Result` |
| `running` | boolean |

### backup.Target

| Feld | Typ |
| --- | --- |
| `dir` | string |
| `enable` | boolean |
| `key` | string |
| `mount` | string |
| `name` | string |

### backup.settingsView

| Feld | Typ |
| --- | --- |
| `enable` | boolean |
| `hasPassword` | boolean |
| `keep` | integer |
| `localEnable` | boolean |
| `password` | string |
| `schedule` | string |

### cloudflared.Tunnel

| Feld | Typ |
| --- | --- |
| `edgeIpVersion` | string |
| `enable` | boolean |
| `key` | string |
| `mode` | string |
| `name` | string |
| `protocol` | string |
| `remark` | string |
| `service` | string |
| `token` | string |

### cron.Task

| Feld | Typ |
| --- | --- |
| `command` | string |
| `enable` | boolean |
| `http` | `webhook.Config` |
| `key` | string |
| `name` | string |
| `remark` | string |
| `spec` | string |
| `type` | string |

### crud.enableRequest

| Feld | Typ |
| --- | --- |
| `enable` | boolean |

### crud.orderRequest

| Feld | Typ |
| --- | --- |
| `keys` | Array von string |

### ddns.Record

| Feld | Typ |
| --- | --- |
| `enable` | boolean |
| `key` | string |
| `line` | string |
| `remark` | string |
| `sub` | string |
| `ttl` | integer |
| `type` | string |
| `value` | string |
| `zone` | string |

### ddns.Task

| Feld | Typ |
| --- | --- |
| `credentials` | `dnsprovider.Credentials` |
| `enable` | boolean |
| `forceInterval` | integer |
| `httpTimeout` | integer |
| `interval` | integer |
| `ipv4` | `ipsource.Source` |
| `ipv6` | `ipsource.Source` |
| `key` | string |
| `name` | string |
| `provider` | string |
| `records` | Array von `ddns.Record` |
| `remark` | string |
| `webhook` | `webhook.Config` |

### dnsprovider.Credentials

| Feld | Typ |
| --- | --- |
| `extra` | Map von string |
| `id` | string |
| `secret` | string |

### events.TypeInfo

| Feld | Typ |
| --- | --- |
| `level` | string |
| `name` | string |
| `type` | string |

### frp.ClientSettings

| Feld | Typ |
| --- | --- |
| `binPath` | string |
| `enable` | boolean |
| `protocol` | string |
| `serverAddr` | string |
| `serverPort` | integer |
| `token` | string |
| `user` | string |

### frp.ImportRequest

| Feld | Typ |
| --- | --- |
| `apply` | boolean |
| `content` | string |
| `proxyMode` | string |
| `role` | string |

### frp.ImportResponse

| Feld | Typ |
| --- | --- |
| `added` | integer |
| `applied` | boolean |
| `client` | `frp.ClientSettings` |
| `proxies` | Array von `frp.Proxy` |
| `role` | string |
| `server` | `frp.ServerSettings` |
| `warnings` | Array von string |

### frp.Proxy

| Feld | Typ |
| --- | --- |
| `customDomains` | Array von string |
| `enable` | boolean |
| `key` | string |
| `localIp` | string |
| `localPort` | integer |
| `name` | string |
| `remark` | string |
| `remotePort` | integer |
| `secretKey` | string |
| `subdomain` | string |
| `type` | string |

### frp.ServerSettings

| Feld | Typ |
| --- | --- |
| `allowPorts` | string |
| `binPath` | string |
| `bindPort` | integer |
| `dashboardAddr` | string |
| `dashboardPort` | integer |
| `dashboardPwd` | string |
| `dashboardUser` | string |
| `enable` | boolean |
| `subdomainHost` | string |
| `token` | string |
| `vhostHttpPort` | integer |
| `vhostHttpsPort` | integer |

### geoip.Info

| Feld | Typ |
| --- | --- |
| `buildTime` | Zeit (RFC 3339) |
| `loaded` | boolean |
| `size` | integer |
| `type` | string |

### ipsource.Source

| Feld | Typ |
| --- | --- |
| `command` | string |
| `enable` | boolean |
| `interface` | string |
| `regex` | string |
| `type` | string |
| `urls` | Array von string |

### nettools.Request

| Feld | Typ |
| --- | --- |
| `cidr` | string |
| `count` | integer |
| `ports` | string |
| `seconds` | integer |
| `server` | string |
| `target` | string |
| `tool` | string |
| `type` | string |
| `url` | string |

### nettools.Settings

| Feld | Typ |
| --- | --- |
| `deny` | Array von string |

### nettools.Task

| Feld | Typ |
| --- | --- |
| `error` | string |
| `finished` | Zeit (RFC 3339) |
| `id` | string |
| `lines` | Array von string |
| `result` | Beliebig |
| `started` | Zeit (RFC 3339) |
| `status` | string |
| `title` | string |
| `tool` | string |

### notify.Channel

| Feld | Typ |
| --- | --- |
| `enable` | boolean |
| `from` | string |
| `host` | string |
| `key` | string |
| `kind` | string |
| `name` | string |
| `port` | integer |
| `remark` | string |
| `secret` | string |
| `security` | string |
| `target` | string |
| `url` | string |
| `user` | string |
| `webhook` | `webhook.Config` |

### notify.Record

| Feld | Typ |
| --- | --- |
| `channel` | string |
| `error` | string |
| `ok` | boolean |
| `rule` | string |
| `time` | Zeit (RFC 3339) |
| `title` | string |
| `type` | string |

### notify.Rule

| Feld | Typ |
| --- | --- |
| `channels` | Array von string |
| `enable` | boolean |
| `events` | Array von string |
| `key` | string |
| `minLevel` | string |
| `name` | string |
| `silence` | integer |

### portforward.Rule

| Feld | Typ |
| --- | --- |
| `enable` | boolean |
| `key` | string |
| `listenAddress` | string |
| `listenPorts` | string |
| `maxConnections` | integer |
| `mode` | string |
| `name` | string |
| `protocols` | Array von string |
| `proxyProtocol` | string |
| `remark` | string |
| `sniRoutes` | Array von `portforward.SNIRoute` |
| `targetAddresses` | Array von string |
| `targetPorts` | string |
| `udpSessionTimeout` | integer |

### portforward.SNIRoute

| Feld | Typ |
| --- | --- |
| `domains` | Array von string |
| `name` | string |
| `proxyProtocol` | string |
| `targetAddresses` | Array von string |
| `targetPort` | integer |

### ssl.ACMEConfig

| Feld | Typ |
| --- | --- |
| `ca` | string |
| `challenge` | string |
| `credentials` | `dnsprovider.Credentials` |
| `directoryUrl` | string |
| `disableAuthoritativeCheck` | boolean |
| `domains` | Array von string |
| `eabHmac` | string |
| `eabKid` | string |
| `email` | string |
| `httpPort` | integer |
| `keyType` | string |
| `propagationTimeout` | integer |
| `provider` | string |
| `resolvers` | Array von string |
| `skipPropagationCheck` | boolean |
| `zone` | string |

### ssl.Cert

| Feld | Typ |
| --- | --- |
| `acme` | `ssl.ACMEConfig` |
| `certFile` | string |
| `certPem` | string |
| `enable` | boolean |
| `issuerPem` | string |
| `key` | string |
| `keyFile` | string |
| `keyPem` | string |
| `mapping` | `ssl.Mapping` |
| `remark` | string |
| `source` | string |
| `updatedAt` | Zeit (RFC 3339) |

### ssl.Mapping

| Feld | Typ |
| --- | --- |
| `dir` | string |
| `enable` | boolean |
| `script` | string |

### storage.Mount

| Feld | Typ |
| --- | --- |
| `enable` | boolean |
| `key` | string |
| `name` | string |
| `password` | string |
| `path` | string |
| `readOnly` | boolean |
| `remark` | string |
| `skipVerify` | boolean |
| `type` | string |
| `url` | string |
| `username` | string |

### storage.MountInfo

| Feld | Typ |
| --- | --- |
| `enable` | boolean |
| `key` | string |
| `name` | string |
| `readOnly` | boolean |
| `type` | string |

### stun.Bind

| Feld | Typ |
| --- | --- |
| `interface` | string |
| `ip` | string |
| `mode` | string |
| `regex` | string |

### stun.Rule

| Feld | Typ |
| --- | --- |
| `autoRetry` | boolean |
| `bind` | `stun.Bind` |
| `disableCheck` | boolean |
| `disableForward` | boolean |
| `enable` | boolean |
| `heartbeatInterval` | integer |
| `keepAliveServers` | Array von string |
| `key` | string |
| `localPort` | integer |
| `maxConnections` | integer |
| `name` | string |
| `natpmp` | boolean |
| `natpmpGateway` | string |
| `protocol` | string |
| `remark` | string |
| `retryInterval` | integer |
| `script` | string |
| `scriptEnable` | boolean |
| `stunServers` | Array von string |
| `stunTimeout` | integer |
| `targetAddresses` | Array von string |
| `targetPort` | integer |
| `udpSessionTimeout` | integer |
| `upnp` | boolean |
| `upnpControlUrl` | string |
| `upnpGateway` | string |
| `upnpInternalPort` | integer |
| `upnpLocalHost` | string |
| `useGlobalServers` | boolean |
| `useGlobalWebhook` | boolean |
| `webhook` | `webhook.Config` |
| `webhookOnlyOnChange` | boolean |

### waf.Instance

| Feld | Typ |
| --- | --- |
| `enable` | boolean |
| `inboundThreshold` | integer |
| `key` | string |
| `mode` | string |
| `name` | string |
| `outboundThreshold` | integer |
| `paranoia` | integer |
| `remark` | string |
| `ruleSets` | Array von `waf.RuleSet` |

### waf.RuleSet

| Feld | Typ |
| --- | --- |
| `enable` | boolean |
| `param` | string |
| `remark` | string |
| `type` | string |

### webhook.Config

| Feld | Typ |
| --- | --- |
| `body` | string |
| `enable` | boolean |
| `headers` | Array von string |
| `insecureSkipVerify` | boolean |
| `method` | string |
| `successContent` | Array von string |
| `timeout` | integer |
| `url` | string |

### webservice.BanInfo

| Feld | Typ |
| --- | --- |
| `key` | string |
| `manual` | boolean |
| `reason` | string |
| `until` | Zeit (RFC 3339) |

### webservice.CacheConfig

| Feld | Typ |
| --- | --- |
| `enable` | boolean |
| `maxMb` | integer |
| `ttl` | integer |

### webservice.ForwardAuth

| Feld | Typ |
| --- | --- |
| `copyHeaders` | Array von string |
| `loginUrl` | string |
| `url` | string |

### webservice.HealthCheck

| Feld | Typ |
| --- | --- |
| `enable` | boolean |
| `fails` | integer |
| `interval` | integer |
| `mode` | string |
| `passes` | integer |
| `path` | string |
| `timeout` | integer |

### webservice.Listener

| Feld | Typ |
| --- | --- |
| `acceptProxyProtocol` | boolean |
| `default` | `webservice.Site` |
| `enable` | boolean |
| `http3` | boolean |
| `key` | string |
| `listenIp` | string |
| `minTls` | string |
| `name` | string |
| `network` | string |
| `port` | integer |
| `remark` | string |
| `sites` | Array von `webservice.Site` |
| `tls` | boolean |

### webservice.Location

| Feld | Typ |
| --- | --- |
| `prefix` | string |
| `requestHeaders` | Array von string |
| `targets` | Array von string |

### webservice.PathRule

| Feld | Typ |
| --- | --- |
| `auth` | string |
| `ipList` | Array von string |
| `ipMode` | string |
| `prefix` | string |

### webservice.Point

| Feld | Typ |
| --- | --- |
| `bytes` | integer |
| `requests` | integer |
| `s4xx` | integer |
| `s5xx` | integer |
| `t` | integer |

### webservice.RateLimit

| Feld | Typ |
| --- | --- |
| `burst` | integer |
| `enable` | boolean |
| `maxConns` | integer |
| `rate` | number |

### webservice.Rewrite

| Feld | Typ |
| --- | --- |
| `code` | integer |
| `pattern` | string |
| `redirect` | boolean |
| `to` | string |

### webservice.ShareLink

| Feld | Typ |
| --- | --- |
| `expiresAt` | Zeit (RFC 3339) |
| `key` | string |
| `maxUses` | integer |
| `name` | string |
| `pathPrefix` | string |
| `token` | string |

### webservice.Site

| Feld | Typ |
| --- | --- |
| `accessLog` | boolean |
| `authOptional` | boolean |
| `basicAuth` | boolean |
| `cache` | `webservice.CacheConfig` |
| `cleanUrls` | boolean |
| `compress` | boolean |
| `contentType` | string |
| `cors` | boolean |
| `countries` | Array von string |
| `countryMode` | string |
| `disableKeepAlives` | boolean |
| `domains` | Array von string |
| `enable` | boolean |
| `forwardAuth` | `webservice.ForwardAuth` |
| `forwardedHeaders` | boolean |
| `health` | `webservice.HealthCheck` |
| `insecureSkipVerify` | boolean |
| `ipList` | Array von string |
| `ipMode` | string |
| `keepPath` | boolean |
| `key` | string |
| `lbPolicy` | string |
| `listDir` | boolean |
| `locations` | Array von `webservice.Location` |
| `name` | string |
| `notFoundPage` | string |
| `oauthUsers` | Array von string |
| `rateLimit` | `webservice.RateLimit` |
| `redirectCode` | integer |
| `redirectTo` | string |
| `remark` | string |
| `requestHeaders` | Array von string |
| `responseHeaders` | Array von string |
| `rewrites` | Array von `webservice.Rewrite` |
| `root` | string |
| `rules` | Array von `webservice.PathRule` |
| `sendProxyProtocol` | boolean |
| `shareLinks` | Array von `webservice.ShareLink` |
| `statusCode` | integer |
| `targets` | Array von string |
| `text` | string |
| `type` | string |
| `upstreamSniFromHost` | boolean |
| `useTargetHost` | boolean |
| `users` | Array von string |
| `waf` | string |
| `webAuth` | boolean |

### webservice.SiteStatsView

| Feld | Typ |
| --- | --- |
| `bytesOut` | integer |
| `series` | Array von `webservice.Point` |
| `status` | Map von integer |
| `topCountries` | Array von `webservice.TopItem` |
| `topIps` | Array von `webservice.TopItem` |
| `upstreams` | Array von `webservice.UpstreamStatus` |

### webservice.TopItem

| Feld | Typ |
| --- | --- |
| `count` | integer |
| `key` | string |

### webservice.UpstreamStatus

| Feld | Typ |
| --- | --- |
| `backup` | boolean |
| `conns` | integer |
| `healthy` | boolean |
| `url` | string |
| `weight` | integer |

### webservice.banRequest

| Feld | Typ |
| --- | --- |
| `key` | string |
| `seconds` | integer |

### webservice.geoipDownload

| Feld | Typ |
| --- | --- |
| `url` | string |

### webterminal.Connection

| Feld | Typ |
| --- | --- |
| `enable` | boolean |
| `key` | string |
| `local` | `webterminal.LocalConfig` |
| `name` | string |
| `quickAccessDirs` | Array von string |
| `remark` | string |
| `ssh` | `webterminal.SSHConfig` |
| `telnet` | `webterminal.TelnetConfig` |
| `type` | string |

### webterminal.Credential

| Feld | Typ |
| --- | --- |
| `authType` | string |
| `enable` | boolean |
| `key` | string |
| `keyFile` | string |
| `name` | string |
| `passphrase` | string |
| `password` | string |
| `privateKey` | string |
| `remark` | string |
| `username` | string |

### webterminal.LocalConfig

| Feld | Typ |
| --- | --- |
| `env` | Map von string |
| `shell` | string |
| `workDir` | string |

### webterminal.Proxy

| Feld | Typ |
| --- | --- |
| `enabled` | boolean |
| `host` | string |
| `password` | string |
| `port` | integer |
| `type` | string |
| `username` | string |

### webterminal.SSHConfig

| Feld | Typ |
| --- | --- |
| `authType` | string |
| `credentialKey` | string |
| `host` | string |
| `hostKey` | string |
| `keepAlive` | integer |
| `keyFile` | string |
| `passphrase` | string |
| `password` | string |
| `port` | integer |
| `privateKey` | string |
| `proxy` | `webterminal.Proxy` |
| `timeout` | integer |
| `username` | string |
| `workDir` | string |

### webterminal.TelnetConfig

| Feld | Typ |
| --- | --- |
| `host` | string |
| `password` | string |
| `port` | integer |
| `proxy` | `webterminal.Proxy` |
| `timeout` | integer |
| `username` | string |

### wol.Device

| Feld | Typ |
| --- | --- |
| `broadcastIps` | Array von string |
| `checkAddr` | string |
| `enable` | boolean |
| `key` | string |
| `macs` | Array von string |
| `name` | string |
| `port` | integer |
| `remark` | string |
| `repeat` | integer |

