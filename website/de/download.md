---
title: "Download"
description: "Laden Sie die neuesten Gately-Binärdateien und fnOS-Pakete herunter oder setzen Sie das Docker-Image ein."
---

# Gately herunterladen

## Docker-Image (empfohlen)

Das Image ist für amd64, arm64 und arm/v7 verfügbar:

```bash
docker pull ghcr.io/wangwanjie/gately:latest
```

Alle Bereitstellungsoptionen finden Sie unter [Installation](/de/docs/guide/install).

## Binärdateien

Laden Sie das Archiv für Ihre Plattform herunter, entpacken Sie es und starten Sie `./gately -c /etc/gately`. Unterstützte Plattformen: Linux (amd64 / arm64 / arm / mips / mipsle / riscv64), macOS, Windows und FreeBSD.

<DownloadList />

::: tip Downloads langsam?
GitHub ist in manchen Netzwerken langsam. Nutzer von NAS und Soft-Routern können zuerst über einen Proxy laden oder stattdessen das Docker-Image verwenden.
:::

## Native fnOS-App {#fnos}

fnOS-Nutzer können direkt die native App installieren: Laden Sie die `.fpk` aus der fnOS-Gruppe in der Liste oben herunter (`x86` für Intel / AMD, `arm` für ARM-Geräte) und wählen Sie sie in fnOS unter „App Center → Manuelle Installation“ aus.

- Funktioniert mit dem einheitlichen fnOS-Gateway: Start über das Desktop-Symbol, auch über Ihre Remote-Domain von fnOS, ohne zusätzlich freigegebenen Port;
- Meldet fnOS-Administratoren mit ihrer fnOS-Sitzung an: Nach dem Aktivieren überspringen bei fnOS angemeldete Administratoren das Gately-Passwort und den Bestätigungscode.

Siehe [Installation · Native fnOS-App](/de/docs/guide/install#fnos).

## Aus dem Quellcode bauen

Voraussetzung sind Go 1.26+, Node.js 24+ und pnpm:

```bash
git clone https://github.com/wangwanjie/gately.git
cd gately
make web      # Frontend bauen
make build    # erzeugt bin/gately
```

## Verifizierung

Jedes Release enthält eine Prüfsummendatei. Prüfen Sie nach dem Download und vor dem Start die SHA-256-Summe.

Seit v0.5.0 tragen die Prüfsummendatei und die Docker-Images eine schlüssellose cosign-Signatur, die Archive einen Herkunftsnachweis, außerdem wird eine SBOM veröffentlicht. Siehe [Releases verifizieren](/en/docs/guide/verify) (Englisch).
