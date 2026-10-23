---
title: Änderungsprotokoll
description: Release-Ankündigungen und Versionshinweise zu jeder Gately-Version.
---

# Änderungsprotokoll

Unten stehen die Ankündigungen zu wichtigen Releases. Alle Änderungen und Downloads jeder Version finden Sie unter [GitHub Releases](https://github.com/wangwanjie/gately/releases).

## [v0.5.7](/de/news/v0.5.7)

24. Oktober 2026 · Speicher-Einbindungen verbinden sich mit SMB-Freigaben; Designs werden zu 15 Paletten mit hell und dunkel, eigenem Akzent und Ecken; die Verwaltungsoberfläche wird komprimiert übertragen und die Binärdatei ist etwa 11 % kleiner.

## [v0.5.6.4](/de/news/v0.5.6.4)

23. Oktober 2026 · Die automatische Zertifikatsbereitstellung kopiert in mehrere lokale Verzeichnisse mit jeweils eigenem Befehl und unterstützt eigene Dateinamen für Zertifikat und Schlüssel.

## [v0.5.6.3](/de/news/v0.5.6.3)

23. Oktober 2026 · Behebt, dass die Übersicht unter OpenWrt die Schnittstelle br-lan ausblendete und sich die Zeitzone nicht einstellen ließ, sodass die Zeit in UTC lief.

## [v0.5.6.2](/de/news/v0.5.6.2)

23. Oktober 2026 · Neu gestaltete Anmelde- und Fehlerseiten des Webdienstes, Zwei-Faktor-Bestätigung im Pop-up. Behebt, dass das Docker-Modul unter Docker 29 keine Verbindung bekam, die Übersicht vor der Anmeldeseite aufblitzte und aus Lucky importierte Cloudflare-Zugangsdaten sowie umbenannte DDNS-Einträge nicht synchronisiert wurden.

## [v0.5.6.1](/de/news/v0.5.6.1)

22. Oktober 2026 · Behebt, dass vom Docker-Image geschriebene Firewall-Regeln ein älteres nft auf dem Host abstürzen ließen.

## [v0.5.6](/de/news/v0.5.6)

22. Oktober 2026 · Neu: Einbruchschutz mit Protokollregeln wie bei fail2ban, Köderports und Portscan-Erkennung; Sperren werden in nftables übertragen.

## [v0.5.5.2](/de/news/v0.5.5.2)

22. Oktober 2026 · Behebt, dass andere lokale Benutzer auf fnOS und ähnlichen Dateisystemen sensible Dateien im Konfigurationsverzeichnis lesen konnten.

## [v0.5.5.1](/de/news/v0.5.5.1)

21. Oktober 2026 · Laufzeitprotokolle werden auf die Festplatte geschrieben und überstehen Neustarts und Upgrades.

## [v0.5.5](/de/news/v0.5.5)

21. Oktober 2026 · nginx-Konfigurationen importieren, Zertifikate automatisch auf andere Hosts und Container verteilen, WAF mit Unterstützung für große Uploads und WebDAV, abonnierte IP-Listen; der Selbsttest prüft jetzt auch Zertifikatsabdeckung und frp-Konflikte.

## [v0.5.4](/de/news/v0.5.4)

21. Oktober 2026 · Online-Upgrade für Linux-Binärinstallationen, PVE-LXC-Container müssen dafür nicht mehr neu angelegt werden; neue OpenWrt-Pakete (.ipk / .apk).

## [v0.5.3](/de/news/v0.5.3)

21. Oktober 2026 · Native fnOS-App, die sich in das einheitliche fnOS-Gateway einfügt und fnOS-Administratoren mit ihrer fnOS-Sitzung anmeldet; Funktionsmodule lassen sich ein- und ausschalten.

## [v0.5.2](/de/news/v0.5.2)

20. Oktober 2026 · SNI-Routing lässt mehrere Dienste Port 443 teilen, die Web-Dienste können nginx ersetzen, gemeinsame Benutzer erhalten TOTP und Single Sign-on, dazu ein Leistungsvergleich mit haproxy + nginx.

## [v0.5.0](/de/news/v0.5.0)

19. Oktober 2026 · Audit-Protokoll, Konfigurationsverlauf und Zurücksetzen, Passkey, Nur-Lese-Konten, ACME HTTP-01, HTTP/3, PROXY-Protokoll, globale Prometheus-Metriken, Netzwerk-Selbsttest und signierte Releases.
