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

## fnOS ネイティブアプリ {#fnos}

各リリースには fnOS 向けのオフラインパッケージ `gately_<version>_fnos_x86.fpk`（Intel / AMD）と `gately_<version>_fnos_arm.fpk`（ARM）が含まれます。公式のアプリセンターには掲載されていないため、手動でインストールします。

1. [ダウンロードページ](/ja/download#fnos) から、デバイスのアーキテクチャに合った `.fpk` を取得します。
2. fnOS で「アプリセンター → 手動インストール」（左下のボタン）を開き、ファイルを選んでウィザードを完了します。
3. fnOS デスクトップの Gately アイコンをクリックすると、管理画面が開きます。

### fnOS 統一ゲートウェイに対応

デスクトップのアイコンは、fnOS へのアクセスに使っているアドレスの `/app/gately/` で、fnOS 統一ゲートウェイ経由で管理画面を開き、同じアクセス入口を共用します。LAN のアドレスでも fnOS のリモートドメインでもそのまま使え、16680 ポートを開放・転送する必要はありません。

ゲートウェイ経由のアクセスは fnOS が制御し、fnOS 管理者だけが開けます。セキュアエントリーと「LAN 内からのみ許可」は 16680 ポートへの直接アクセスにのみ適用されます。LAN 内では `http://<NAS のアドレス>:16680` も引き続き使えます。

### fnOS 管理者のログイン状態でログイン

「設定 → アクセス制御」で「fnOS のログイン状態で自動ログイン」を有効にすると、fnOS にログイン済みの管理者はデスクトップのアイコンからパスワードや 2 段階認証コードなしでそのまま Gately に入れます。

- 自動ログインの対象は fnOS 管理者だけで、Gately の管理者としてログインします。一般の fnOS ユーザーは引き続きユーザー名とパスワードを入力します。
- 統一ゲートウェイ経由のアクセスにのみ適用されます。16680 ポートへの直接アクセスでは通常のログインが必要です。
- ログアウトした後は自動で再ログインせず、ログイン画面に「fnOS アカウント &lt;ユーザー名&gt; でログイン」ボタンが表示されます。
- この設定は既定で無効で、fnOS パッケージでのみ表示されます。

### 注意事項

- アプリは `root` で動作するため、80/443 の待ち受け、`docker.sock` を通じた Docker の管理、実際の Web ターミナルが利用できます。自分で管理している NAS にのみインストールしてください。
- 初回起動時に `admin` の初期パスワードが `/var/apps/gately/var/gately.log` に書き込まれます：`grep -i "initial password" /var/apps/gately/var/gately.log`。
- 設定はアプリの `var` ディレクトリに保存され、アンインストール時も保持されます。アンインストールウィザードで「すべて削除」を選んだ場合のみ消去されます。
- パッケージには `/usr/local/bin/gately` コマンドも含まれ、`gately ctl` と `gately mcp` を使えます。
- 16680 ポートは Docker イメージと共通です。ネイティブアプリを起動する前に `gately` コンテナを停止してください。

## 初回ログイン

管理画面はデフォルトで `16680` ポートを待ち受けます。ユーザー名は `admin`、初期パスワードは初回起動時にログへ出力されます。自動デプロイでは環境変数 `GATELY_INITIAL_PASSWORD` で初期パスワードを指定でき、その場合はランダムなパスワードは生成されません。

初回ログイン時には初期設定ガイドが表示され、パスワードの変更、2 段階認証または Passkey の有効化、ネットワーク自己診断の実行、Lucky からのインポート、通知の設定を順に案内します。

## アップグレード

### オンラインアップグレード {#online-upgrade}

Linux バイナリ導入（systemd、PVE の LXC コンテナ、OpenWrt など）では、管理画面の「バージョン情報」ページからオンラインでアップグレードできます。「アップデートを確認」をクリックし、新しいバージョンが見つかったら「今すぐアップグレード」をクリックします。PVE で OCI イメージから作成した LXC コンテナも対象で、削除して作り直す必要はありません。Gately は次の順に処理します。

1. このプラットフォーム用の `gately_<バージョン>_linux_<アーキテクチャ>.tar.gz` を GitHub からダウンロードします。
2. 同じリリースの `checksums.txt` と SHA-256 を照合します。
3. 新しいバイナリを `-version` で試験実行し、このマシンで動くことを確認します。
4. 実行ファイルを置き換えて同じ引数で再起動します。旧バージョンは同じディレクトリに `gately.old` として残ります。

すべてダイアログ内で進みます。まずそのバージョンの更新内容が表示され、開始後は現在のステップ、ダウンロードの進捗と速度が表示されます。プログラムを置き換える前（バージョン取得・ダウンロード・検証の段階）なら、いつでも「アップグレードを取消」をクリックでき、現在のプログラムは変更されません。

GitHub への接続が不安定な場合は、「バージョン情報」ページまたは「設定 → ネットワークと実行」で**更新用プロキシ**を設定できます。`http://`、`https://`、`socks5://` に対応し、更新確認とダウンロードの両方がこのプロキシを経由します。空欄の場合は直接接続し、`HTTPS_PROXY` 環境変数があればそれに従います。

インターネットに接続できない環境では、[ダウンロードページ](/ja/download) から tar.gz を入手し、「バージョン情報」ページの「アップグレードパッケージをアップロード」でアップグレードできます。アップロードはバージョンの新旧を問わないため、旧バージョンへのロールバックにも使えます。

オンラインアップグレードは管理者が操作したときだけ実行され、Gately が自動でダウンロードやインストールを行うことはありません。新しいバージョンが起動しない場合は、残しておいた旧バージョンで手動でロールバックできます（systemd の例）。

```bash
mv /usr/local/bin/gately.old /usr/local/bin/gately
systemctl restart gately
```

`setcap` でバイナリに低番号ポートの権限を付与している場合は、アップグレード後にもう一度 `setcap` を実行してください。

### その他のデプロイ方式

- Docker：新しいイメージを `docker pull` してからコンテナを再作成します（設定ディレクトリをマウントしたままにしておけば問題ありません）。
- fnOS：アプリセンターで新しい `.fpk` を手動インストールします。設定は保持されます。
- Windows、macOS、FreeBSD：実行ファイルを置き換えて、サービスを再起動します。

アップグレード前に設定のバックアップをエクスポートするか、設定ディレクトリごと git に入れておくことをおすすめします。ダウンロードしたアーカイブやイメージは、[リリースの検証](/en/docs/guide/verify) に従って署名を確認できます。
