---
title: "ダウンロード"
description: "Gately の最新バージョンのバイナリをダウンロードするか、Docker イメージでデプロイします。"
---

# Gately のダウンロード

## Docker イメージ（推奨）

イメージは amd64、arm64、arm/v7 の 3 アーキテクチャを提供しています。

```bash
docker pull ghcr.io/wangwanjie/gately:latest
```

デプロイ時のパラメーターの詳細は [インストールとデプロイ](/ja/docs/guide/install) を参照してください。

## バイナリ

使用するプラットフォーム向けのアーカイブをダウンロードして展開し、`./gately -c /etc/gately` を実行します。Linux（amd64 / arm64 / arm / mips / mipsle / riscv64）、macOS、Windows、FreeBSD に対応しています。

<DownloadList />

::: tip ダウンロードが遅い場合
ネットワーク環境によっては GitHub へのアクセスが遅くなることがあります。NAS やソフトルーターのユーザーは、先にプロキシ経由で取得するか、Docker イメージを利用してください。
:::

## ソースからビルド

Go 1.26+、Node.js 24+、pnpm が必要です。

```bash
git clone https://github.com/wangwanjie/gately.git
cd gately
make web      # フロントエンドをビルド
make build    # bin/gately を出力
```

## 検証

各バージョンにはチェックサムファイルが付属しています。ダウンロード後に SHA-256 を照合してから実行してください。
