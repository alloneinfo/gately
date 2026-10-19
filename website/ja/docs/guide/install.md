# インストールとデプロイ

## Docker

イメージのアドレスは `ghcr.io/wangwanjie/gately` で、amd64、arm64、arm/v7 に対応しています。

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
```

### Docker Compose

```yaml
services:
  gately:
    image: ghcr.io/wangwanjie/gately:latest
    container_name: gately
    restart: unless-stopped
    # ポート転送、STUN 穿孔、Web サービスはいずれもホストのポートを直接待ち受ける必要があります
    network_mode: host
    volumes:
      # 設定ディレクトリ（平文の JSON。git でのバックアップも可能。ssl.json には証明書の秘密鍵が含まれます）
      - ./data:/etc/gately
      # Docker 管理（不要なら削除してください）
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TZ=Asia/Shanghai
```

イメージには `sh`、`bash`、`curl` が含まれているため、一般的な通知スクリプトはそのまま実行できます。

## バイナリ

1. [ダウンロードページ](/ja/download) から使用するプラットフォーム向けのアーカイブを取得します。対応は Linux（amd64 / arm64 / arm / mips / mipsle / riscv64）、macOS、Windows、FreeBSD です。
2. 展開して次のコマンドを実行します。

```bash
./gately -c /etc/gately
```

### systemd

リポジトリには完全なサービスファイル [deploy/systemd/gately.service](https://github.com/wangwanjie/gately/blob/main/deploy/systemd/gately.service) が用意されており、低い番号のポートで待ち受けるために必要なケーパビリティも付与済みです。

```bash
sudo install -m 0755 gately /usr/local/bin/gately
sudo install -m 0644 gately.service /etc/systemd/system/gately.service
sudo systemctl daemon-reload && sudo systemctl enable --now gately
journalctl -u gately | grep -i "initial password"
```

### OpenWrt

procd 用の起動スクリプト [deploy/openwrt/gately.init](https://github.com/wangwanjie/gately/blob/main/deploy/openwrt/gately.init) を使います。

```bash
cp gately /usr/bin/gately && chmod +x /usr/bin/gately
cp gately.init /etc/init.d/gately && chmod +x /etc/init.d/gately
/etc/init.d/gately enable && /etc/init.d/gately start
```

## 初回ログイン

管理画面はデフォルトで `16680` ポートを待ち受けます。ユーザー名は `admin`、初期パスワードは初回起動時にログへ出力されます。自動デプロイでは環境変数 `GATELY_INITIAL_PASSWORD` で初期パスワードを指定でき、その場合はランダムなパスワードは生成されません。

初回ログイン時には初期設定ガイドが表示され、パスワードの変更、2 段階認証または Passkey の有効化、ネットワーク自己診断の実行、Lucky からのインポート、通知の設定を順に案内します。

## アップグレード

Gately はオンライン自動更新を内蔵していません。アップグレードはデプロイ方式に従って行います。

- Docker：新しいイメージを `docker pull` してからコンテナを再作成します（設定ディレクトリをマウントしたままにしておけば問題ありません）。
- バイナリ：実行ファイルを置き換えて、サービスを再起動します。

アップグレード前に設定のバックアップをエクスポートするか、設定ディレクトリごと git に入れておくことをおすすめします。ダウンロードしたアーカイブやイメージは、[リリースの検証](/en/docs/guide/verify) に従って署名を確認できます。
