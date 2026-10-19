# SSL 憑證

自動申請並續期憑證，供 [Web 服務](/zh-TW/docs/modules/web) 使用，也可以匯出給其他軟體。

## 能力

- **ACME DNS-01** 自動簽發與續期，重用 [動態網域](/zh-TW/docs/modules/ddns) 中的 DNS 服務商，無需開放 80 / 443 連接埠；
- **ACME HTTP-01**：不需要 DNS 服務商，要求公網能存取各網域的 80 連接埠，不支援萬用字元；
- 萬用字元憑證與 EAB（如 ZeroSSL）；
- 上傳既有憑證，或從檔案讀取；
- 對應到目錄：憑證變動後寫入指定目錄並執行指令碼，方便其他服務重新載入。

## HTTP-01 驗證

在憑證的「驗證方式」中選擇 HTTP-01。簽發時 CA 會存取 `http://<網域>/.well-known/acme-challenge/<token>`：

- [Web 服務](/zh-TW/docs/modules/web) 已在 80 連接埠監聽時，由它直接回應，不影響現有站台；
- 80 連接埠閒置時，Gately 會在簽發期間臨時監聽該連接埠，完成後立即關閉；
- 路由器把公網 80 連接埠轉發到本機其他連接埠時，在「臨時監聽埠」中填寫該連接埠。

許多家用寬頻（例如中國大陸）會封鎖入站 80 連接埠，此時請使用 DNS-01。

## 憑證對應

啟用憑證後，Gately 會把 `.crt`、`.key`、`.pem` 與 `_issuerCertificate.crt` 寫入對應目錄，並執行變更指令碼。

## 安全提示

私鑰儲存在設定目錄的 `ssl.json` 中，請妥善保管該檔案，納入版本庫時確認儲存庫為私有。

## 從 Lucky 遷移

已簽發的憑證可原樣匯入，到期前不會重新申請。ZeroSSL 沿用 Lucky 中的 EAB 憑據註冊新的 ACME 帳號。詳見 [遷移指南](/zh-TW/docs/guide/migration)。
