# Tailscale / frp / cloudflared

三個模組以行程託管方式整合官方程式：自動重啟、日誌彙整到介面、設定在介面中完成。Gately 本體並未內建這些程式，因此始終可以使用上游最新版本。

## 程式放置

依以下順序尋找可執行檔：

1. 模組設定中填寫的路徑；
2. `PATH`；
3. 設定目錄下的 `bin/` 子目錄。

Docker 部署時把執行檔放進掛載的設定目錄即可，例如 `./gately-data/bin/frpc`。Tailscale 需要同時提供 `tailscale` 與 `tailscaled`。

權杖與金鑰以 0600 權限儲存，cloudflared 的權杖透過環境變數傳遞，不會出現在行程清單中。

## Tailscale

- 託管 `tailscaled`（使用者空間網路，對容器友善）或接入系統既有的常駐程式；
- 驗證金鑰或登入連結，支援 Headscale；
- 子網路由與出口節點；
- 節點清單與 SOCKS5 代理；
- 固定 WireGuard UDP 連接埠，方便在防火牆放行直連；
- tailscaled 額外參數與環境變數。主機 iptables 是 nf_tables 後端時，在環境變數中加入 `TS_DEBUG_FIREWALL_MODE=nftables`，子網轉發才會生效。

## frp

- **frpc 用戶端**：tcp / udp / http / https / stcp / xtcp 代理，新增刪除即生效；
- **frps 伺服器端**：連接埠白名單、子網域、控制台；
- **匯入既有設定**：頁首的「匯入設定」支援舊版 `frpc.ini` / `frps.ini`（含 `[common]` 段）和 frp 0.52+ 的 `toml`，可選擇檔案或貼上內容。先預覽解析結果與被忽略的欄位，再確認匯入；用戶端可選擇附加（同名代理保持不變）或取代全部代理。匯入不會改變啟用狀態，也可透過介面 `POST /api/frp/import` 呼叫。
- **從 Lucky 遷移**：`lucky_frp.lkcf` 中的第一個用戶端實例與第一個伺服器端實例可直接匯入，見 [從 Lucky 遷移](/zh-TW/docs/guide/migration)。

## Cloudflare 通道

- 權杖通道與 trycloudflare 臨時通道；
- 多通道、自動重啟；
- 顯示公網位址。
- 可從 Lucky 匯入使用通道權杖的通道。

## 如何選擇

| 情境 | 建議 |
| --- | --- |
| 在多台裝置之間組成私有網路 | Tailscale |
| 自己有公網伺服器，把內網服務對應出去 | frp |
| 沒有公網 IP 也沒有伺服器，快速公開 Web 服務 | Cloudflare 通道 |
| 電信業者提供了可打洞的 NAT | [STUN 穿透](/zh-TW/docs/modules/stun) |
