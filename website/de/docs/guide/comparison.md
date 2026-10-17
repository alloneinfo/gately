# Vergleich mit Lucky

Gately ist von Lucky inspiriert, deckt die gängigen Funktionen quelloffen ab und kann bestehende Konfigurationen importieren. Lucky liegt bei Funktionsumfang und Nutzerbasis vorn, und ohne Lucky gäbe es Gately nicht. Vielen Dank dafür.

## Philosophie

| Punkt | Gately | Lucky |
| --- | --- | --- |
| Quellcode | Open Source (Apache-2.0) | Closed Source |
| Konfigurationsdatei | Einfaches JSON, Git-tauglich | Proprietäres Format (`.lkcf`) |
| Offizielle Dienste | Standardmäßig keine; alle ausgehenden Adressen sind änderbar und abschaltbar | Online-Updates, Relay für Anmeldung über Drittanbieter und weitere offizielle Dienste |
| Anmeldung über Drittanbieter | Eigene OAuth-Apps | Über den offiziellen Relay-Dienst |
| Upgrades | Docker / Paketmanager | Eingebautes Online-Selbst-Update |
| Funktionsumfang | Deckt die gängigen Kernfunktionen ab | Größer, z. B. DLNA und Cloud-Laufwerk-Einbindung |

## Funktionsabdeckung

| Funktion | Gately |
| --- | --- |
| Dynamisches DNS | ✅ |
| STUN-Traversal | ✅ |
| Portweiterleitung | ✅ |
| SSL-Zertifikate | ✅ |
| Web-Dienste (Reverse Proxy) | ✅ |
| WAF | ✅ |
| Tailscale / frp / cloudflared | ✅ |
| Docker-Verwaltung | ✅ |
| Web-Terminal | ✅ |
| WebDAV / FTP / Speicherverwaltung | ✅ |
| Wake-on-LAN / geplante Aufgaben | ✅ |
| Benachrichtigungszentrale / Konfigurationssicherung / Netzwerkdiagnose | ✅ |
| API-Token / OpenAPI / CLI / MCP | ✅ |
| DLNA, Cloud-Laufwerk-Einbindung, rclone, globale IP-Filterung, HTTP/3 | Geplant, Reihenfolge nach Bedarf der Community |

## Bewusste Unterschiede

- Kein eingebautes Online-Selbst-Update, das die Binärdatei überschreibt; Upgrades überlassen wir Ihrem Paketmanager oder Docker.
- Kein Konfigurations-Push aus der Ferne und keine Übermittlung von Geräteinformationen.
- Die Konfiguration liegt im Klartext, und sensible Felder werden in der API-Ausgabe maskiert.

## Migration

Beide lassen sich parallel betreiben, Modul für Modul umschalten und jederzeit zurückwechseln. Siehe die [Migrationsanleitung](/de/docs/guide/migration).
