# 快速開始

五分鐘內完成部署、登入與第一條規則。

## 1. 啟動

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep -i "initial password"
```

不需要 Docker 管理功能時，可以移除 `docker.sock` 掛載。連接埠轉發、STUN、Web 服務需要監聽主機的連接埠，建議使用 `--network host`。

## 2. 登入

瀏覽器開啟 `http://<裝置IP>:16680`，使用者名稱 `admin`，密碼為日誌中輸出的初始密碼（僅在首次啟動時列印一次）。

## 3. 完成基礎強化

登入後立即在「設定」中：

1. 修改管理員密碼；
2. 開啟兩步驟驗證（TOTP）；
3. 視需求設定安全入口、僅限內網存取或管理介面 HTTPS。

更多建議請見 [強化建議](/zh-TW/docs/guide/hardening)。

## 4. 新增第一條規則

依需求選擇一個模組開始：

| 需求 | 模組 |
| --- | --- |
| 家裡的公網 IP 經常變動，需要固定網域存取 | [動態網域](/zh-TW/docs/modules/ddns) |
| 電信業者沒有提供固定公網連接埠，想從外網存取家裡的服務 | [STUN 穿透](/zh-TW/docs/modules/stun) |
| 把一個連接埠轉發到內網另一台裝置 | [連接埠轉發](/zh-TW/docs/modules/portforward) |
| 用一個網域和 HTTPS 存取多個內網服務 | [SSL 憑證](/zh-TW/docs/modules/ssl) + [Web 服務](/zh-TW/docs/modules/web) |
| 異地組網 | [Tailscale / frp](/zh-TW/docs/modules/tunnels) |

## 從 Lucky 遷移

已經在使用 Lucky 時，不必重新設定：在「設定 → 備份與遷移」匯入 `.lkcf` 或整站設定 zip 即可，詳見 [遷移指南](/zh-TW/docs/guide/migration)。
