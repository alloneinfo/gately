# Gately

[English](README.md) | [简体中文](README.zh-CN.md) | **繁體中文** | [日本語](README.ja.md) | [한국어](README.ko.md) | [Español](README.es.md) | [Français](README.fr.md) | [Deutsch](README.de.md) | [Русский](README.ru.md) | [Português (Brasil)](README.pt-BR.md)

開源、可稽核的網路閘道工具集，家用網路、NAS、雲端主機與各類伺服器皆可使用，靈感來自 [Lucky](https://github.com/gdy666/lucky)，以完全開源的方式提供同類功能。DDNS、STUN 穿透、連接埠轉發、反向代理、憑證、WAF 等常用功能整合在一個約 22 MB 的單一檔案中，並可一鍵匯入 Lucky 設定。

## 為什麼做 Gately

Lucky 是中國 NAS 與家用網路使用者熟悉的優秀工具，把 DDNS、STUN 穿透、反向代理等功能整合在一起，也為 Gately 提供了產品思路，沒有 Lucky 就不會有 Gately。Lucky 以閉源方式免費散布，Gately 選擇完全開源，讓使用者和社群可以直接檢視原始碼與設定。對於運行在網路邊界、掌握 DNS 金鑰與憑證私鑰的軟體，開源讓使用者能夠看見它在做什麼。Gately 的承諾：

- 原始碼以 Apache-2.0 開源，接受社群審查
- 預設不連線任何專案方伺服器，不回報裝置資訊；第三方登入使用你自己的 OAuth 應用程式，不經過中轉
- 所有對外位址（公網 IP 查詢、STUN 伺服器等）皆可檢視、可修改、可關閉
- 設定為明文 JSON，可直接納入 git 進行稽核與還原

## 功能

| 模組 | 功能 |
| --- | --- |
| 動態網域 | Alibaba Cloud、Cloudflare、Tencent Cloud、DNSPod、Huawei Cloud、GoDaddy、Porkbun、dynv6、DuckDNS、自訂回呼；以 URL／網路卡／指令取得 IP；IPv4/IPv6；Webhook |
| STUN 穿透 | TCP/UDP、保活、UPnP/NAT-PMP、內建轉發、位址變更時觸發 Webhook 與腳本 |
| 連接埠轉發 | TCP/UDP、IPv4/IPv6、連接埠範圍、多目標故障轉移、連線數限制、流量統計 |
| SSL 憑證 | ACME DNS-01 自動簽發與續期（萬用字元、EAB）、上傳、讀取檔案、對應到目錄並執行腳本 |
| Tailscale | 託管 tailscaled（使用者空間網路，容器友善）或接入系統既有的常駐程式；驗證金鑰／登入連結、Headscale、子網路由、出口節點、節點清單與 SOCKS5 代理、固定連接埠、tailscaled 額外參數與環境變數 |
| frp | frpc 用戶端（tcp/udp/http/https/stcp/xtcp 代理，新增刪除即時生效）與 frps 伺服器端（連接埠白名單、子網域、主控台），可一鍵匯入舊版 ini／新版 toml 設定 |
| Cloudflare 通道 | cloudflared 權杖通道與 trycloudflare 臨時通道，多通道、自動重啟、公開位址顯示 |
| Web 服務 | 依網域反向代理／重新導向／靜態檔案／固定文字；HTTPS；WebSocket；PROXY 協定收發；基本驗證、網頁驗證、GitHub/Google/QQ/OIDC 登入；IP 黑白名單、路徑規則、臨時分享連結、驗證失敗自動封鎖、GeoIP 國家過濾；後端權重／備援／負載平衡策略、主動與被動健康檢查、壓縮、限流、回應快取、路徑改寫；分鐘級存取統計與 Prometheus 指標 |
| WAF 防火牆 | Coraza + OWASP CRS v4，可依檔案啟用規則、自訂 SecLang、攔截／僅偵測 |
| Docker 管理 | 容器建立與編輯、日誌與終端機、Compose、映像檔更新檢查與一鍵升級（失敗自動還原）、網路與磁碟區 |
| Web 終端機 | SSH（主機金鑰首次信任）、Telnet、本機 Shell（預設關閉）；代理、持久工作階段、SFTP 檔案管理、2FA 與協定確認、快捷指令、分割畫面 |
| 網路診斷 | Ping、路由追蹤、連接埠探測、DNS 查詢、測速、區域網路掃描 |
| 通知中心 | Webhook、Telegram、Bark、WeCom、DingTalk、Feishu、電子郵件；等級、萬用字元與靜音去重規則 |
| 設定備份 | 加密備份（scrypt + AES-GCM）、保留策略、定時備份到本機目錄或儲存掛載點 |
| 命令列與 MCP | `gately ctl` 呼叫執行中的實例，`gately mcp` 以 stdio 提供 MCP 服務（預設唯讀） |
| 網路喚醒 | 魔術封包廣播、上線探測 |
| WebDAV | 多使用者、獨立目錄、唯讀 |
| FTP | 多使用者、獨立目錄、唯讀，支援 FTPS 與被動連接埠範圍 |
| 儲存管理 | 本機目錄與遠端 WebDAV 掛載，線上檔案管理 |
| 排程工作 | cron 運算式，指令或 HTTP 請求，執行記錄 |
| 安全與設定 | 兩步驟驗證（TOTP）、登入失敗限流、安全入口、僅內網存取、管理介面 HTTPS、多組 API 權杖（依模組讀寫授權）、自訂 DNS（支援 DoH）、時區、設定備份與還原 |
| 遷移 | 匯入 Lucky 的 `.lkcf` 或整站設定 zip：連接埠轉發、DDNS、STUN、憑證、反向代理、WAF、網路喚醒、Web 終端機、WebDAV、基礎設定 |

## 快速開始

### Docker

映像檔支援 amd64、arm64、arm/v7：

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep -i "initial password"
```

開啟 `http://<裝置IP>:16680`，使用 `admin` 與日誌中的初始密碼登入，然後在「設定」中修改密碼並開啟兩步驟驗證。完整的 Compose 範例見 [deploy/docker-compose.yml](deploy/docker-compose.yml)。

連接埠轉發、STUN、Web 服務需要監聽主機連接埠，建議使用 `--network host`。不需要 Docker 管理時可拿掉 `docker.sock` 掛載。

### 二進位檔

從 [Releases](https://github.com/wangwanjie/gately/releases) 下載對應平台的壓縮檔（Linux amd64/arm64/arm/mips/mipsle/riscv64、macOS、Windows、FreeBSD），解壓縮後執行：

```bash
./gately -c /etc/gately
```

參數：`-c` 設定目錄（也可用環境變數 `GATELY_CONFIG_DIR`），`-listen` 覆寫管理介面監聽位址，`-version` 輸出版本。

### API 呼叫

在管理後台「API 權杖」頁建立權杖（可建立多組，依模組分別授予唯讀或讀寫，可設定到期時間，建立後明文只會顯示一次），腳本即可批次呼叫管理介面：

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks
```

帳號、兩步驟驗證、權杖管理以及備份、還原、重啟、系統設定介面只能使用登入工作階段。全部介面、所需權限與資料結構見管理後台「API 文件」頁，也可以用 `make api-docs` 匯出到 [docs/api](docs/api/zh-TW)（`openapi.json` 與 `API.md`），官網 https://gately.cn 發布時直接引用。

從舊版升級時，原本的全域 API 權杖會自動變成一筆「舊版權杖」（值不變，擁有全部模組權限），建議在 API 權杖頁收緊權限或換成新權杖。

### Tailscale、frp、cloudflared

這三個模組託管官方的外部程式（自動重啟、日誌彙整到介面），Gately 本體不內建它們，因此始終可以使用上游的最新版本。程式依下列順序尋找：模組設定中填寫的路徑、`PATH`、設定目錄下的 `bin/` 子目錄。Docker 部署時把二進位檔放進掛載的設定目錄即可，例如 `./gately-data/bin/frpc`；Tailscale 需要同時提供 `tailscale` 與 `tailscaled`。設定檔中的權杖與金鑰以 0600 權限儲存，cloudflared 的權杖透過環境變數傳遞，不會出現在處理程序清單中。

### 從 Lucky 遷移

見[遷移指南](https://gately.cn/zh-TW/docs/guide/migration)。簡要步驟：在「設定 → 從 Lucky 匯入」選擇 Lucky 設定目錄中的 `lucky_*.lkcf` 檔案，預覽後匯入；匯入的規則預設停用，依模組逐一在 Lucky 中停用、在 Gately 中啟用。

## 開發

相依套件：Go 1.26+、Node.js 24+、pnpm。

```bash
make web      # 建置前端（輸出到 web/dist，由 go:embed 嵌入）
make build    # 編譯 bin/gately
make test     # go test -race ./...
make lint     # gofmt、go vet、前端型別檢查
```

前端開發時執行 `cd web && pnpm dev`，`/api` 會代理到本機 `127.0.0.1:16680`。

部分測試預設跳過，需要額外條件：

| 測試 | 啟用方式 |
| --- | --- |
| ACME 端對端（Pebble） | 安裝 pebble 與 pebble-challtestsrv，設定 `GATELY_PEBBLE_BIN`、`GATELY_PEBBLE_SRC`（CI 中自動執行） |
| 真實公共 STUN 伺服器 | `GATELY_NET_TEST=1` |
| 以真實 Lucky 設定驗證匯入 | `GATELY_LUCKY_SAMPLE` 等，見各模組 `TestImportRealSample` |

目錄結構與技術選型見 [docs/ROADMAP.md](docs/ROADMAP.md)。

官網（[gately.cn](https://gately.cn)）原始碼在 `website/`，基於 VitePress：`cd website && pnpm install && pnpm dev`；`pnpm deploy:site` 建置並同步到伺服器。

## 貢獻約定

- 與 Lucky 的相容性基於公開可觀察的行為、介面和設定格式。請勿提交從 Lucky 二進位檔或前端產物中複製的程式碼。
- 新模組需要：設定結構與驗證、執行階段啟停、REST 介面、單元測試；能從 Lucky 遷移的模組實作 `luckyimport.Importer`，希望在總覽頁顯示狀態的模組實作 `module.Summarizer`。
- 程式碼註解使用中文；控制語句必須使用大括號。

## Star 趨勢

[![Star History Chart](https://api.star-history.com/svg?repos=wangwanjie/gately&type=Date)](https://star-history.com/#wangwanjie/gately&Date)

## 授權條款

[Apache License 2.0](LICENSE)
