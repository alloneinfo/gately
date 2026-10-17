# 強化建議

管理介面控制著 DNS、憑證與連接埠，建議依以下清單設定。

## 管理介面

- [ ] 首次登入後修改管理員密碼
- [ ] 開啟兩步驟驗證（TOTP）
- [ ] 設定安全入口，使管理介面只能透過自訂路徑存取
- [ ] 開啟「僅限內網存取」，需要從外網管理時透過 VPN（如 [Tailscale](/zh-TW/docs/modules/tunnels)）進入
- [ ] 在不可信的網路中存取時，啟用管理介面 HTTPS
- [ ] 自動化指令碼與 [命令列 / MCP](/zh-TW/docs/guide/cli-mcp) 使用最小權限的 [API 權杖](/zh-TW/docs/reference/api)，設定到期時間，避免共用管理員密碼
- [ ] 為 [設定備份](/zh-TW/docs/modules/backup) 設定加密密碼，並開啟排程備份
- [ ] 在 [通知中心](/zh-TW/docs/modules/notify) 設定登入、憑證、備份失敗等事件的推送

## 對外服務

- 透過 [Web 服務](/zh-TW/docs/modules/web) 公開內網服務時，啟用 HTTPS，並搭配基本驗證、網頁驗證或第三方登入；
- 對公網站台開啟 [WAF](/zh-TW/docs/modules/waf)，先以「僅偵測」觀察誤判，再切換為「攔截」；
- 使用 IP 黑白名單、GeoIP 國家過濾與驗證失敗自動封鎖限制來源。

## Docker

掛載 `docker.sock` 等同於授予主機 root 等級的權限。不需要 [Docker 管理](/zh-TW/docs/modules/docker) 時請移除該掛載。

## Web 終端機

本機 Shell 預設關閉，確有需要時再開啟，並確保管理介面已啟用兩步驟驗證。

## 設定備份

把設定目錄納入私有 git 儲存庫或定期備份。`ssl.json` 中含憑證私鑰，請妥善保管。
