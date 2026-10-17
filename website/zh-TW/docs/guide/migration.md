# 從 Lucky 遷移到 Gately

Gately 可以直接讀取 Lucky 的 `.lkcf` 設定檔，或 Lucky「備份設定」匯出的整站 zip，把規則、工作、憑證遷移過來。整個過程不需要移除 Lucky，可以兩邊並存、逐個模組切換。

## 1. 準備

1. 找到 Lucky 的設定目錄：Docker 部署通常是對應到容器 `/app/conf` 的主機目錄，裡面有 `lucky_base.lkcf`、`lucky_ddns.lkcf` 等檔案。
2. 部署 Gately（見 README）。建議：
   - 使用 `--network host`，與 Lucky 一樣直接監聽主機的連接埠；
   - 管理連接埠預設為 16680，與 Lucky 的 16601 不衝突；
   - 如果 Lucky 的指令碼或憑證對應用到了容器內路徑（例如 `/etc/lucky/custom_scripts`、`/data/nginx/...`），在 Gately 容器中掛載到**相同路徑**，匯入後無需修改。

## 2. 匯入

開啟 Gately「設定 → 備份與遷移」，選擇全部 `lucky_*.lkcf` 檔案，或直接選擇 Lucky 匯出的整站設定 zip（壓縮檔內的 `.lkcf` 會被自動辨識，其他檔案忽略），先點「預覽」查看每個模組能匯入多少筆、有哪些提示，確認後點「匯入」。

重複匯入是安全的：與既有項目同名的規則、工作、憑證會被略過。

支援的檔案：

| 檔案 | 匯入到 | 說明 |
| --- | --- | --- |
| lucky_portforward.lkcf | 連接埠轉發 | TLS/串流加密、限速選項暫不支援，會提示 |
| lucky_ddns.lkcf | 動態網域 | 支援 11 家服務商；Lucky 官方 IP 查詢位址會被移除 |
| lucky_stun.lkcf | STUN 穿透 | 含全域 STUN 伺服器清單與全域 Webhook |
| lucky_ssl.lkcf | SSL 憑證 | 已簽發的憑證原樣匯入，到期前不會重新申請 |
| lucky_reverseproxy.lkcf | Web 服務 | 站台、驗證、WAF 引用（子規則 WAF 為 main 時沿用監聽規則的實例） |
| lucky_coraza.lkcf | WAF 防火牆 | 沿用實例 key，站台引用保持有效 |
| lucky_wol.lkcf | 網路喚醒 | |
| lucky_webterminal.lkcf | Web 終端機 | SSH 連線；首次連線時記錄主機指紋 |
| lucky_webdav.lkcf | WebDAV | 使用者的掛載清單轉為多個目錄，同名同密碼的使用者合併；引用儲存管理的掛載解析為對應的本機目錄 |
| lucky_ftpserver.lkcf | FTP | 監聽連接埠、被動連接埠範圍、TLS 與逾時參數，以及使用者（處理方式同 WebDAV） |
| lucky_storagemanagement.lkcf | 儲存管理 | 本機目錄與 WebDAV 儲存；阿里雲盤、系統掛載與存取代理不遷移 |
| lucky_frp.lkcf | frp | 第一個用戶端實例（伺服器位址、權杖、傳輸協定與代理）與第一個伺服器端實例；訪問者、外掛代理與控制台不遷移 |
| lucky_cloudflared.lkcf | Cloudflare 通道 | 使用通道權杖的通道；Access 存取端與只填寫 API 權杖的通道不遷移 |
| lucky_base.lkcf | 全域設定、Web 服務 | 僅允許內網存取、登入失敗上限、工作階段時長、時區、自訂 DNS（含 doh@ 寫法）、GC 參數、PROXY 協定可信來源。管理連接埠、安全入口、管理員帳號與兩步驟驗證不遷移 |

Lucky 的排程工作、IP 過濾、Docker 設定在常見設定中為空，暫不遷移；有內容時請手動設定。

**匯入的規則預設停用**：避免在測試機上匯入後把正式網域解析到錯誤位址，或與仍在執行的 Lucky 爭用連接埠和路由器 UPnP 對應。確認 Gately 就執行在目標裝置上、且 Lucky 已停止時，可以勾選「匯入後保持 Lucky 中的啟用狀態」，規則會依 Lucky 中的啟用狀態（包括各模組的總開關）立即執行。

## 3. 切換

建議依模組逐個切換，切一個驗證一個：

1. **憑證**：在 Gately 中啟用憑證。啟用後會把憑證寫入對應目錄並執行變更指令碼（與 Lucky 相同的檔名），請確認目錄可寫入。
2. **DDNS**：在 Lucky 中停用工作 → 在 Gately 中啟用對應工作 → 點「強制同步」確認記錄已更新。
3. **連接埠轉發 / STUN**：先在 Lucky 中停用規則（釋放連接埠和 UPnP 對應），再在 Gately 中啟用。STUN 規則啟用後公網連接埠可能變動，指令碼與 Webhook 會自動通知。
4. **Web 服務**：停用 Lucky 的監聽規則 → 在 Gately 中啟用。HTTPS 站台需要憑證模組中已有涵蓋該網域的已啟用憑證。
5. 全部確認無誤後再停止 Lucky 容器。

## 4. 已在真實設定上驗證的行為

以下結論來自一份包含數十條規則的真實 Lucky 2.27.2 設定，在同一台裝置上讓 Gately 與 Lucky 並行執行比對得出：

- **DDNS**：阿里雲工作用匯入的金鑰完成驗證，全部記錄與線上值一致，未產生任何寫入。
- **連接埠轉發**：每條規則的轉發結果與直連目標一致；目標離線的規則在兩邊表現相同。
- **Web 服務**：每個站台經 Gately 與經 Lucky 回傳相同的狀態碼、頁面與憑證；WAF 對路徑穿越的攔截結果一致。
- **憑證對應**：寫出的 `.crt/.key/.pem/_issuerCertificate.crt` 與 Lucky 寫出的檔案逐位元組相同。
- **Web 終端機**：可連線的主機均驗證成功；無法連線或金鑰失效的連線兩邊表現相同（可用「測試連線」快速檢查）。
- **STUN**：規則設定遷移正確。路由器 WAN 口為電信級 NAT（100.64.0.0/10）時，外部無法存取穿透連接埠，這是網路本身的限制，Lucky 同樣無法穿透。

## 5. Docker 部署的掛載

Lucky 的指令碼與憑證對應使用的是 Lucky 容器內的路徑，Gately 容器需要把相同的主機目錄掛載到相同的容器內路徑。對照 Lucky 的 compose 逐條新增，例如：

```yaml
volumes:
  - ./config:/etc/gately
  - /var/run/docker.sock:/var/run/docker.sock
  # Lucky 中憑證對應到 /data/nginx/ssl，則掛載同一個主機目錄到同一路徑
  - /srv/nginx/ssl:/data/nginx/ssl
  # 規則中呼叫的指令碼（請核對 Lucky 中填寫的路徑在 Lucky 容器內是否真實存在）
  - /srv/lucky/custom_scripts:/etc/lucky/custom_scripts:ro
```

Gately 映像檔內建 `sh`、`bash` 與 `curl`，常見的通知指令碼可以直接執行。

## 6. 需要人工處理的項目

- **第三方登入**：Lucky 的 GitHub/QQ/Google 登入經過 Lucky 官方中轉伺服器，Gately 改為使用你自己申請的 OAuth 應用程式。匯入時請同時選擇 `lucky_thirdPartyAuthManager.lkcf`（身分表）、`lucky_reverseproxy.lkcf` 與 `lucky_base.lkcf`：
  - 站台與管理後台引用的第三方身分會依身分表遷移為允許清單（GitHub 用數字 ID、Google 用 sub、QQ 用 openid，使用者名稱與電子郵件寫在備註裡）。
  - 管理後台的登入方式在「設定 → 帳號與安全 → 第三方登入」，站台的登入方式在「Web 服務 → 全域設定」；填寫自己的 Client ID/Secret 與回呼位址並啟用後才會生效，匯入不會自動啟用。登入方式識別碼使用 `github`、`google`、`qq`，才能與遷移來的身分對應。
  - QQ 的 openid 在每個 QQ 互聯應用程式中都不同：換成自己的應用程式後，首次登入會被拒絕並提示新的帳號識別碼，把它加入允許清單即可。
  - 只依賴第三方登入、且找不到可遷移身分的站台，匯入後維持停用。
- **ZeroSSL 憑證**：沿用 Lucky 中的 EAB 憑據註冊新的 ACME 帳號；若 ZeroSSL 拒絕，請在 ZeroSSL 後台重新產生 EAB。
- **frp 實例**：Gately 只有一個 frpc 與一個 frps，Lucky 中多餘的實例會在報告中列出，請改用其他機器或合併到同一實例；Gately 中已設定連線其他伺服器端的 frpc、或已啟用的 frps 不會被覆蓋。frps 控制台的帳號由 Lucky 管理，遷移後如需控制台請重新設定連接埠與密碼。
- **儲存引用**：Lucky 的 FTP / WebDAV 使用者可以掛載「儲存管理」中的儲存。匯入時請同時選擇 `lucky_storagemanagement.lkcf`（或直接匯入整站 zip），本機儲存會解析為對應目錄；WebDAV、阿里雲盤儲存與 rclone 掛載暫不支援，會在報告中提示。
- **尚未實作的 Lucky 功能**：DLNA、rclone 網路硬碟掛載、阿里雲盤、IP 庫（`lucky_ipdb.lkcf`）等暫未遷移，匯入時會略過。
- **Lucky 官方伺服器**：Gately 不會連線 `66666.host`、`lucky666.cn` 等位址，遷移後公網 IP 查詢使用第三方公共服務（可在 DDNS 工作中修改）。

## 7. 退回

Gately 不會修改 Lucky 的任何檔案。如需退回，停用 Gately 中對應的規則，再在 Lucky 中重新啟用即可。
