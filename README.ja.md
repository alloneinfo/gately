# Gately

[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | **日本語** | [한국어](README.ko.md) | [Español](README.es.md) | [Français](README.fr.md) | [Deutsch](README.de.md) | [Русский](README.ru.md) | [Português (Brasil)](README.pt-BR.md)

オープンソースで監査可能なネットワークゲートウェイツールセットです。家庭内ネットワーク、NAS、クラウドホスト、各種サーバーで利用できます。[Lucky](https://github.com/gdy666/lucky) にインスパイアされ、同種の機能を完全なオープンソースで提供します。DDNS、STUN 穿孔、ポート転送、リバースプロキシ、証明書、WAF など、よく使う機能を約 22 MB の単一ファイルにまとめ、Lucky の設定もワンクリックでインポートできます。

![Gately 概要](docs/images/overview.webp)

![ダイナミック DNS](docs/images/ddns.webp)

![Docker](docs/images/docker.webp)

## なぜ Gately なのか

Lucky は中国の NAS・家庭内ネットワークユーザーに広く知られた優れたツールで、DDNS、STUN 穿孔、リバースプロキシなどを一つにまとめており、Gately のプロダクトの方向性も示してくれました。Lucky がなければ Gately は生まれませんでした。Lucky はクローズドソースで無償配布されていますが、Gately は完全にオープンソースとし、ユーザーとコミュニティがソースコードと設定を直接確認できるようにしています。ネットワークの境界で動作し、DNS の鍵や証明書の秘密鍵を扱うソフトウェアだからこそ、オープンソースであることで何をしているかを自分の目で確かめられます。Gately の約束:

- ソースコードは Apache-2.0 で公開し、コミュニティによるレビューを歓迎します
- 既定ではプロジェクト側のサーバーに一切接続せず、デバイス情報も送信しません。サードパーティログインにはご自身の OAuth アプリを使用し、中継は経由しません
- 外部へのアドレス（パブリック IP 取得、STUN サーバーなど）はすべて確認・変更・無効化できます
- 設定はプレーンテキストの JSON なので、そのまま git に入れて監査やロールバックができます

## 機能

| モジュール | 機能 |
| --- | --- |
| ダイナミック DNS | Alibaba Cloud、Cloudflare、Tencent Cloud、DNSPod、Huawei Cloud、GoDaddy、Porkbun、dynv6、DuckDNS、カスタムコールバック。URL／ネットワークインターフェース／コマンドによる IP 取得、IPv4/IPv6、Webhook |
| STUN 穿孔 | TCP/UDP、キープアライブ、UPnP/NAT-PMP、内蔵フォワーディング、アドレス変更時の Webhook・スクリプト実行 |
| ポート転送 | TCP/UDP、IPv4/IPv6、ポート範囲、複数ターゲットのフェイルオーバー、接続数制限、トラフィック統計 |
| SSL 証明書 | ACME DNS-01 による自動発行・更新（ワイルドカード、EAB）、アップロード、ファイル読み込み、ディレクトリへのマッピングとスクリプト実行 |
| Tailscale | tailscaled の管理（ユーザー空間ネットワーク、コンテナ対応）またはシステム既存のデーモンへの接続。認証キー／ログインリンク、Headscale、サブネットルート、出口ノード、ノード一覧と SOCKS5 プロキシ、固定ポート、tailscaled の追加引数と環境変数 |
| frp | frpc クライアント（tcp/udp/http/https/stcp/xtcp プロキシ、追加・削除は即時反映）と frps サーバー（ポート許可リスト、サブドメイン、ダッシュボード）。旧形式の ini／新形式の toml 設定をワンクリックでインポート |
| Cloudflare トンネル | cloudflared のトークントンネルと trycloudflare の一時トンネル、複数トンネル、自動再起動、パブリックアドレス表示 |
| Web サービス | ドメインごとのリバースプロキシ／リダイレクト／静的ファイル／固定テキスト、HTTPS、WebSocket、PROXY プロトコルの送受信、Basic 認証、Web 認証、GitHub/Google/QQ/OIDC ログイン、IP 許可・拒否リスト、パスルール、一時共有リンク、認証失敗時の自動ブロック、GeoIP による国別フィルタ、バックエンドの重み／スタンバイ／負荷分散ポリシー、アクティブ／パッシブヘルスチェック、圧縮、レート制限、レスポンスキャッシュ、パス書き換え、1 分単位のアクセス統計と Prometheus メトリクス |
| WAF ファイアウォール | Coraza + OWASP CRS v4、ファイル単位のルール有効化、カスタム SecLang、ブロック／検知のみ |
| Docker 管理 | コンテナの作成と編集、ログとターミナル、Compose、イメージ更新チェックとワンクリックアップグレード（失敗時は自動ロールバック）、ネットワークとボリューム |
| Web ターミナル | SSH（ホストキーは初回接続時に信頼）、Telnet、ローカルシェル（既定は無効）。プロキシ、永続セッション、SFTP ファイル管理、2FA とプロトコル確認、クイックコマンド、画面分割 |
| ネットワーク診断 | Ping、トレースルート、ポート探査、DNS 照会、速度測定、LAN スキャン |
| 通知センター | Webhook、Telegram、Bark、WeCom、DingTalk、Feishu、メール。レベル、ワイルドカード、サイレント・重複排除ルール |
| 設定バックアップ | 暗号化バックアップ（scrypt + AES-GCM）、保持ポリシー、ローカルディレクトリやストレージのマウントポイントへの定期バックアップ |
| コマンドラインと MCP | `gately ctl` で稼働中のインスタンスを操作、`gately mcp` で stdio 経由の MCP サービスを提供（既定は読み取り専用） |
| Wake-on-LAN | マジックパケットのブロードキャスト、オンライン検出 |
| WebDAV | 複数ユーザー、ユーザーごとのディレクトリ、読み取り専用 |
| FTP | 複数ユーザー、ユーザーごとのディレクトリ、読み取り専用、FTPS とパッシブポート範囲に対応 |
| ストレージ管理 | ローカルディレクトリとリモート WebDAV のマウント、オンラインファイル管理 |
| スケジュールタスク | cron 式、コマンドまたは HTTP リクエスト、実行履歴 |
| セキュリティと設定 | 2 段階認証（TOTP）、ログイン失敗のレート制限、セキュアエントリー、内部ネットワークのみのアクセス、管理画面の HTTPS、複数の API トークン（モジュール単位の読み書き権限）、カスタム DNS（DoH 対応）、タイムゾーン、設定のバックアップと復元 |
| 移行 | Lucky の `.lkcf` またはサイト全体の設定 zip をインポート: ポート転送、DDNS、STUN、証明書、リバースプロキシ、WAF、Wake-on-LAN、Web ターミナル、WebDAV、基本設定 |

## クイックスタート

### Docker

イメージは amd64、arm64、arm/v7 に対応しています。

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep -i "initial password"
```

`http://<デバイスのIP>:16680` を開き、`admin` とログに出力された初期パスワードでログインします。その後「設定」でパスワードを変更し、2 段階認証を有効にしてください。完全な Compose の例は [deploy/docker-compose.yml](deploy/docker-compose.yml) を参照してください。

ポート転送、STUN、Web サービスはホストのポートで待ち受けるため、`--network host` を推奨します。Docker 管理が不要な場合は `docker.sock` のマウントを外せます。

### バイナリ

[Releases](https://github.com/wangwanjie/gately/releases) からお使いのプラットフォーム向けのアーカイブ（Linux amd64/arm64/arm/mips/mipsle/riscv64、macOS、Windows、FreeBSD）をダウンロードし、展開して実行します。

```bash
./gately -c /etc/gately
```

オプション: `-c` は設定ディレクトリ（環境変数 `GATELY_CONFIG_DIR` でも指定可能）、`-listen` は管理画面の待ち受けアドレスの上書き、`-version` はバージョン表示です。

### API の利用

管理コンソールの「API トークン」ページでトークンを作成します（複数作成可能で、モジュールごとに読み取り専用または読み書きを付与でき、有効期限も設定できます。平文は作成直後に一度だけ表示されます）。スクリプトから管理 API を一括で呼び出せます。

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks
```

アカウント、2 段階認証、トークン管理、およびバックアップ、復元、再起動、システム設定の API はログインセッションでのみ利用できます。すべての API、必要な権限、データ構造は管理コンソールの「API ドキュメント」ページで確認できます。`make api-docs` で [docs/api](docs/api/ja)（`openapi.json` と `API.md`）にエクスポートすることもでき、公式サイト https://gately.cn の公開時にそのまま参照されます。

旧バージョンからアップグレードすると、従来のグローバル API トークンは自動的に「レガシートークン」になります（値は同じで、全モジュールの権限を持ちます）。API トークンページで権限を絞るか、新しいトークンに切り替えることを推奨します。

### Tailscale、frp、cloudflared

この 3 つのモジュールは公式の外部プログラムを管理します（自動再起動、ログは画面に集約）。Gately 本体には同梱していないため、常に上流の最新版を使えます。プログラムは次の順に探索されます: モジュール設定で指定したパス、`PATH`、設定ディレクトリ配下の `bin/` サブディレクトリ。Docker 環境では、マウントした設定ディレクトリにバイナリを置くだけです（例: `./gately-data/bin/frpc`）。Tailscale には `tailscale` と `tailscaled` の両方が必要です。設定ファイル内のトークンとキーは 0600 のパーミッションで保存され、cloudflared のトークンは環境変数で渡されるため、プロセス一覧には現れません。

### Lucky からの移行

[移行ガイド](https://gately.cn/ja/docs/guide/migration)を参照してください。手順の概要: 「設定 → Lucky からインポート」で Lucky の設定ディレクトリにある `lucky_*.lkcf` ファイルを選び、プレビューしてからインポートします。インポートされたルールは既定で無効になっています。モジュールごとに、Lucky 側で停止し、Gately 側で有効化してください。

## 開発

必要環境: Go 1.26+、Node.js 24+、pnpm。

```bash
make web      # フロントエンドをビルド（web/dist に出力、go:embed で埋め込み）
make build    # bin/gately をコンパイル
make test     # go test -race ./...
make lint     # gofmt、go vet、フロントエンドの型チェック
```

フロントエンド開発時は `cd web && pnpm dev` を実行します。`/api` はローカルの `127.0.0.1:16680` にプロキシされます。

一部のテストは既定でスキップされ、追加の条件が必要です。

| テスト | 有効化方法 |
| --- | --- |
| ACME エンドツーエンド（Pebble） | pebble と pebble-challtestsrv をインストールし、`GATELY_PEBBLE_BIN`、`GATELY_PEBBLE_SRC` を設定（CI では自動実行） |
| 実際のパブリック STUN サーバー | `GATELY_NET_TEST=1` |
| 実際の Lucky 設定によるインポート検証 | `GATELY_LUCKY_SAMPLE` など。各モジュールの `TestImportRealSample` を参照 |

ディレクトリ構成と技術選定は [docs/ROADMAP.md](docs/ROADMAP.md) を参照してください。

公式サイト（[gately.cn](https://gately.cn)）のソースは `website/` にあり、VitePress で構築されています: `cd website && pnpm install && pnpm dev`。`pnpm deploy:site` でビルドしてサーバーに同期します。

## コントリビューションの方針

- Lucky との互換性は、公開されて観察可能な挙動、インターフェース、設定フォーマットに基づきます。Lucky のバイナリやフロントエンドの成果物からコピーしたコードは提出しないでください。
- 新しいモジュールには、設定構造とバリデーション、実行時の起動・停止、REST API、ユニットテストが必要です。Lucky から移行できるモジュールは `luckyimport.Importer` を、概要ページにステータスを表示したいモジュールは `module.Summarizer` を実装してください。
- コードコメントは中国語で記述し、制御文には必ず波括弧を使用します。

## コミュニティ

使い方の質問やフィードバックは Telegram グループへどうぞ：[t.me/gately_chat](https://t.me/gately_chat)。不具合は [GitHub Issues](https://github.com/wangwanjie/gately/issues) に報告してください。

## Star 履歴

[![Star History Chart](https://api.star-history.com/svg?repos=wangwanjie/gately&type=Date)](https://star-history.com/#wangwanjie/gately&Date)

## ライセンス

[Apache License 2.0](LICENSE)
