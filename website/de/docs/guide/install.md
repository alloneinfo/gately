# Installation

## Docker

Image: `ghcr.io/wangwanjie/gately`, verfügbar für amd64, arm64 und arm/v7.

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
```

### Docker Compose

```yaml
services:
  gately:
    image: ghcr.io/wangwanjie/gately:latest
    container_name: gately
    restart: unless-stopped
    # Portweiterleitung, STUN-Traversal und Web-Dienste lauschen direkt auf Ports des Hosts
    network_mode: host
    volumes:
      # Konfigurationsverzeichnis (einfaches JSON, kann in Git gesichert werden; ssl.json enthält private Zertifikatsschlüssel)
      - ./data:/etc/gately
      # Docker-Verwaltung (entfernen, wenn nicht benötigt)
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TZ=Asia/Shanghai
```

Das Image enthält `sh`, `bash` und `curl`, sodass gängige Benachrichtigungsskripte direkt laufen.

## Binärdatei

1. Laden Sie das Archiv für Ihre Plattform von der [Download-Seite](/de/download) herunter: Linux (amd64 / arm64 / arm / mips / mipsle / riscv64), macOS, Windows oder FreeBSD.
2. Entpacken Sie es und starten Sie:

```bash
./gately -c /etc/gately
```

### systemd-Beispiel

```ini
[Unit]
Description=Gately
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/gately -c /etc/gately
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## Erste Anmeldung

Die Verwaltungsoberfläche lauscht standardmäßig auf Port `16680`. Der Benutzername lautet `admin`, das Initialpasswort wird beim ersten Start ins Log geschrieben. Ändern Sie nach der ersten Anmeldung das Passwort und aktivieren Sie die Zwei-Faktor-Authentifizierung.

## Upgrade

Gately hat kein eingebautes Online-Selbst-Update; Upgrades übernimmt Ihre Bereitstellungsmethode:

- Docker: das neue Image mit `docker pull` laden und den Container neu erstellen (das Konfigurationsverzeichnis bleibt eingebunden);
- Binärdatei: die ausführbare Datei ersetzen und den Dienst neu starten.

Exportieren Sie vor dem Upgrade eine Konfigurationssicherung oder legen Sie das Konfigurationsverzeichnis in Git ab.
