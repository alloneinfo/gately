---
title: "Download"
description: "Download the latest Gately binaries or deploy with the Docker image."
---

# Download Gately

## Docker image (recommended)

The image is published for amd64, arm64 and arm/v7:

```bash
docker pull ghcr.io/wangwanjie/gately:latest
```

See [Installation](/en/docs/guide/install) for the full set of deployment options.

## Binaries

Download the archive for your platform, extract it and run `./gately -c /etc/gately`. Supported platforms: Linux (amd64 / arm64 / arm / mips / mipsle / riscv64), macOS, Windows and FreeBSD.

<DownloadList />

::: tip Slow downloads?
GitHub can be slow on some networks. NAS and soft-router users can pull through a proxy first, or use the Docker image instead.
:::

## Build from source

Requires Go 1.26+, Node.js 24+ and pnpm:

```bash
git clone https://github.com/wangwanjie/gately.git
cd gately
make web      # build the frontend
make build    # outputs bin/gately
```

## Verification

Every release ships with a checksum file. Verify the SHA-256 after downloading and before running.

Since v0.5.0, the checksum file and Docker images carry cosign keyless signatures, archives come with build provenance, and an SBOM is published. See [Verifying release artifacts](/en/docs/guide/verify).
