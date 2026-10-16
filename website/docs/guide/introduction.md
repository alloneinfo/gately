# 项目简介

Gately 是开源、可审计的网络网关工具集，家庭网络、NAS、云主机与各类服务器均可使用，定位为 [Lucky](https://github.com/gdy666/lucky) 的开源替代。DDNS、STUN 穿透、端口转发、反向代理、证书、WAF 等常用能力集成在一个约 22 MB 的单文件中，并可一键导入 Lucky 配置。

## 为什么做 Gately

Lucky 功能强大，但闭源、二进制加壳，配置文件经过混淆，并包含与官方服务器通信的逻辑。对于运行在网络边界、手握 DNS 密钥和证书私钥的软件，用户应当能看到它在做什么。Gately 的承诺：

- 源码以 Apache-2.0 开源，接受社区审查
- 默认不连接任何项目方服务器，不上报设备信息；第三方登录使用你自己的 OAuth 应用，不经过中转
- 所有对外地址（公网 IP 查询、STUN 服务器等）可见、可改、可关
- 配置为明文 JSON，可直接纳入 git 做审计和回滚

## 设计原则

1. **透明**：默认不连接任何项目方服务器，所有对外地址可见、可改、可关。
2. **可迁移**：配置字段尽量与 Lucky 对齐，提供 `.lkcf` 导入。
3. **可审计的配置**：配置为明文 JSON（文件权限 0600），可直接纳入 git 做 diff 和回滚。
4. **单文件部署**：一个静态二进制内嵌前端，支持 Docker、systemd、OpenWrt。

## 技术栈

| 层 | 选型 |
| --- | --- |
| 后端 | Go，静态编译，可交叉编译到 arm / arm64 / mips / riscv 路由器与 NAS |
| HTTP | 标准库 `net/http` |
| 存储 | 每个模块一个明文 JSON 文件，原子写入 |
| 认证 | bcrypt 密码 + HMAC 签名会话令牌，登录失败限流，TOTP 两步验证 |
| 前端 | Vue 3 + TypeScript + Vite + Element Plus，通过 `go:embed` 打包进二进制 |

## 功能一览

动态域名、STUN 穿透、端口转发、SSL 证书、Web 服务（反向代理）、WAF 防火墙、Tailscale、frp、Cloudflare 隧道、Docker 管理、Web 终端、网络唤醒、WebDAV、FTP、存储管理、计划任务，以及两步验证、安全入口、API 令牌等安全设置。详见 [模块总览](/docs/modules/)。

## 下一步

- 想马上试用：[快速开始](/docs/guide/quick-start)
- 正在使用 Lucky：[迁移指南](/docs/guide/migration)
- 关心隐私与安全：[安全模型与隐私承诺](/docs/guide/security)
