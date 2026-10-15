# Gately 技术方案与路线图

## 目标

做一个开源、可审计、社区驱动的 Lucky 替代品：覆盖家庭网络/NAS 用户最常用的 DDNS、STUN 穿透、端口转发、反向代理、证书管理，并能一键导入现有 Lucky 配置。

设计原则：

1. **透明**：默认不连接任何项目方服务器；所有对外地址可见、可改、可关。
2. **可迁移**：配置字段尽量与 Lucky 对齐，提供 `.lkcf` 导入。
3. **可审计的配置**：配置为明文 JSON（文件权限 0600），可直接纳入 git 做 diff 和回滚。
4. **单文件部署**：一个静态二进制内嵌前端，支持 Docker、systemd、OpenWrt。

## 技术栈

| 层 | 选型 | 理由 |
| --- | --- | --- |
| 后端 | Go 1.26+ | 与 Lucky 同生态，可直接复用 lego、pion/stun、frp、cloudflared、coraza、rclone 等成熟库；静态编译，交叉编译到 arm/arm64/mips/riscv 路由器和 NAS |
| HTTP | 标准库 `net/http`（Go 1.22 路由语法） | 依赖最少，便于审计 |
| 存储 | 每模块一个明文 JSON 文件，原子写入 | 与 Lucky 的模块化配置对应，便于导入、备份和 git diff |
| 认证 | bcrypt 密码 + HMAC 签名会话令牌，登录失败限流，后续支持 TOTP | 无外部依赖 |
| 日志 | 每模块内存环形缓冲 + 增量拉取接口 | 对应 Lucky 的 `logs/lastlogs` 交互 |
| 前端 | Vue 3 + TypeScript + Vite + Element Plus，`go:embed` 打包 | 与 Lucky 界面习惯一致，降低迁移成本 |
| 发布 | GoReleaser 多架构二进制 + 多架构 Docker 镜像 | — |

## 目录结构

```
cmd/gately/            程序入口
internal/app/          组装：加载配置、启动模块、HTTP 服务
internal/config/       模块化 JSON 配置存储
internal/auth/         管理员认证与会话
internal/logbuf/       模块日志环形缓冲
internal/httpx/        HTTP 工具与中间件
internal/module/       模块接口与注册表
internal/modules/*/    各功能模块
internal/luckyimport/  Lucky .lkcf 解密与字段映射
web/                   前端工程，构建产物由 web/embed.go 嵌入
docs/analysis/         Lucky 逆向分析结论
```

## 里程碑

按用户实际使用规模排序（STUN 18、端口转发 25、反代 27、证书 5、DDNS 75 条记录）。

| 阶段 | 内容 | 验收 |
| --- | --- | --- |
| ✅ **M0 骨架** | 配置存储、管理员登录、模块框架、日志缓冲、管理 UI 外壳、Dockerfile | 登录后能看到总览与模块菜单 |
| ✅ **M1 端口转发** | TCP/UDP、v4/v6、多端口与端口段、多目标、连接数限制、启停与日志；导入 Lucky 端口转发规则 | 单元测试覆盖 TCP/UDP 转发；真实样本 25/25 条导入通过 |
| ✅ **M2 DDNS** | 取 IP（URL/网卡/命令）、IPv4/IPv6、阿里云/Cloudflare/DNSPod(腾讯云)/华为云/自定义回调、Webhook；导入 | 11 家服务商（阿里云签名经官方示例校验）；真实样本 2/2 任务、75 条记录导入通过 |
| **M3 STUN 穿透** | STUN 获取公网映射、TCP 保活、UPnP/NAT-PMP、内置转发、Webhook 与脚本变量；导入 | 本地 STUN 服务端测试 |
| **M4 证书** | ACME（lego，DNS-01 复用 M2 Provider）、自动续期、导出到路径、变更脚本 | Pebble 测试环境 |
| **M5 Web 服务** | 反向代理、重定向、静态文件、按域名路由、自动 TLS、BasicAuth、IP 过滤；导入 | httptest 覆盖路由与鉴权 |
| M6 运维 | 网络唤醒、计划任务、IP 过滤、Web 终端、TOTP | — |
| M7 生态 | frpc、cloudflared、Coraza WAF、WebDAV/FTP/DLNA、Docker 管理 | 按社区需求排期 |

## 与 Lucky 的差异（有意为之）

- 不内置在线自更新覆盖二进制，由包管理器/Docker 负责升级。
- 不下发远程配置，不上报设备信息。
- 配置明文存储；敏感字段在 API 输出中脱敏。
