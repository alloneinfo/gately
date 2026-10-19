# Dynamisches DNS (DDNS)

Die öffentliche IP eines Heim-Breitbandanschlusses ändert sich mit der Zeit. DDNS aktualisiert Ihre DNS-Einträge nach einer IP-Änderung automatisch, sodass Ihre Domain stets auf Ihre aktuelle Adresse zeigt.

## Unterstützte Anbieter

Alibaba Cloud, Cloudflare, Tencent Cloud, DNSPod, Huawei Cloud, GoDaddy, Porkbun, dynv6, DuckDNS, Amazon Route 53, OVHcloud, Gandi, NS1, Bunny DNS, Namecheap, DynDNS2 sowie ein eigener Callback (für Anbieter, die nicht eingebaut sind, oder für selbst gehostete Endpunkte).

| Anbieter | Zugangsdaten | Für DNS-Validierung von Zertifikaten nutzbar |
| --- | --- | --- |
| Amazon Route 53 | Access Key ID / Secret Access Key, optional Hosted-Zone-ID | Ja |
| OVHcloud | Application Key / Secret und Consumer Key, Endpunkt ovh-eu / ovh-ca / ovh-us | Ja |
| Gandi | Personal Access Token | Ja |
| NS1 | API Key | Ja |
| Bunny DNS | Account API Key | Ja |
| Namecheap | Dynamic-DNS-Passwort der Domain (nur A-Einträge) | Nein |
| DynDNS2 | Benutzername, Passwort und Update-URL; für No-IP, Dynu und andere Dienste, die das dyndns2-Protokoll unterstützen | Nein |

## So wird die IP ermittelt

- **URL**: Es wird eine Adresse zur Abfrage der öffentlichen IP angefragt (ersetzen Sie sie durch einen beliebigen Dienst oder hosten Sie selbst einen);
- **Netzwerkschnittstelle**: Die Adresse einer bestimmten Netzwerkschnittstelle wird direkt gelesen, geeignet für Geräte mit öffentlicher IP;
- **Befehl**: Ein Befehl wird ausgeführt und seine Ausgabe verwendet, geeignet für ungewöhnliche Netzwerkaufbauten.

IPv4 und IPv6 werden beide unterstützt.

## Benachrichtigungen

Bei einer IP-Änderung oder einer fehlgeschlagenen Synchronisierung kann ein Webhook aufgerufen werden.

## Empfehlungen

- Legen Sie beim DNS-Anbieter einen Schlüssel für ein Unterkonto an, der auf das Bearbeiten von DNS-Einträgen beschränkt ist. Vermeiden Sie den Schlüssel des Hauptkontos;
- Klicken Sie nach dem Anlegen einer Aufgabe auf „Sync erzwingen“, um zu prüfen, ob der Eintrag aktualisiert wurde;
- Die Adresse zur Abfrage der öffentlichen IP hängt von keinem vom Projekt betriebenen Dienst ab und lässt sich jederzeit in einer Aufgabe ändern.

## Migration von Lucky

DDNS-Aufgaben lassen sich aus `lucky_ddns.lkcf` importieren, mit 11 Anbietern. Die offiziellen IP-Abfrageadressen von Lucky werden beim Import entfernt. Details finden Sie in der [Migrationsanleitung](/de/docs/guide/migration).
