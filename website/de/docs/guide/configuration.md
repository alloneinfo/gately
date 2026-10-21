# Konfiguration & Laufzeitoptionen

## Kommandozeilen-Parameter

| Parameter | Beschreibung |
| --- | --- |
| `-c` | Konfigurationsverzeichnis; lässt sich auch über die Umgebungsvariable `GATELY_CONFIG_DIR` setzen |
| `-listen` | Überschreibt die Listen-Adresse der Verwaltungsoberfläche, z. B. `:16680` |
| `-apply` | Verzeichnis für deklarative Konfiguration; lässt sich auch über die Umgebungsvariable `GATELY_APPLY_DIR` setzen, siehe unten |
| `-version` | Gibt die Versionsinformationen aus |

| Umgebungsvariable | Beschreibung |
| --- | --- |
| `GATELY_INITIAL_PASSWORD` | Initialpasswort des Administrators beim ersten Start (mindestens 8 Zeichen), ohne Angabe gilt das Standardpasswort `gately.cn`; ohne Wirkung, wenn das Konto bereits existiert |

## Konfigurationsverzeichnis

Die gesamte Konfiguration liegt im Konfigurationsverzeichnis, mit einer einfachen `<name>.json`-Textdatei pro Modul, atomar geschrieben und mit Dateirechten 0600. Damit können Sie:

- sie mit jedem Texteditor einsehen;
- das Verzeichnis in Git ablegen, jede Änderung per Diff prüfen und jederzeit zurückrollen;
- sichern, indem Sie einfach das Verzeichnis kopieren.

::: warning Hinweis
`ssl.json` enthält private Zertifikatsschlüssel. Stellen Sie vor dem Einchecken in die Versionsverwaltung sicher, dass das Repository privat ist, oder nehmen Sie die Datei in Ihre Ignore-Liste auf.
:::

## Deklarative Konfiguration

Mit `-apply <Verzeichnis>` kopiert Gately bei jedem Start die Dateien `<Name>.json` aus diesem Verzeichnis in das Konfigurationsverzeichnis; Dateien mit identischem Inhalt werden nicht neu geschrieben. So lässt sich die Konfiguration zentral aus einem Git-Repository, einer Kubernetes-ConfigMap oder einem Deployment-Skript ausrollen.

- Vor dem Schreiben wird geprüft, ob jede Datei gültiges JSON ist; ist eine Datei ungültig, verweigert Gately den Start;
- `account.json` und `tokens.json` dürfen nicht überschrieben werden; Konten und Token werden ausschließlich in der Verwaltungsoberfläche verwaltet;
- Der überschriebene alte Inhalt landet im [Konfigurationsverlauf](/de/docs/guide/audit) und lässt sich jederzeit zurücksetzen.

## Verzeichnis für externe Programme

Die Module Tailscale, frp und cloudflared betreiben die offiziellen externen Programme. Gately sucht die ausführbare Datei in dieser Reihenfolge:

1. dem in den Moduleinstellungen angegebenen Pfad;
2. `PATH`;
3. dem Unterverzeichnis `bin/` des Konfigurationsverzeichnisses.

Legen Sie bei Docker-Bereitstellungen die Binärdateien in das eingebundene Konfigurationsverzeichnis, zum Beispiel `./gately-data/bin/frpc`. Tailscale benötigt sowohl `tailscale` als auch `tailscaled`. Details finden Sie unter [Tailscale / frp / cloudflared](/de/docs/modules/tunnels).

## Laufzeitprotokolle

Die Laufzeitprotokolle des Systems und aller Module werden standardmäßig nach `logs/` im Konfigurationsverzeichnis geschrieben und sind daher nach einem Neustart oder Upgrade weiterhin auf der Seite Systemprotokolle und in jedem Modul sichtbar:

- Eine JSON-Lines-Datei pro Quelle: `logs/<Modul-ID>.log`, Systemprotokolle in `logs/gately.log`. Beim Start werden die neuesten Einträge, bis zu „Aufbewahrte Protokollzeilen“, aus der Datei wiederhergestellt;
- Neue Einträge werden gebündelt mit höchstens 1 Sekunde Verzögerung geschrieben. Überschreitet eine Datei das Limit (standardmäßig 2 MB), wird sie nach `.1` rotiert und nur diese eine ältere Datei behalten, jede Quelle belegt also höchstens das Doppelte des Limits;
- Unter Einstellungen → Netzwerk & Laufzeit → Laufzeitparameter lässt sich „Protokolle speichern“ abschalten oder das „Protokolldatei-Limit“ ändern, sofort wirksam. Abgeschaltet bleiben Protokolle nur im Arbeitsspeicher und werden beim Neustart gelöscht;
- Werden die Protokolle eines Moduls in der Oberfläche geleert, werden auch seine Dateien gelöscht;
- Konfigurationssicherungen enthalten weder `logs/` noch das Audit-Protokoll `audit.log`.

## Sicherung und Wiederherstellung

Unter „Einstellungen“ finden Sie Konfigurationssicherung und -wiederherstellung; dort liegt auch der Einstieg für den Import von Lucky-Konfigurationen.
