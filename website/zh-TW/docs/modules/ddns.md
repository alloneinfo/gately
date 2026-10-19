# 動態網域（DDNS）

家用寬頻的公網 IP 會變動，DDNS 在 IP 變動後自動更新 DNS 解析，讓網域始終指向目前的位址。

## 支援的服務商

阿里雲（Alibaba Cloud）、Cloudflare、騰訊雲、DNSPod、華為雲、GoDaddy、Porkbun、dynv6、DuckDNS、Amazon Route 53、OVHcloud、Gandi、NS1、Bunny DNS、Namecheap、DynDNS2，以及自訂回呼（用於未內建的服務商或自建介面）。

| 服務商 | 憑據 | 可用於憑證 DNS 驗證 |
| --- | --- | --- |
| Amazon Route 53 | Access Key ID / Secret Access Key，可選託管區域 ID | 是 |
| OVHcloud | Application Key / Secret 與 Consumer Key，接入點 ovh-eu / ovh-ca / ovh-us | 是 |
| Gandi | Personal Access Token | 是 |
| NS1 | API Key | 是 |
| Bunny DNS | Account API Key | 是 |
| Namecheap | 網域的 Dynamic DNS 密碼（僅 A 記錄） | 否 |
| DynDNS2 | 使用者名稱、密碼與更新網址，適用於 No-IP、Dynu 等相容 dyndns2 協定的服務 | 否 |

## 取得 IP 的方式

- **URL**：請求公網 IP 查詢位址（可自行替換為任意服務，或自建）；
- **網卡**：直接讀取指定網卡的位址，適合擁有公網 IP 的裝置；
- **命令**：執行命令並取其輸出，適合特殊網路環境。

支援 IPv4 與 IPv6。

## 通知

IP 變動或同步失敗時可呼叫 Webhook。

## 使用建議

- 為 DNS 服務商建立僅具備解析編輯權限的子帳號金鑰，不要使用主帳號金鑰；
- 新增工作後點擊「強制同步」確認記錄已更新；
- 公網 IP 查詢位址不再依賴任何專案方服務，可以在工作中隨時更換。

## 從 Lucky 遷移

DDNS 工作可從 `lucky_ddns.lkcf` 匯入，支援 11 家服務商。Lucky 官方 IP 查詢位址會在匯入時被移除。詳見 [遷移指南](/zh-TW/docs/guide/migration)。
