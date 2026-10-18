# 參與貢獻

Gately 是社群驅動的開源專案，歡迎提交 Issue 與 Pull Request。儲存庫：<https://github.com/wangwanjie/gately>

## 本機開發

需求：Go 1.26+、Node.js 24+、pnpm。

```bash
make web      # 建置前端（輸出到 web/dist，由 go:embed 嵌入）
make build    # 編譯 bin/gately
make test     # go test -race ./...
make lint     # gofmt、go vet、前端型別檢查
```

前端開發時執行 `cd web && pnpm dev`，`/api` 會代理到本機 `127.0.0.1:16680`。

## 貢獻約定

- 相容性程式碼只能建立在公開可觀察的行為、介面和設定格式上，請勿提交從第三方執行檔或前端產物中複製的程式碼。
- 新模組需要：設定結構與驗證、執行期啟停、REST 介面、單元測試；能從 Lucky 遷移的模組實作 `luckyimport.Importer`，希望在總覽頁顯示狀態的模組實作 `module.Summarizer`。
- 程式碼註解使用中文；控制語句必須使用大括號。

## 額外測試

部分測試預設略過，需要額外條件：

| 測試 | 啟用方式 |
| --- | --- |
| ACME 端對端（Pebble） | 安裝 pebble 與 pebble-challtestsrv，設定 `GATELY_PEBBLE_BIN`、`GATELY_PEBBLE_SRC` |
| 真實公共 STUN 伺服器 | `GATELY_NET_TEST=1` |
| 用真實 Lucky 設定驗證匯入 | `GATELY_LUCKY_SAMPLE` 等，見各模組 `TestImportRealSample` |

## 授權條款

[Apache License 2.0](https://github.com/wangwanjie/gately/blob/main/LICENSE)
