# 与 Lucky 对比

Gately 是 Lucky 的开源替代，覆盖常用能力，并可导入现有配置。

## 理念差异

| 项目 | Gately | Lucky |
| --- | --- | --- |
| 源码 | 开源（Apache-2.0） | 闭源 |
| 二进制 | 未加壳 | 加壳 |
| 配置文件 | 明文 JSON，可 git 管理 | 混淆格式（`.lkcf`） |
| 官方服务器通信 | 默认无，所有对外地址可改可关 | 包含与官方服务器通信的逻辑 |
| 第三方登录 | 使用自己的 OAuth 应用 | 经官方中转服务器 |
| 升级方式 | Docker / 包管理器 | 内置在线自更新 |

Lucky 的分析结论见仓库中的 [逆向分析报告](https://github.com/wangwanjie/gately/blob/main/docs/analysis/lucky-2.27.2.md)。

## 功能覆盖

| 能力 | Gately |
| --- | --- |
| 动态域名 | ✅ |
| STUN 穿透 | ✅ |
| 端口转发 | ✅ |
| SSL 证书 | ✅ |
| Web 服务（反向代理） | ✅ |
| WAF | ✅ |
| Tailscale / frp / cloudflared | ✅ |
| Docker 管理 | ✅ |
| Web 终端 | ✅ |
| WebDAV / FTP / 存储管理 | ✅ |
| 网络唤醒 / 计划任务 | ✅ |
| DLNA、网盘挂载、rclone、全局 IP 过滤、HTTP/3 | 规划中，视社区需求排期 |

## 有意为之的差异

- 不内置在线自更新覆盖二进制，由包管理器或 Docker 负责升级。
- 不下发远程配置，不上报设备信息。
- 配置明文存储，敏感字段在 API 输出中脱敏。

## 迁移

两者可以并存，逐个模块切换，随时回退。见 [迁移指南](/docs/guide/migration)。
