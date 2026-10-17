# Konfiguration & Laufzeitoptionen

## Kommandozeilen-Parameter

| Parameter | Beschreibung |
| --- | --- |
| `-c` | Konfigurationsverzeichnis; lässt sich auch über die Umgebungsvariable `GATELY_CONFIG_DIR` setzen |
| `-listen` | Überschreibt die Listen-Adresse der Verwaltungsoberfläche, z. B. `:16680` |
| `-version` | Gibt die Versionsinformationen aus |

## Konfigurationsverzeichnis

Die gesamte Konfiguration liegt im Konfigurationsverzeichnis, mit einer einfachen `<name>.json`-Textdatei pro Modul, atomar geschrieben und mit Dateirechten 0600. Damit können Sie:

- sie mit jedem Texteditor einsehen;
- das Verzeichnis in Git ablegen, jede Änderung per Diff prüfen und jederzeit zurückrollen;
- sichern, indem Sie einfach das Verzeichnis kopieren.

::: warning Hinweis
`ssl.json` enthält private Zertifikatsschlüssel. Stellen Sie vor dem Einchecken in die Versionsverwaltung sicher, dass das Repository privat ist, oder nehmen Sie die Datei in Ihre Ignore-Liste auf.
:::

## Verzeichnis für externe Programme

Die Module Tailscale, frp und cloudflared betreiben die offiziellen externen Programme. Gately sucht die ausführbare Datei in dieser Reihenfolge:

1. dem in den Moduleinstellungen angegebenen Pfad;
2. `PATH`;
3. dem Unterverzeichnis `bin/` des Konfigurationsverzeichnisses.

Legen Sie bei Docker-Bereitstellungen die Binärdateien in das eingebundene Konfigurationsverzeichnis, zum Beispiel `./gately-data/bin/frpc`. Tailscale benötigt sowohl `tailscale` als auch `tailscaled`. Details finden Sie unter [Tailscale / frp / cloudflared](/de/docs/modules/tunnels).

## Sicherung und Wiederherstellung

Unter „Einstellungen“ finden Sie Konfigurationssicherung und -wiederherstellung; dort liegt auch der Einstieg für den Import von Lucky-Konfigurationen.
