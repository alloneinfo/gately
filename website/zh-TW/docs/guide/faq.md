# 常見問題

## 需要連線到專案方的伺服器嗎？

不需要。Gately 預設不連線任何專案方伺服器，也不回報裝置資訊。公網 IP 查詢等對外位址使用第三方公共服務，可以在對應工作中修改。

## 初始密碼在哪裡？

首次啟動時輸出在日誌中，僅列印一次。Docker 部署請執行 `docker logs gately 2>&1 | grep -i "initial password"`。

## 忘記管理員密碼怎麼辦？

停止 Gately，刪除設定目錄中的 `account.json`，再啟動：程式會重新建立 `admin` 帳號，並把新的初始密碼列印到日誌。登入後請檢查兩步驟驗證等安全設定。

## 支援飛牛 fnOS 嗎？

支援。每個版本附帶飛牛離線安裝包（`.fpk`），在應用中心手動安裝。它接入飛牛統一閘道，點擊桌面圖示即可開啟，遠端存取飛牛的網域同樣可用；在「設定 → 存取控制」開啟「飛牛登入狀態自動登入」後，已登入的飛牛管理員可直接進入，無需再輸入密碼與兩步驟驗證碼。詳見 [飛牛 fnOS 原生應用](/zh-TW/docs/guide/install#fnos)。

## 支援 OpenWrt / iStoreOS 嗎？

支援。每個版本附帶 OpenWrt 安裝套件：OpenWrt 24.10 及以前（含 iStoreOS）用 `.ipk`，執行 `opkg install` 安裝；OpenWrt 25.12 起用 `.apk`，執行 `apk add --allow-untrusted` 安裝。安裝後自動啟動，設定儲存在 `/etc/gately`，升級時安裝新版安裝套件即可。程式約 35 MB，適合儲存空間較大的軟路由。詳見 [OpenWrt](/zh-TW/docs/guide/install#openwrt)。

## 為什麼建議使用 `--network host`？

連接埠轉發、STUN、Web 服務需要直接監聽主機的連接埠。使用橋接網路時，需要自行對應每一個用到的連接埠。

## 可以和 Lucky 同時執行嗎？

可以。管理連接埠預設為 16680，與 Lucky 的 16601 互不衝突。匯入的規則預設停用，避免與仍在執行的 Lucky 爭用連接埠和路由器 UPnP 對應。

## Tailscale、frp、cloudflared 為什麼要自己放執行檔？

這三個模組託管官方外部程式，Gately 本體並未內建它們，因此始終可以使用上游最新版本。放置位置請見 [設定與執行參數](/zh-TW/docs/guide/configuration)。

## STUN 規則啟用後外網仍無法存取？

路由器 WAN 口位於電信級 NAT（100.64.0.0/10）時，外部無法存取穿透連接埠，這是網路環境本身的限制。可改用 [Tailscale / frp / Cloudflare 通道](/zh-TW/docs/modules/tunnels)。

## 如何回報問題或參與開發？

在 [GitHub Issues](https://github.com/wangwanjie/gately/issues) 提交問題，開發約定請見 [參與貢獻](/zh-TW/docs/guide/contributing)。
