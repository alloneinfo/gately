# Mitwirken

Gately ist ein von der Community getragenes Open-Source-Projekt. Issues und Pull Requests sind willkommen. Repository: <https://github.com/wangwanjie/gately>

## Lokale Entwicklung

Voraussetzungen: Go 1.26+, Node.js 24+, pnpm.

```bash
make web      # Frontend bauen (Ausgabe in web/dist, per go:embed eingebettet)
make build    # bin/gately kompilieren
make test     # go test -race ./...
make lint     # gofmt, go vet, Typprüfung des Frontends
```

Starten Sie für die Frontend-Entwicklung `cd web && pnpm dev`; `/api` wird an das lokale `127.0.0.1:16680` weitergeleitet.

## Konventionen

- Die Kompatibilität mit Lucky beruht auf öffentlich beobachtbarem Verhalten, Schnittstellen und Konfigurationsformaten. Reichen Sie keinen Code ein, der aus Lucky-Binärdateien oder Frontend-Artefakten kopiert wurde.
- Ein neues Modul braucht: eine Konfigurationsstruktur mit Validierung, Start/Stopp zur Laufzeit, REST-Endpunkte und Unit-Tests. Module, die sich von Lucky migrieren lassen, implementieren `luckyimport.Importer`, und Module, die auf der Übersichtsseite einen Status anzeigen sollen, implementieren `module.Summarizer`.
- Code-Kommentare werden auf Chinesisch geschrieben; Kontrollanweisungen müssen geschweifte Klammern verwenden.

## Zusätzliche Tests

Einige Tests werden standardmäßig übersprungen und brauchen zusätzliche Einrichtung:

| Test | So aktivieren |
| --- | --- |
| ACME-End-to-End (Pebble) | pebble und pebble-challtestsrv installieren und `GATELY_PEBBLE_BIN` sowie `GATELY_PEBBLE_SRC` setzen |
| Echte öffentliche STUN-Server | `GATELY_NET_TEST=1` |
| Importprüfung mit einer echten Lucky-Konfiguration | `GATELY_LUCKY_SAMPLE` und weitere; siehe `TestImportRealSample` des jeweiligen Moduls |

## Lizenz

[Apache License 2.0](https://github.com/wangwanjie/gately/blob/main/LICENSE)
