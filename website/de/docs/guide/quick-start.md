# Schnellstart

Bereitstellen, anmelden und die erste Regel anlegen in fünf Minuten.

## 1. Starten

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep -i "initial password"
```

Wenn Sie die Docker-Verwaltung nicht brauchen, lassen Sie die Einbindung von `docker.sock` weg. Portweiterleitung, STUN und Web-Dienste lauschen auf Ports des Hosts, daher empfehlen wir `--network host`.

## 2. Anmelden

Öffnen Sie `http://<geräte-ip>:16680` im Browser. Der Benutzername lautet `admin`, das Passwort ist das im Log ausgegebene Initialpasswort (es erscheint nur einmal, beim ersten Start).

## 3. Grundlegende Härtung

Gehen Sie direkt nach der Anmeldung zu „Einstellungen“ und:

1. ändern Sie das Administratorpasswort;
2. aktivieren Sie die Zwei-Faktor-Authentifizierung (TOTP);
3. richten Sie nach Bedarf einen sicheren Einstiegspfad, den Zugriff nur aus dem LAN oder HTTPS für die Verwaltungsoberfläche ein.

Weitere Empfehlungen finden Sie im [Härtungsleitfaden](/de/docs/guide/hardening).

## 4. Erste Regel anlegen

Wählen Sie das Modul, das zu Ihrem Bedarf passt:

| Bedarf | Modul |
| --- | --- |
| Ihre öffentliche Heim-IP ändert sich ständig und Sie möchten einen festen Domainnamen | [Dynamisches DNS](/de/docs/modules/ddns) |
| Ihr Provider vergibt keinen festen öffentlichen Port und Sie möchten Dienste zu Hause von außen erreichen | [STUN-Traversal](/de/docs/modules/stun) |
| Einen Port an ein anderes Gerät im LAN weiterleiten | [Portweiterleitung](/de/docs/modules/portforward) |
| Mehrere LAN-Dienste über eine Domain mit HTTPS erreichen | [SSL-Zertifikate](/de/docs/modules/ssl) + [Web-Dienste](/de/docs/modules/web) |
| Standorte miteinander vernetzen | [Tailscale / frp](/de/docs/modules/tunnels) |

## Migration von Lucky

Wenn Sie bereits Lucky nutzen, müssen Sie nichts neu konfigurieren. Importieren Sie unter „Einstellungen → Sicherung & Migration“ eine `.lkcf`-Datei oder ein Zip der gesamten Site-Konfiguration. Siehe die [Migrationsanleitung](/de/docs/guide/migration).
