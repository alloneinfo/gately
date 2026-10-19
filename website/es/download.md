---
title: "Descargas"
description: "Descarga los últimos binarios de Gately o despliégalo con la imagen de Docker."
---

# Descargar Gately

## Imagen de Docker (recomendada)

La imagen se publica para amd64, arm64 y arm/v7:

```bash
docker pull ghcr.io/wangwanjie/gately:latest
```

Consulta [Instalación](/es/docs/guide/install) para ver todas las opciones de despliegue.

## Binarios

Descarga el archivo para tu plataforma, descomprímelo y ejecuta `./gately -c /etc/gately`. Plataformas compatibles: Linux (amd64 / arm64 / arm / mips / mipsle / riscv64), macOS, Windows y FreeBSD.

<DownloadList />

::: tip ¿Descargas lentas?
GitHub puede ser lento en algunas redes. Los usuarios de NAS y routers software pueden descargar primero a través de un proxy o usar la imagen de Docker.
:::

## Compilar desde el código fuente

Requiere Go 1.26+, Node.js 24+ y pnpm:

```bash
git clone https://github.com/wangwanjie/gately.git
cd gately
make web      # compila el frontend
make build    # genera bin/gately
```

## Verificación

Cada versión incluye un archivo de sumas de verificación. Comprueba el SHA-256 tras la descarga y antes de ejecutar.

Desde v0.5.0, el archivo de sumas y las imágenes Docker llevan firmas sin clave de cosign, los archivos incluyen atestación de procedencia y se publica un SBOM. Consulta [Verificar versiones](/en/docs/guide/verify) (en inglés).
