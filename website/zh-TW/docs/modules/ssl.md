# SSL 憑證

自動申請並續期憑證，供 [Web 服務](/zh-TW/docs/modules/web) 使用，也可以匯出給其他軟體。

## 能力

- **ACME DNS-01** 自動簽發與續期，重用 [動態網域](/zh-TW/docs/modules/ddns) 中的 DNS 服務商，無需開放 80 / 443 連接埠；
- 萬用字元憑證與 EAB（如 ZeroSSL）；
- 上傳既有憑證，或從檔案讀取；
- 對應到目錄：憑證變動後寫入指定目錄並執行指令碼，方便其他服務重新載入。

## 憑證對應

啟用憑證後，Gately 會把 `.crt`、`.key`、`.pem` 與 `_issuerCertificate.crt` 寫入對應目錄，並執行變更指令碼。

## 安全提示

私鑰儲存在設定目錄的 `ssl.json` 中，請妥善保管該檔案，納入版本庫時確認儲存庫為私有。

## 從 Lucky 遷移

已簽發的憑證可原樣匯入，到期前不會重新申請。ZeroSSL 沿用 Lucky 中的 EAB 憑據註冊新的 ACME 帳號。詳見 [遷移指南](/zh-TW/docs/guide/migration)。
