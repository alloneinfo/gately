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

### systemd

Das Repository enthält eine vollständige Unit-Datei [deploy/systemd/gately.service](https://github.com/wangwanjie/gately/blob/main/deploy/systemd/gately.service), die bereits die nötigen Capabilities zum Lauschen auf privilegierten Ports gewährt:

```bash
sudo install -m 0755 gately /usr/local/bin/gately
sudo install -m 0644 gately.service /etc/systemd/system/gately.service
sudo systemctl daemon-reload && sudo systemctl enable --now gately
journalctl -u gately | grep -i "initial password"
```

### OpenWrt

Verwenden Sie das procd-Init-Skript [deploy/openwrt/gately.init](https://github.com/wangwanjie/gately/blob/main/deploy/openwrt/gately.init):

```bash
cp gately /usr/bin/gately && chmod +x /usr/bin/gately
cp gately.init /etc/init.d/gately && chmod +x /etc/init.d/gately
/etc/init.d/gately enable && /etc/init.d/gately start
```

## Erste Anmeldung

Die Verwaltungsoberfläche lauscht standardmäßig auf Port `16680`. Der Benutzername lautet `admin`, das Initialpasswort wird beim ersten Start ins Log geschrieben. Bei automatisierten Bereitstellungen können Sie das Initialpasswort über die Umgebungsvariable `GATELY_INITIAL_PASSWORD` vorgeben; dann wird kein Zufallspasswort erzeugt.

Bei der ersten Anmeldung erscheint ein Einrichtungsassistent, der nacheinander dazu auffordert, das Passwort zu ändern, Zwei-Faktor-Authentifizierung oder einen Passkey zu aktivieren, den Netzwerk-Selbsttest auszuführen, aus Lucky zu importieren und Benachrichtigungen einzurichten.

## Upgrade

Gately hat kein eingebautes Online-Selbst-Update; Upgrades übernimmt Ihre Bereitstellungsmethode:

- Docker: das neue Image mit `docker pull` laden und den Container neu erstellen (das Konfigurationsverzeichnis bleibt eingebunden);
- Binärdatei: die ausführbare Datei ersetzen und den Dienst neu starten.

Exportieren Sie vor dem Upgrade eine Konfigurationssicherung oder legen Sie das Konfigurationsverzeichnis in Git ab. Die Signaturen heruntergeladener Archive und Images können Sie gemäß [Releases verifizieren](/en/docs/guide/verify) prüfen.
