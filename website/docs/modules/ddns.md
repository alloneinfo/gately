# 动态域名（DDNS）

家庭宽带的公网 IP 会变化，DDNS 在 IP 变化后自动更新 DNS 解析，让域名始终指向当前地址。

## 支持的服务商

阿里云、Cloudflare、腾讯云、DNSPod、华为云、GoDaddy、Porkbun、dynv6、DuckDNS、Amazon Route 53、OVHcloud、Gandi、NS1、Bunny DNS、Namecheap、DynDNS2，以及自定义回调（用于未内置的服务商或自建接口）。

| 服务商 | 凭据 | 可用于证书 DNS 验证 |
| --- | --- | --- |
| Amazon Route 53 | Access Key ID / Secret Access Key，可选托管区域 ID | 是 |
| OVHcloud | Application Key / Secret 与 Consumer Key，接入点 ovh-eu / ovh-ca / ovh-us | 是 |
| Gandi | Personal Access Token | 是 |
| NS1 | API Key | 是 |
| Bunny DNS | Account API Key | 是 |
| Namecheap | 域名的 Dynamic DNS 密码（仅 A 记录） | 否 |
| DynDNS2 | 用户名、密码与更新地址，适用于 No-IP、Dynu 等兼容 dyndns2 协议的服务 | 否 |

## 获取 IP 的方式

- **URL**：请求公网 IP 查询地址（可自行替换为任意服务，或自建）；
- **网卡**：直接读取指定网卡的地址，适合拥有公网 IP 的设备；
- **命令**：执行命令并取其输出，适合特殊网络环境。

支持 IPv4 与 IPv6。

## 通知

IP 变化或同步失败时可调用 Webhook。

## 使用建议

- 为 DNS 服务商创建仅具备解析编辑权限的子账号密钥，不要使用主账号密钥；
- 新建任务后点击「强制同步」确认记录已更新；
- 公网 IP 查询地址不再依赖任何项目方服务，可以在任务中随时更换。

## 从 Lucky 迁移

DDNS 任务可从 `lucky_ddns.lkcf` 导入，支持 11 家服务商。Lucky 官方 IP 查询地址会在导入时被移除。详见 [迁移指南](/docs/guide/migration)。
