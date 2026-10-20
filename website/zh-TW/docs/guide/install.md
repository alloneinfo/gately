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

## 飛牛 fnOS 原生應用 {#fnos}

每個版本附帶飛牛 fnOS 的離線安裝包：`gately_<版本>_fnos_x86.fpk`（Intel / AMD）與 `gately_<版本>_fnos_arm.fpk`（ARM）。安裝包沒有上架官方應用中心，需要手動安裝：

1. 從 [下載頁](/zh-TW/download#fnos) 取得與裝置架構對應的 `.fpk`；
2. 在飛牛開啟「應用中心 → 手動安裝」（左下角按鈕），選擇檔案並完成精靈；
3. 點擊飛牛桌面上的 Gately 圖示開啟管理介面。

### 支援飛牛統一閘道

桌面圖示經飛牛統一閘道開啟管理介面，位址為存取飛牛所用的位址加 `/app/gately/`，與飛牛共用同一個存取入口。區域網路位址、遠端存取飛牛的網域都能直接使用，無需額外開放或對應 16680 連接埠。

經閘道存取時由飛牛負責存取控制，只有飛牛管理員能開啟；安全入口與「僅允許內網存取」只約束直接存取 16680 連接埠。區域網路內仍可直接存取 `http://<NAS 位址>:16680`。

### 支援飛牛管理員登入狀態直接登入

在「設定 → 存取控制」開啟「飛牛登入狀態自動登入」後，已登入飛牛的管理員點擊桌面圖示即可直接進入 Gately，無需再輸入密碼與兩步驟驗證碼：

- 只有飛牛管理員會以 Gately 管理員身分登入，一般飛牛使用者仍需輸入帳號密碼；
- 只對經統一閘道的存取生效，直接存取 16680 連接埠時仍需正常登入；
- 主動登出後不會靜默重新登入，登入頁會顯示「使用飛牛帳號 &lt;使用者名稱&gt; 登入」按鈕；
- 該開關預設關閉，只在飛牛安裝包中顯示。

### 執行說明

- 應用以 `root` 身分執行，可以監聽 80/443、透過 `docker.sock` 管理 Docker、使用真實的 Web 終端機。請只安裝在自己掌控的 NAS 上。
- 首次啟動時 `admin` 的初始密碼寫在 `/var/apps/gately/var/gately.log`：`grep -i "initial password" /var/apps/gately/var/gately.log`。
- 設定儲存在應用的 `var` 目錄，解除安裝時預設保留；在解除安裝精靈中選擇「全部刪除」才會清除。
- 安裝包同時提供命令列入口 `/usr/local/bin/gately`，可使用 `gately ctl` 與 `gately mcp`。
- 16680 連接埠與 Docker 映像檔相同，啟動原生應用前請先停止 `gately` 容器。

## 初始登入

管理介面預設監聽 `16680` 連接埠，使用者名稱 `admin`，初始密碼會在首次啟動時輸出到日誌。自動化部署時可用環境變數 `GATELY_INITIAL_PASSWORD` 指定初始密碼，此時不會產生隨機密碼。

首次登入會出現啟動引導，依序提示修改密碼、開啟兩步驟驗證或 Passkey、執行網路自檢、從 Lucky 匯入與設定通知。

## 升級

Gately 不內建線上自動更新，升級由部署方式負責：

- Docker：`docker pull` 新映像檔後重建容器（設定目錄維持掛載即可）；
- 執行檔：替換可執行檔並重新啟動服務；
- 飛牛 fnOS：在應用中心手動安裝新版 `.fpk`，設定自動保留。

升級前建議匯出一份設定備份，或直接把設定目錄納入 git。下載的壓縮檔與映像檔可依 [驗證發佈檔案](/en/docs/guide/verify) 核對簽章。
