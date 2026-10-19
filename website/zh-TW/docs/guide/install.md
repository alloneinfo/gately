# 安裝部署

## Docker

映像檔位址 `ghcr.io/wangwanjie/gately`，支援 amd64、arm64、arm/v7。

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
    # 連接埠轉發、STUN 穿透、Web 服務都需要直接監聽主機的連接埠
    network_mode: host
    volumes:
      # 設定目錄（純文字 JSON，可納入 git 備份；ssl.json 中含憑證私鑰）
      - ./data:/etc/gately
      # Docker 管理（不需要可刪除）
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TZ=Asia/Taipei
```

映像檔內建 `sh`、`bash` 與 `curl`，常見的通知指令碼可以直接執行。

## 執行檔

1. 從 [下載頁](/zh-TW/download) 取得對應平台的壓縮檔：Linux（amd64 / arm64 / arm / mips / mipsle / riscv64）、macOS、Windows、FreeBSD。
2. 解壓縮後執行：

```bash
./gately -c /etc/gately
```

### systemd

儲存庫提供完整的服務檔 [deploy/systemd/gately.service](https://github.com/wangwanjie/gately/blob/main/deploy/systemd/gately.service)，已授予監聽低位連接埠所需的能力：

```bash
sudo install -m 0755 gately /usr/local/bin/gately
sudo install -m 0644 gately.service /etc/systemd/system/gately.service
sudo systemctl daemon-reload && sudo systemctl enable --now gately
journalctl -u gately | grep -i "initial password"
```

### OpenWrt

使用 procd 啟動指令碼 [deploy/openwrt/gately.init](https://github.com/wangwanjie/gately/blob/main/deploy/openwrt/gately.init)：

```bash
cp gately /usr/bin/gately && chmod +x /usr/bin/gately
cp gately.init /etc/init.d/gately && chmod +x /etc/init.d/gately
/etc/init.d/gately enable && /etc/init.d/gately start
```

## 初始登入

管理介面預設監聽 `16680` 連接埠，使用者名稱 `admin`，初始密碼會在首次啟動時輸出到日誌。自動化部署時可用環境變數 `GATELY_INITIAL_PASSWORD` 指定初始密碼，此時不會產生隨機密碼。

首次登入會出現啟動引導，依序提示修改密碼、開啟兩步驟驗證或 Passkey、執行網路自檢、從 Lucky 匯入與設定通知。

## 升級

Gately 不內建線上自動更新，升級由部署方式負責：

- Docker：`docker pull` 新映像檔後重建容器（設定目錄維持掛載即可）；
- 執行檔：替換可執行檔並重新啟動服務。

升級前建議匯出一份設定備份，或直接把設定目錄納入 git。下載的壓縮檔與映像檔可依 [驗證發佈檔案](/en/docs/guide/verify) 核對簽章。
