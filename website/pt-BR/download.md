---
title: "Download"
description: "Baixe os binários mais recentes do Gately ou implante com a imagem Docker."
---

# Baixar o Gately

## Imagem Docker (recomendada)

A imagem é publicada para amd64, arm64 e arm/v7:

```bash
docker pull ghcr.io/wangwanjie/gately:latest
```

Veja todas as opções de implantação em [Instalação](/pt-BR/docs/guide/install).

## Binários

Baixe o pacote da sua plataforma, extraia e execute `./gately -c /etc/gately`. Plataformas compatíveis: Linux (amd64 / arm64 / arm / mips / mipsle / riscv64), macOS, Windows e FreeBSD.

<DownloadList />

::: tip Download lento?
O GitHub pode ficar lento em algumas redes. Usuários de NAS e soft routers podem baixar antes por meio de um proxy ou usar a imagem Docker.
:::

## Compilar a partir do código-fonte

Requer Go 1.26+, Node.js 24+ e pnpm:

```bash
git clone https://github.com/wangwanjie/gately.git
cd gately
make web      # compila o frontend
make build    # gera bin/gately
```

## Verificação

Cada versão acompanha um arquivo de checksums. Confira o SHA-256 depois de baixar e antes de executar.
