# Portweiterleitung

Leiten Sie einen Port, auf dem Gately lauscht, an ein anderes Gerät oder einen anderen Dienst weiter.

## Funktionen

- TCP und UDP;
- IPv4 und IPv6, auch mit Weiterleitung zwischen v4 und v6;
- Portbereiche und mehrere Ports;
- Mehrere Ziele mit Failover;
- Verbindungslimits;
- Traffic-Statistiken und Laufzeit-Logs.

## Typische Szenarien

- Dienstports eines NAS über einen einzigen Einstiegspunkt bereitstellen;
- Datenverkehr zwischen IPv6 und IPv4 überbrücken;
- Für einen Dienst ein primäres und ein Standby-Backend einrichten.

## Hinweis zur Bereitstellung

Die Portweiterleitung lauscht direkt auf Ports des Hosts. Bei Docker-Bereitstellungen empfehlen wir `--network host`.

## Migration von Lucky

Regeln lassen sich aus `lucky_portforward.lkcf` importieren. Optionen wie TLS-/Stream-Verschlüsselung und Ratenbegrenzung werden noch nicht unterstützt, beim Import erscheint dazu ein Hinweis.
