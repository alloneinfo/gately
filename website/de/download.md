---
title: "Download"
description: "Laden Sie die neuesten Gately-Binärdateien herunter oder setzen Sie das Docker-Image ein."
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
