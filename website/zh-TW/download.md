---
title: "下載"
description: "下載 Gately 最新版本的執行檔，或使用 Docker 映像檔部署。"
---

# 下載 Gately

## Docker 映像檔（建議）

映像檔同時提供 amd64、arm64、arm/v7：

```bash
docker pull ghcr.io/wangwanjie/gately:latest
```

完整的部署參數請見 [安裝部署](/zh-TW/docs/guide/install)。

## 執行檔

下載對應平台的壓縮檔，解壓縮後執行 `./gately -c /etc/gately`。支援 Linux（amd64 / arm64 / arm / mips / mipsle / riscv64）、macOS、Windows 與 FreeBSD。

<DownloadList />

::: tip 下載速度慢？
GitHub 在部分網路環境下存取較慢，NAS 與軟路由使用者可以先透過代理拉取，或改用 Docker 映像檔。
:::

## 從原始碼建置

需要 Go 1.26+、Node.js 24+ 與 pnpm：

```bash
git clone https://github.com/wangwanjie/gately.git
cd gately
make web      # 建置前端
make build    # 輸出 bin/gately
```

## 校驗

每個版本都附有校驗碼檔案，下載後請先核對 SHA-256 再執行。

自 v0.5.0 起，校驗碼檔案與 Docker 映像都附有 cosign 無金鑰簽章，壓縮檔附帶建置來源證明，並提供 SBOM。驗證方法見[驗證發佈檔案](/en/docs/guide/verify)（英文）。
