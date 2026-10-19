# Portweiterleitung

Leiten Sie einen Port, auf dem Gately lauscht, an ein anderes Gerät oder einen anderen Dienst weiter.

## Funktionen

- TCP und UDP;
- IPv4 und IPv6, auch mit Weiterleitung zwischen v4 und v6;
- Portbereiche und mehrere Ports;
- Mehrere Ziele mit Failover;
- Verbindungslimits;
- Traffic-Statistiken und Laufzeit-Logs;
- Senden des PROXY-Protokolls (v1 / v2) an das Ziel, damit nachgelagerte Dienste die echte Client-Adresse erhalten.

## PROXY-Protokoll

Die Portweiterleitung leitet TCP nur weiter, daher sieht der nachgelagerte Dienst standardmäßig nur die Adresse von Gately selbst. Setzen Sie in der Regel „PROXY-Protokoll“ auf v1 oder v2, sendet Gately nach dem Verbindungsaufbau zum Ziel zuerst einen PROXY-Header. Der nachgelagerte Dienst erhält so die echte Client-Adresse für Logs, Ratenbegrenzung und Zugriffskontrolle.

Das Ziel muss die passende Empfangsoption aktiviert haben, sonst wird die Verbindung abgelehnt, zum Beispiel:

- haproxy: `bind 192.168.1.8:4434 accept-proxy`;
- nginx: `listen 4435 ssl proxy_protocol;`;
- Gately-Web-Dienst: im Listener „PROXY-Protokoll akzeptieren“ aktivieren und die Quelle zu den vertrauenswürdigen PROXY-Quellen hinzufügen.

Die Option wirkt nur auf TCP; UDP-Weiterleitungen sind nicht betroffen.

## Typische Szenarien

- Dienstports eines NAS über einen einzigen Einstiegspunkt bereitstellen;
- Datenverkehr zwischen IPv6 und IPv4 überbrücken;
- Für einen Dienst ein primäres und ein Standby-Backend einrichten.

## Hinweis zur Bereitstellung

Die Portweiterleitung lauscht direkt auf Ports des Hosts. Bei Docker-Bereitstellungen empfehlen wir `--network host`.

## Migration von Lucky

Regeln lassen sich aus `lucky_portforward.lkcf` importieren. Optionen wie TLS-/Stream-Verschlüsselung und Ratenbegrenzung werden noch nicht unterstützt, beim Import erscheint dazu ein Hinweis.
