# Web 終端機介面約定

所有介面掛載在 `/api/webterminal/` 下，需登入。錯誤統一為 `{"error": "..."}`。

## 1. 設定結構（`conf` 中的 webterminal 檔案）

```jsonc
{
  "settings": {
    "allowLocalShell": false,
    "maxSessions": 20,
    "idleTimeout": 0,          // 分鐘，工作階段無輸入輸出超過該時間自動結束，0 不限
    "bufferSize": 256,         // KB，每個工作階段保留的輸出回放緩衝區，重新連線時回放
    "heartbeatInterval": 30,   // 秒，WebSocket 心跳（伺服器端 ping / 前端 ping）
    "sessionKeepAlive": 0      // 分鐘，瀏覽器中斷連線後工作階段繼續保留的時間；0 表示中斷即結束
  },
  "connections": [Connection],
  "shortcuts": [{ "name": "檢視磁碟", "command": "df -h", "sendEnter": true }],
  "splitLayout": null,         // 前端自訂的分割畫面版面配置，後端原樣存取（json.RawMessage）
  "security": {
    "agreementAccepted": false,
    "twoFAEnable": false,
    "twoFAKey": ""             // TOTP 金鑰，介面輸出時遮蔽
  },
  "credentials": [Credential]  // SSH 認證預設
}
```

Connection（`type` 為 `ssh` / `telnet` / `local`）：

```jsonc
{
  "key": "", "name": "", "type": "ssh", "remark": "", "enable": true,
  "quickAccessDirs": ["/etc", "/var/log"],      // SFTP 快速存取目錄
  "ssh": { "host", "port", "username", "authType": "password|key|credential",
           "credentialKey": "",  // authType=credential 時引用的認證預設 key；此時 username 可留空，取預設的預設使用者名稱
           "password", "privateKey",
           "keyFile": "",        // 伺服器本機上的私鑰檔案路徑，authType=key 且 privateKey 為空時使用
           "passphrase", "hostKey", "keepAlive", "workDir": "", "timeout": 15,
           "proxy": Proxy },
  "telnet": { "host", "port": 23, "username", "password", "timeout": 15, "proxy": Proxy },
  "local": { "shell", "workDir", "env": {"K":"V"} }
}
Proxy = { "enabled": false, "type": "socks5|http", "host", "port", "username", "password" }
```

Credential（認證預設，可被多個 SSH 連線引用）：

```jsonc
{ "key": "", "name": "", "authType": "password|key", "username": "", // username 選填，連線未填寫使用者名稱時使用
  "password", "privateKey", "keyFile", "passphrase", "remark": "", "enable": true }
```

密碼、私鑰、私鑰密語、代理密碼輸出時用 `crud.Mask` 遮蔽，修改時用 `crud.KeepSecret` 保留。

## 2. 連線管理

| 方法 | 路徑 | 說明 |
|---|---|---|
| GET/POST | `/connections` | 清單 / 新增 |
| PUT/DELETE | `/connections/{key}` | 修改 / 刪除 |
| PUT | `/connections-order` | 排序 `{"keys":[...]}` |
| POST | `/connections/{key}/test` | 測試 SSH / Telnet 連通性 |
| POST | `/connections/test` | 測試未儲存的連線（body 為 Connection；已儲存的密文以 `key` 欄位指向原連線合併） |
| DELETE | `/connections/{key}/hostkey` | 重設主機指紋 |
| PUT | `/connections/{key}/quickaccess` | body `{"dirs":["/a","/b"]}` |
| GET/POST | `/credentials` | 認證預設清單 / 新增 |
| PUT/DELETE | `/credentials/{key}` | 修改 / 刪除（仍被連線引用時回傳 409 並列出連線名稱） |
| PUT | `/credentials-order` | 預設排序 `{"keys":[...]}` |
| GET | `/shells` | 本機可用 Shell 清單 `["/bin/bash", ...]`（讀取 /etc/shells 並過濾出存在的項目） |

補充說明（以實作為準）：
- `POST /connections/test` 與 `POST /connections/{key}/test` 回傳 200，`{ok, error?, serverVersion?, newHostKey?, latencyMs}`；`local` 類型回傳 400。Telnet 僅驗證 TCP（含代理）可達。
- `PUT /connections/{key}/quickaccess` 回傳 `{ok, dirs}`（去除空白與重複）；編輯連線時若不帶 `quickAccessDirs` 則沿用舊值。
- 連線的 `ssh.timeout` / `telnet.timeout` 取值 1~120 秒，預設 15。`ssh.workDir` 在登入後透過 `cd` 切換。
- `authType=credential` 的連線儲存時會清空連線內的密碼、私鑰、私鑰檔案與密語，連線時從預設解析憑證；修改預設會中斷引用它的池化 SSH 連線。
- Telnet 登入自動填入僅在偵測到 `login:` / `username:` / `password:` 等提示時各嘗試一次。

## 3. 終端機工作階段（持久工作階段，可斷線重連）

- `GET /connections/{key}/ws?cols=&rows=`：新建工作階段並升級為 WebSocket。
- `GET /sessions/{id}/ws?cols=&rows=`：附著到既有工作階段，先回放緩衝區再即時轉發；同一工作階段允許多個用戶端同時附著（輸入共用、輸出廣播）。

WebSocket 協定：
- 伺服器端 → 瀏覽器：**二進位訊框**為終端機輸出；**文字訊框**為 JSON：
  `{"t":"s","d":"<sessionId>"}` 工作階段建立（新建與附著後都會先送一次）；
  `{"t":"n","d":"提示"}` 提示；`{"t":"x","d":"原因"}` 工作階段已結束；`{"t":"p"}` 心跳回應。
- 瀏覽器 → 伺服器端：文字 JSON：`{"t":"i","d":"輸入"}`、`{"t":"r","c":欄,"r":列}`、`{"t":"p"}` 心跳。
- 瀏覽器 WebSocket 中斷：`sessionKeepAlive > 0` 時工作階段保留（detached），逾時無人附著後關閉；為 0 時立即關閉。
- 工作階段的行程自行結束時送出 `x` 並關閉 WebSocket，從工作階段表中移除。

- 超過 `maxSessions` 時新建工作階段回傳 HTTP 429；連線失敗時透過 `n` 提示訊框告知後關閉 WebSocket。
- `bytesIn` 為瀏覽器輸入的位元組數，`bytesOut` 為終端機輸出的位元組數；`attached` 為目前附著的用戶端數量（number）；時間欄位為 RFC3339。

REST：

| 方法 | 路徑 | 說明 |
|---|---|---|
| GET | `/sessions` | `[{id, connKey, connName, type, remark, createdAt, lastActive, attached, cols, rows, bytesIn, bytesOut}]` |
| GET | `/sessions/{id}` | 單一工作階段 |
| PUT | `/sessions/{id}/remark` | `{"remark":""}` |
| GET | `/sessions/{id}/stats` | `{bytesIn, bytesOut, durationSec, idleSec, attached, cols, rows}` |
| DELETE | `/sessions/{id}` | 結束工作階段 |

## 4. 設定、快捷命令、分割畫面版面配置

| 方法 | 路徑 | 說明 |
|---|---|---|
| GET/PUT | `/settings` | `{settings, activeSessions}` / body 為 Settings |
| GET/PUT | `/globalshortcuts` | 快捷命令陣列 |
| GET/PUT/DELETE | `/splitlayout` | 任意 JSON 版面配置 |

## 5. 安全（使用者協議 + 兩步驟驗證）

| 方法 | 路徑 | 說明 |
|---|---|---|
| GET | `/security` | `{agreementAccepted, twoFAEnable, verified}`，`verified` 表示目前登入權杖已通過兩步驟驗證 |
| POST | `/security/agreement` | 接受協議 |
| POST | `/security/2fa/setup` | 產生待啟用的金鑰 `{secret, url}`（不寫入磁碟） |
| POST | `/security/2fa/enable` | `{secret, code}` 驗證通過後啟用 |
| POST | `/security/2fa/disable` | `{code}` 驗證通過後關閉 |
| POST | `/security/verify2fa` | `{code}` 通過後目前登入權杖 30 分鐘內視為已驗證 |
| GET | `/security/check2fa` | `{need: bool}`，是否還需要驗證 |

2FA 驗證碼連續失敗 5 次後，該登入權杖鎖定 1 分鐘（HTTP 429）；`2fa/setup` 在已啟用時回傳 400（需先關閉）；`2fa/enable` 成功後目前權杖視為已驗證。
門禁：開啟本機 Shell / 新建工作階段 / 附著工作階段 / SFTP 所有介面，在協議未接受時回傳 403 `{"error":"...","needAgreement":true}`，
啟用 2FA 且目前權杖未驗證時回傳 403 `{"error":"...","need2fa":true}`。其中「開啟本機 Shell」指 `PUT /settings` 把 `allowLocalShell` 由 false 改為 true。Go 端統一呼叫 `m.requireAccess(w, r) bool`。

## 6. SFTP 檔案管理

路徑參數 `{key}` 為**連線 key**。SSH 連線走 SFTP（每次請求在重複使用的 SSH 連線上開一個 SFTP 子系統通道）；`local` 連線直接操作伺服器本機檔案系統（同一套介面）。
Telnet 等其他類型回傳 400；連線不存在回傳 404。後端為每個連線重複使用一條 SSH 連線，閒置 5 分鐘後關閉（`m.acquireSSH`）。
`local` 連線要求設定裡已開啟「允許本機 Shell」，否則回傳 403（本機檔案權限等同本機 Shell）。SSH 連線失敗回傳 502。

通用約定：
- 所有 `path` / `from` / `to` / `dest` 必須是以 `/` 開頭的絕對路徑（`list` 的 `path` 為空時取預設目錄：SSH 為登入目錄，local 為連線的 workDir 或使用者家目錄）；相對路徑回傳 400。
- 錯誤統一為 `{"error":"錯誤訊息"}`：不存在 404、權限不足 403、過大 413、類型不支援 415、目標已存在 409（本節擴充）、參數錯誤 400、其餘 500。
- 成功的寫入類介面回傳 `{"ok":true, ...}`。

| 方法 | 路徑 | 說明 |
|---|---|---|
| GET | `/sftp/{key}/list?path=` | `{path, parent, entries:[{name, path, isDir, isLink, size, mode, modeStr, modTime, owner?}]}`，目錄在前，依名稱不分大小寫排序。`mode` 為八進位權限字串如 `"755"`，`modeStr` 為 `drwxr-xr-x` 形式，`modTime` 為 RFC3339，`owner` 在 local 為使用者名稱、SSH 為數字 uid。符號連結 `isLink=true`，指向目錄時 `isDir=true` |
| GET | `/sftp/{key}/read?path=&download=1` | 無 download：文字預覽，回傳 `{path, name, size, modTime, content}`；超過 1MB 回傳 413，含 NUL 或非 UTF-8 的二進位檔案回傳 415。download=1：串流下載（`application/octet-stream` + `Content-Disposition: attachment`），支援 `Range` 與 `If-Modified-Since` |
| POST | `/sftp/{key}/write` | `{path, content}` 儲存文字（覆寫，目標為目錄時回傳 400）→ `{ok, path, size}` |
| POST | `/sftp/{key}/upload` | multipart 串流寫入：`path`（目標目錄，必須已存在，且**必須位於 files 之前**）、`files`（可多個）、選填 `relativePaths`（每個檔案一個，依出現順序與其後的 files 一一對應，必須位於對應的 files 之前；也接受 JSON 陣列字串）。未提供 `relativePaths` 時使用 filename 原文（瀏覽器上傳目錄時可帶子路徑）。同名檔案覆寫，缺少的中間目錄自動建立；包含 `..` 的路徑回傳 400，確保不會寫出目標目錄 → `{ok, count, files:[絕對路徑]}` |
| POST | `/sftp/{key}/mkdir` | `{path}` 建立目錄（含缺少的上層目錄），已存在時 409 |
| POST | `/sftp/{key}/touch` | `{path}` 建立空檔案（含缺少的上層目錄），已存在時 409 |
| POST | `/sftp/{key}/rename` | `{from, to}`，`to` 已存在時 409，把目錄移到自身內部時 400，禁止重新命名根目錄 403；local 跨裝置時自動改為複製後刪除 |
| POST | `/sftp/{key}/copy` | `{from:[...], to}`，`to` 是目標**目錄**，每項複製到 `to/原名`（遞迴，符號連結以連結本身複製，忽略裝置/管道等特殊檔案）。目標重名時**自動改名**為 `name 副本.ext`、`name 副本 2.ext`……，不報錯；複製目錄到自身內部時 400 → `{ok, paths:[新路徑]}` |
| POST | `/sftp/{key}/chmod` | `{path, mode:"755", recursive}`，`mode` 為 0–7777 的八進位字串；遞迴時不跟隨符號連結，檔案與目錄統一設為同一權限 |
| DELETE | `/sftp/{key}/remove` | `{paths:[...]}`（body JSON，遞迴刪除，符號連結只刪連結本身）；任一路徑為 `/` 時回傳 403 且不刪除任何項目 → `{ok, count}` |
| POST | `/sftp/{key}/compress` | `{paths:[...], dest, format:"zip|tar.gz"}`（format 預設 zip，`tgz` 視為 tar.gz）。`dest` 為輸出壓縮檔的完整路徑；若 `dest` 是已存在的目錄或以 `/` 結尾，則在其中自動命名（單一項目用其名稱，多個項目用其上層目錄名稱，加上 `.zip` / `.tar.gz`）。每個項目以自身名稱作為封存頂層，同名項目 400，輸出檔案已存在 409，輸出檔案本身不會被打包進去 → `{ok, path}` |
| POST | `/sftp/{key}/decompress` | `{path, dest}` 支援 zip / tar / tar.gz / tgz（依副檔名辨識，其他 415）。`dest` 為空時解壓縮到壓縮檔所在目錄，不存在則建立；同名檔案覆寫。含 `..` 的項目整體拒絕（400，防範 zip-slip）；符號連結、硬連結等非一般檔案的項目會被略過；檔案權限只保留 rwx 位元 → `{ok, dest, count}` |
| GET | `/sftp/{key}/preview-archive?path=` | 壓縮檔項目清單 `[{name,size,isDir}]`（最多 10000 筆，`name` 不含結尾的 `/`） |

壓縮/解壓縮全部由 Go 透過檔案系統介面完成（local 用 os，SSH 用 SFTP 讀寫），不依賴遠端命令。

## 7. 其他

- 日誌：模組日誌已由全域日誌頁統一顯示，無需單獨介面。
- Lucky 匯入：補齊 telnet、代理、keyFile、workDir、timeout、env、quickAccessDirs、sessionConfig、globalShortcuts、
  securityConfig（使用者協議與 2FA 金鑰一併遷移）。
