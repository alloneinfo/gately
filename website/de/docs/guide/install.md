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

## Native fnOS-App {#fnos}

Jedes Release enthält Offline-Pakete für fnOS: `gately_<version>_fnos_x86.fpk` (Intel / AMD) und `gately_<version>_fnos_arm.fpk` (ARM). Sie stehen nicht im offiziellen App Center und werden daher manuell installiert:

1. Laden Sie die `.fpk` für die Architektur Ihres Geräts von der [Download-Seite](/de/download#fnos) herunter;
2. Öffnen Sie in fnOS „App Center → Manuelle Installation“ (Schaltfläche unten links), wählen Sie die Datei aus und schließen Sie den Assistenten ab;
3. Klicken Sie auf das Gately-Symbol auf dem fnOS-Desktop, um die Verwaltungsoberfläche zu öffnen.

### Funktioniert mit dem einheitlichen fnOS-Gateway

Das Desktop-Symbol öffnet die Verwaltungsoberfläche über das einheitliche fnOS-Gateway unter `/app/gately/` auf der Adresse, die Sie für fnOS verwenden – über denselben Zugang. Sowohl die LAN-Adresse als auch Ihre Remote-Domain von fnOS funktionieren, ohne Port 16680 freizugeben oder weiterzuleiten.

Den Zugriff über das Gateway kontrolliert fnOS; nur fnOS-Administratoren können es öffnen. „Sicherer Einstieg“ und „Nur LAN-Zugriff“ gelten nur für den direkten Zugriff auf Port 16680. Im LAN funktioniert `http://<NAS-Adresse>:16680` weiterhin.

### Anmeldung mit der fnOS-Administratorsitzung

Aktivieren Sie „Mit fnOS anmelden“ unter „Einstellungen → Zugriffskontrolle“, dann gelangen bei fnOS angemeldete Administratoren über das Desktop-Symbol direkt in Gately, ohne Passwort und Bestätigungscode:

- Angemeldet werden nur fnOS-Administratoren, und zwar als Gately-Administrator; normale fnOS-Benutzer geben weiterhin Benutzername und Passwort ein;
- Es gilt nur für den Zugriff über das einheitliche Gateway; der direkte Zugriff auf Port 16680 erfordert weiterhin eine normale Anmeldung;
- Nach dem Abmelden meldet Gately Sie nicht stillschweigend wieder an; stattdessen zeigt die Anmeldeseite eine Schaltfläche „Als fnOS-Benutzer &lt;Name&gt; anmelden“;
- Die Einstellung ist standardmäßig aus und wird nur im fnOS-Paket angezeigt.

### Hinweise

- Die App läuft als `root`, damit sie auf 80/443 lauschen, Docker über `docker.sock` verwalten und ein echtes Web-Terminal öffnen kann. Installieren Sie sie nur auf einem NAS, das Sie selbst kontrollieren.
- Beim ersten Start wird das initiale `admin`-Passwort in `/var/apps/gately/var/gately.log` geschrieben: `grep -i "initial password" /var/apps/gately/var/gately.log`.
- Die Konfiguration liegt im `var`-Verzeichnis der App und bleibt bei der Deinstallation erhalten, sofern Sie im Deinstallationsassistenten nicht „Alles löschen“ wählen.
- Das Paket installiert außerdem einen Befehl `/usr/local/bin/gately` für `gately ctl` und `gately mcp`.
- Port 16680 wird mit dem Docker-Image geteilt. Stoppen Sie den `gately`-Container, bevor Sie die native App starten.

## Erste Anmeldung

Die Verwaltungsoberfläche lauscht standardmäßig auf Port `16680`. Der Benutzername lautet `admin`, das Initialpasswort wird beim ersten Start ins Log geschrieben. Bei automatisierten Bereitstellungen können Sie das Initialpasswort über die Umgebungsvariable `GATELY_INITIAL_PASSWORD` vorgeben; dann wird kein Zufallspasswort erzeugt.

Bei der ersten Anmeldung erscheint ein Einrichtungsassistent, der nacheinander dazu auffordert, das Passwort zu ändern, Zwei-Faktor-Authentifizierung oder einen Passkey zu aktivieren, den Netzwerk-Selbsttest auszuführen, aus Lucky zu importieren und Benachrichtigungen einzurichten.

## Upgrade

### Online-Upgrade {#online-upgrade}

Linux-Binärinstallationen (systemd, PVE-LXC-Container, OpenWrt usw.) lassen sich auf der Seite „Über“ der Verwaltungsoberfläche online aktualisieren: Klicken Sie auf „Nach Updates suchen“ und bei einer neuen Version auf „Jetzt aktualisieren“. Das gilt auch für LXC-Container, die in PVE aus dem OCI-Image erstellt wurden; Löschen und Neuanlegen ist nicht nötig. Gately geht so vor:

1. `gately_<Version>_linux_<Architektur>.tar.gz` für diese Plattform von GitHub herunterladen;
2. die SHA-256 mit der `checksums.txt` desselben Releases abgleichen;
3. die neue Binärdatei mit `-version` testweise starten, um sicherzustellen, dass sie auf diesem Rechner läuft;
4. die ausführbare Datei ersetzen und mit denselben Argumenten neu starten; die vorherige Version bleibt als `gately.old` im selben Verzeichnis.

Alles läuft in einem Dialog ab: Er zeigt zuerst die Versionshinweise und nach dem Start den aktuellen Schritt sowie Fortschritt und Geschwindigkeit des Downloads. Bis die Binärdatei ersetzt wird (Abruf, Download und Prüfung), können Sie jederzeit auf „Upgrade abbrechen“ klicken; die aktuelle Binärdatei bleibt dabei unverändert.

Ist GitHub schlecht erreichbar, legen Sie auf der Seite „Über“ oder unter Einstellungen → Netzwerk & Laufzeit einen **Update-Proxy** fest. Unterstützt werden `http://`, `https://` und `socks5://`; Update-Prüfung und Download laufen beide darüber. Ist das Feld leer, verbindet Gately direkt und beachtet weiterhin die Umgebungsvariable `HTTPS_PROXY`.

Ohne Internetzugang laden Sie das tar.gz von der [Download-Seite](/de/download) und klicken auf der Seite „Über“ auf „Upgrade-Paket hochladen“. Jede Version wird akzeptiert, sodass Sie auch auf eine ältere zurückgehen können.

Online-Upgrades laufen nur, wenn ein Administrator sie startet; Gately lädt oder installiert nie von selbst eine Version. Startet die neue Version nicht, gehen Sie mit der behaltenen Kopie manuell zurück (Beispiel mit systemd):

```bash
mv /usr/local/bin/gately.old /usr/local/bin/gately
systemctl restart gately
```

Haben Sie der Binärdatei mit `setcap` Zugriff auf niedrige Ports gegeben, führen Sie `setcap` nach dem Upgrade erneut aus.

### Andere Bereitstellungen

- Docker: das neue Image mit `docker pull` laden und den Container neu erstellen (das Konfigurationsverzeichnis bleibt eingebunden);
- fnOS: die neuere `.fpk` manuell über das App Center installieren; Ihre Konfiguration bleibt erhalten;
- Windows, macOS, FreeBSD: die ausführbare Datei ersetzen und den Dienst neu starten.

Exportieren Sie vor dem Upgrade eine Konfigurationssicherung oder legen Sie das Konfigurationsverzeichnis in Git ab. Die Signaturen heruntergeladener Archive und Images können Sie gemäß [Releases verifizieren](/en/docs/guide/verify) prüfen.
