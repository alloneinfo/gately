---
title: 下载
description: 下载 Gately 最新版本的二进制，或使用 Docker 镜像部署。
---

# 下载 Gately

## Docker 镜像（推荐）

镜像同时提供 amd64、arm64、arm/v7：

```bash
docker pull ghcr.io/wangwanjie/gately:latest
```

完整的部署参数见 [安装部署](/docs/guide/install)。

## 二进制

下载对应平台的压缩包，解压后运行 `./gately -c /etc/gately`。支持 Linux（amd64 / arm64 / arm / mips / mipsle / riscv64）、macOS、Windows 与 FreeBSD。

<DownloadList />

::: tip 下载较慢？
GitHub 在部分网络环境下访问较慢，NAS 与软路由用户可以先通过代理拉取，或使用 Docker 镜像。
:::

## 从源码构建

依赖 Go 1.26+、Node.js 24+ 与 pnpm：

```bash
git clone https://github.com/wangwanjie/gately.git
cd gately
make web      # 构建前端
make build    # 输出 bin/gately
```

## 校验

每个版本附带校验和文件，下载后请核对 SHA-256，再运行。

从 v0.5.0 起，校验和文件与 Docker 镜像都带有 cosign 无密钥签名，压缩包附带构建来源证明，并提供 SBOM。校验方法见[校验发布产物](/docs/guide/verify)。
