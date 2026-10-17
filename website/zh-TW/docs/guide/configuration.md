# 設定與執行參數

## 命令列參數

| 參數 | 說明 |
| --- | --- |
| `-c` | 設定目錄，也可以用環境變數 `GATELY_CONFIG_DIR` 指定 |
| `-listen` | 覆寫管理介面監聽位址，例如 `:16680` |
| `-version` | 輸出版本資訊 |

## 設定目錄

所有設定都儲存在設定目錄中，每個模組一個 `<name>.json` 純文字檔案，原子寫入，檔案權限 0600。因此：

- 可以直接用文字編輯器檢視；
- 可以把目錄納入 git，用 diff 稽核每次變更並隨時回復；
- 備份只需複製目錄。

::: warning 注意
`ssl.json` 中含有憑證私鑰，納入版本庫前請確認儲存庫為私有，或把該檔案加入忽略清單。
:::

## 外部程式目錄

Tailscale、frp、cloudflared 這三個模組託管官方的外部程式，依以下順序尋找可執行檔：

1. 模組設定中填寫的路徑；
2. `PATH`；
3. 設定目錄下的 `bin/` 子目錄。

Docker 部署時把執行檔放進掛載的設定目錄即可，例如 `./gately-data/bin/frpc`。Tailscale 需要同時提供 `tailscale` 與 `tailscaled`。詳見 [Tailscale / frp / cloudflared](/zh-TW/docs/modules/tunnels)。

## 備份與還原

「設定」中提供設定備份與還原，並承擔從 Lucky 匯入設定的入口。
