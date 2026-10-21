# Web ターミナル API の仕様

すべてのエンドポイントは `/api/webterminal/` 配下にあり、ログインが必要です。エラーは常に `{"error": "..."}` で返されます。

## 1. 設定の構造（`conf` 内の webterminal ファイル）

```jsonc
{
  "settings": {
    "allowLocalShell": false,
    "maxSessions": 20,
    "idleTimeout": 0,          // 分。この時間入出力がないセッションは自動で終了。0 は無制限
    "bufferSize": 256,         // KB。セッションごとに保持し、再接続時に再生する出力バッファ
    "heartbeatInterval": 30,   // 秒。WebSocket のハートビート（サーバーの ping / フロントエンドの ping）
    "sessionKeepAlive": 0      // 分。ブラウザ切断後にセッションを保持する時間。0 は切断と同時に終了
  },
  "connections": [Connection],
  "shortcuts": [{ "name": "Show disk usage", "command": "df -h", "sendEnter": true }],
  "splitLayout": null,         // フロントエンドがカスタマイズした分割レイアウト。バックエンドはそのまま保存して返す（json.RawMessage）
  "security": {
    "agreementAccepted": false,
    "twoFAEnable": false,
    "twoFAKey": ""             // TOTP シークレット。API の出力ではマスクされる
  },
  "credentials": [Credential]  // SSH 認証情報のプリセット
}
```

Connection（`type` は `ssh` / `telnet` / `local`）：

```jsonc
{
  "key": "", "name": "", "type": "ssh", "remark": "", "enable": true,
  "quickAccessDirs": ["/etc", "/var/log"],      // SFTP のクイックアクセスディレクトリ
  "ssh": { "host", "port", "username", "authType": "password|key|credential",
           "credentialKey": "",  // authType=credential のときの認証情報プリセットの key。この場合 username は空でもよく、プリセットの既定値が使われる
           "password", "privateKey",
           "keyFile": "",        // サーバー自身にある秘密鍵ファイルのパス。authType=key で privateKey が空のときに使われる
           "passphrase", "hostKey", "keepAlive", "workDir": "", "timeout": 15,
           "proxy": Proxy },
  "telnet": { "host", "port": 23, "username", "password", "timeout": 15, "proxy": Proxy },
  "local": { "shell", "workDir", "env": {"K":"V"} }
}
Proxy = { "enabled": false, "type": "socks5|http", "host", "port", "username", "password" }
```

Credential（認証情報のプリセット。複数の SSH 接続から参照できます）：

```jsonc
{ "key": "", "name": "", "authType": "password|key", "username": "", // 任意。接続に username がないときに使われる
  "password", "privateKey", "keyFile", "passphrase", "remark": "", "enable": true }
```

パスワード、秘密鍵、秘密鍵のパスフレーズ、プロキシのパスワードは、出力時に `crud.Mask` でマスクされ、更新時に `crud.KeepSecret` で保持されます。

## 2. 接続の管理

| メソッド | パス | 説明 |
|---|---|---|
| GET/POST | `/connections` | 一覧 / 作成 |
| PUT/DELETE | `/connections/{key}` | 更新 / 削除 |
| PUT | `/connections-order` | 並べ替え `{"keys":[...]}` |
| POST | `/connections/{key}/test` | SSH / Telnet の接続テスト |
| POST | `/connections/test` | 未保存の接続をテストします（ボディは Connection。`key` フィールドで元の接続を指定すると、保存済みのシークレットがマージされます） |
| DELETE | `/connections/{key}/hostkey` | ホストのフィンガープリントをリセット |
| PUT | `/connections/{key}/quickaccess` | ボディ `{"dirs":["/a","/b"]}` |
| GET/POST | `/credentials` | 認証情報プリセットの一覧 / 作成 |
| PUT/DELETE | `/credentials/{key}` | 更新 / 削除（参照されている間は、接続名を挙げて 409 を返します） |
| PUT | `/credentials-order` | プリセットの並べ替え `{"keys":[...]}` |
| GET | `/shells` | このマシンで使えるシェルの一覧 `["/bin/bash", ...]`（/etc/shells を読み、実在するものに絞り込みます） |

補足（実装が正です）：
- `POST /connections/test` と `POST /connections/{key}/test` は 200 で `{ok, error?, serverVersion?, newHostKey?, latencyMs}` を返します。`local` タイプは 400 を返します。Telnet は TCP の到達性（プロキシ経由を含む）のみを確認します。
- `PUT /connections/{key}/quickaccess` は `{ok, dirs}`（空白と重複を除去済み）を返します。`quickAccessDirs` を含まずに接続を編集した場合は、古い値が保持されます。
- `ssh.timeout` / `telnet.timeout` は 1〜120 秒で、既定値は 15 です。`ssh.workDir` はログイン後に `cd` で適用されます。
- `authType=credential` の接続は、保存時にインラインのパスワード、秘密鍵、鍵ファイル、パスフレーズが消去され、接続時にプリセットから認証情報を解決します。プリセットを更新すると、それを参照しているプール内の SSH 接続は破棄されます。
- Telnet のログイン自動入力は、`login:` / `username:` / `password:` などのプロンプトを検出したときだけ、それぞれ 1 回ずつ試します。

## 3. ターミナルセッション（切断後に再接続できる永続セッション）

- `GET /connections/{key}/ws?cols=&rows=`：新しいセッションを作成し、WebSocket にアップグレードします。
- `GET /sessions/{id}/ws?cols=&rows=`：既存のセッションに接続し、まずバッファを再生してから、リアルタイムで転送します。複数のクライアントが同時に同じセッションに接続できます（入力は共有され、出力はブロードキャストされます）。

WebSocket プロトコル：
- サーバー → ブラウザ：**バイナリフレーム**でターミナルの出力を、**テキストフレーム**で JSON を送ります。
  `{"t":"s","d":"<sessionId>"}` セッション確立（作成時と接続時のどちらも 1 回送信）；
  `{"t":"n","d":"notice"}` 通知；`{"t":"x","d":"reason"}` セッション終了；`{"t":"p"}` ハートビートの応答。
- ブラウザ → サーバー：テキストの JSON：`{"t":"i","d":"input"}`、`{"t":"r","c":cols,"r":rows}`、`{"t":"p"}` ハートビート。
- ブラウザの WebSocket が切断されたとき：`sessionKeepAlive > 0` ならセッションを保持し（切り離し状態）、誰も接続しないままタイムアウトすると閉じます。0 ならすぐに閉じます。
- セッションのプロセスが自ら終了した場合は、`x` メッセージを送り、WebSocket を閉じて、セッションをセッション表から削除します。

- `maxSessions` を超えてセッションを作成すると HTTP 429 を返します。接続に失敗した場合は `n` 通知フレームで伝え、その後 WebSocket を閉じます。
- `bytesIn` はブラウザからの入力バイト数、`bytesOut` はターミナルの出力バイト数、`attached` は現在接続しているクライアント数（数値）です。時刻のフィールドは RFC3339 です。

REST：

| メソッド | パス | 説明 |
|---|---|---|
| GET | `/sessions` | `[{id, connKey, connName, type, remark, createdAt, lastActive, attached, cols, rows, bytesIn, bytesOut}]` |
| GET | `/sessions/{id}` | 単一のセッション |
| PUT | `/sessions/{id}/remark` | `{"remark":""}` |
| GET | `/sessions/{id}/stats` | `{bytesIn, bytesOut, durationSec, idleSec, attached, cols, rows}` |
| DELETE | `/sessions/{id}` | セッションを終了 |

## 4. 設定、ショートカット、分割レイアウト

| メソッド | パス | 説明 |
|---|---|---|
| GET/PUT | `/settings` | `{settings, activeSessions}` / ボディは Settings |
| GET/PUT | `/globalshortcuts` | ショートカットコマンドの配列 |
| GET/PUT/DELETE | `/splitlayout` | 任意の JSON レイアウト |

## 5. セキュリティ（利用規約 + 2 段階認証）

| メソッド | パス | 説明 |
|---|---|---|
| GET | `/security` | `{agreementAccepted, twoFAEnable, verified}`。`verified` は現在のログイントークンが 2 段階認証を通過済みかどうかを示します |
| POST | `/security/agreement` | 利用規約に同意 |
| POST | `/security/2fa/setup` | 仮のシークレット `{secret, url}` を生成（保存はしません） |
| POST | `/security/2fa/enable` | `{secret, code}`。検証に通ると 2FA を有効化 |
| POST | `/security/2fa/disable` | `{code}`。検証に通ると 2FA を無効化 |
| POST | `/security/verify2fa` | `{code}`。通過すると、現在のログイントークンは 30 分間検証済みとして扱われます |
| GET | `/security/check2fa` | `{need: bool}`：まだ検証が必要かどうか |

2FA コードを 5 回続けて間違えると、そのログイントークンは 1 分間ロックされます（HTTP 429）。2FA がすでに有効な場合、`2fa/setup` は 400 を返します（先に無効化してください）。`2fa/enable` が成功すると、現在のトークンは検証済みとして扱われます。
ゲート：ローカルシェルの有効化、セッションの作成、セッションへの接続、すべての SFTP エンドポイントは、利用規約に同意していない場合は 403 `{"error":"...","needAgreement":true}` を返し、
2FA が有効で現在のトークンが未検証の場合は 403 `{"error":"...","need2fa":true}` を返します。ここでの「ローカルシェルの有効化」は、`PUT /settings` で `allowLocalShell` を false から true に変更することを指します。Go 側では、これらはすべて `m.requireAccess(w, r) bool` を呼び出します。

## 6. SFTP ファイル管理

パスパラメーター `{key}` は**接続の key** です。SSH 接続は SFTP を使います（リクエストごとに、再利用している SSH 接続上で SFTP サブシステムのチャネルを開きます）。`local` 接続はサーバー自身のファイルシステムを直接操作します（同じエンドポイントを使います）。
Telnet などその他のタイプは 400、存在しない接続は 404 を返します。バックエンドは接続ごとに 1 本の SSH 接続を再利用し、5 分間アイドルが続くと閉じます（`m.acquireSSH`）。
`local` 接続は設定で「ローカルシェルを許可」が有効になっている必要があり、無効なら 403 を返します（ローカルのファイル権限はローカルシェルと同等のため）。SSH 接続に失敗した場合は 502 を返します。

共通の仕様：
- `path` / `from` / `to` / `dest` はすべて `/` で始まる絶対パスでなければなりません（`list` の `path` が空の場合は既定のディレクトリを使います。SSH ではログイン時のディレクトリ、local では接続の workDir またはユーザーのホームディレクトリです）。相対パスは 400 を返します。
- エラーは常に `{"error":"message"}` です。見つからない 404、権限がない 403、大きすぎる 413、未対応の形式 415、対象がすでに存在する 409（この節で追加）、パラメーターの誤り 400、それ以外は 500。
- 書き込み系のエンドポイントは成功すると `{"ok":true, ...}` を返します。

| メソッド | パス | 説明 |
|---|---|---|
| GET | `/sftp/{key}/list?path=` | `{path, parent, entries:[{name, path, isDir, isLink, size, mode, modeStr, modTime, owner?}]}`。ディレクトリが先、その後は大文字小文字を区別しない名前順です。`mode` は `"755"` のような 8 進数の権限文字列、`modeStr` は `drwxr-xr-x` 形式、`modTime` は RFC3339、`owner` は local ではユーザー名、SSH では数値の uid です。シンボリックリンクは `isLink=true` で、ディレクトリを指す場合は `isDir=true` になります |
| GET | `/sftp/{key}/read?path=&download=1` | download なし：テキストのプレビューで、`{path, name, size, modTime, content}` を返します。1MB を超えると 413、バイナリの内容（NUL を含む、または UTF-8 でない）は 415 を返します。download=1：ストリーミングでダウンロードします（`application/octet-stream` + `Content-Disposition: attachment`）。`Range` と `If-Modified-Since` に対応します |
| POST | `/sftp/{key}/write` | `{path, content}` でテキストを保存します（上書き。対象がディレクトリなら 400）→ `{ok, path, size}` |
| POST | `/sftp/{key}/upload` | multipart のストリーミング書き込み：`path`（保存先ディレクトリ。すでに存在している必要があり、**ファイルより前に置く必要があります**）、`files`（複数可）、任意の `relativePaths`（ファイルごとに 1 つ。後に続くファイルと出現順に対応し、対応するファイルより前に置く必要があります。JSON 配列の文字列も受け付けます）。`relativePaths` がない場合はファイル名をそのまま使います（ブラウザでディレクトリをアップロードした場合はサブパスを含むことがあります）。同名のファイルは上書きされ、途中のディレクトリがなければ自動で作成します。`..` を含むパスは 400 を返し、保存先ディレクトリの外には何も書き込まないことを保証します → `{ok, count, files:[絶対パス]}` |
| POST | `/sftp/{key}/mkdir` | `{path}` でディレクトリを作成します（途中のディレクトリも作成）。すでに存在する場合は 409 |
| POST | `/sftp/{key}/touch` | `{path}` で空のファイルを作成します（途中のディレクトリも作成）。すでに存在する場合は 409 |
| POST | `/sftp/{key}/rename` | `{from, to}`。`to` がすでに存在する場合は 409、ディレクトリをそれ自身の中へ移動しようとした場合は 400、ルートディレクトリの名前変更は 403 です。local でデバイスをまたぐ場合は、自動でコピーしてから削除する方法に切り替えます |
| POST | `/sftp/{key}/copy` | `{from:[...], to}`。`to` はコピー先の**ディレクトリ**で、各項目は `to/元の名前` にコピーされます（再帰的。シンボリックリンクはリンクのままコピーし、デバイスやパイプなどの特殊ファイルは無視します）。名前が衝突した場合はエラーにせず、`name copy.ext`、`name copy 2.ext` のように**自動で名前を変えて**コピーします。ディレクトリをそれ自身の中へコピーしようとした場合は 400 → `{ok, paths:[新しいパス]}` |
| POST | `/sftp/{key}/chmod` | `{path, mode:"755", recursive}`。`mode` は 0〜7777 の 8 進数の文字列です。再帰処理はシンボリックリンクをたどらず、ファイルにもディレクトリにも同じ権限を設定します |
| DELETE | `/sftp/{key}/remove` | `{paths:[...]}`（JSON ボディ。再帰的に削除し、シンボリックリンクはリンクだけを削除）。いずれかのパスが `/` なら 403 を返し、何も削除しません → `{ok, count}` |
| POST | `/sftp/{key}/compress` | `{paths:[...], dest, format:"zip|tar.gz"}`（format の既定値は zip。`tgz` は tar.gz として扱います）。`dest` は出力するアーカイブのフルパスです。`dest` が既存のディレクトリか `/` で終わる場合は、その中に自動で名前を付けて作成します（項目が 1 つならその名前、複数なら親ディレクトリの名前に `.zip` / `.tar.gz` を付けたもの）。各項目は自身の名前でアーカイブの最上位に入ります。同名の項目があれば 400、出力ファイルがすでに存在すれば 409 を返し、出力ファイル自体をアーカイブに含めることはありません → `{ok, path}` |
| POST | `/sftp/{key}/decompress` | `{path, dest}`。zip / tar / tar.gz / tgz に対応します（拡張子で判定し、それ以外は 415）。`dest` が空の場合はアーカイブのあるディレクトリに展開し、必要に応じて作成します。同名のファイルは上書きされます。`..` を含むエントリが 1 つでもあるアーカイブは丸ごと拒否します（400、zip-slip 対策）。シンボリックリンクやハードリンクなど通常のファイル以外のエントリはスキップし、ファイルの権限は rwx ビットだけを保持します → `{ok, dest, count}` |
| GET | `/sftp/{key}/preview-archive?path=` | アーカイブのエントリ一覧 `[{name,size,isDir}]`（最大 10000 件。`name` の末尾に `/` は付きません） |

圧縮と展開はすべて Go の中でファイルシステムのインターフェースを通じて行い（local は os、SSH は SFTP の読み書き）、リモートのコマンドには依存しません。

## 7. その他

- ログ：モジュールのログはすでに全体のログページに表示されるため、専用のエンドポイントは不要です。
- Lucky からのインポート：telnet、proxy、keyFile、workDir、timeout、env、quickAccessDirs、sessionConfig、globalShortcuts、
  securityConfig を取り込みます（利用規約への同意と 2FA シークレットも一緒に移行します）。
