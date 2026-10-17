---
title: "Télécharger"
description: "Téléchargez les derniers binaires de Gately ou déployez-le avec l'image Docker."
---

# Télécharger Gately

## Image Docker (recommandée)

L'image est publiée pour amd64, arm64 et arm/v7 :

```bash
docker pull ghcr.io/wangwanjie/gately:latest
```

Consultez [Installation](/fr/docs/guide/install) pour l'ensemble des options de déploiement.

## Binaires

Téléchargez l'archive de votre plateforme, extrayez-la et lancez `./gately -c /etc/gately`. Plateformes prises en charge : Linux (amd64 / arm64 / arm / mips / mipsle / riscv64), macOS, Windows et FreeBSD.

<DownloadList />

::: tip Téléchargement lent ?
GitHub peut être lent sur certains réseaux. Les utilisateurs de NAS et de routeurs logiciels peuvent d'abord passer par un proxy, ou utiliser l'image Docker.
:::

## Compiler depuis les sources

Nécessite Go 1.26+, Node.js 24+ et pnpm :

```bash
git clone https://github.com/wangwanjie/gately.git
cd gately
make web      # compile le frontend
make build    # produit bin/gately
```

## Vérification

Chaque version est accompagnée d'un fichier de sommes de contrôle. Vérifiez le SHA-256 après le téléchargement et avant l'exécution.
