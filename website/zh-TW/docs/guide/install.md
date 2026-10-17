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

### systemd 範例

```ini
[Unit]
Description=Gately
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/gately -c /etc/gately
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## 初始登入

管理介面預設監聽 `16680` 連接埠，使用者名稱 `admin`，初始密碼會在首次啟動時輸出到日誌。首次登入後請修改密碼並開啟兩步驟驗證。

## 升級

Gately 不內建線上自動更新，升級由部署方式負責：

- Docker：`docker pull` 新映像檔後重建容器（設定目錄維持掛載即可）；
- 執行檔：替換可執行檔並重新啟動服務。

升級前建議匯出一份設定備份，或直接把設定目錄納入 git。
