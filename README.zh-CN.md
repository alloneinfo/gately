# Gately

[English](README.md) | **简体中文** | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md) | [한국어](README.ko.md) | [Español](README.es.md) | [Français](README.fr.md) | [Deutsch](README.de.md) | [Русский](README.ru.md) | [Português (Brasil)](README.pt-BR.md)

开源、可审计的网络网关工具集，家庭网络、NAS、云主机与各类服务器均可使用，灵感来自 [Lucky](https://github.com/gdy666/lucky)，以完全开源的方式提供同类能力。DDNS、STUN 穿透、端口转发、反向代理、证书、WAF 等常用能力集成在一个约 22 MB 的单文件中，并可一键导入 Lucky 配置。

## 为什么做 Gately

Lucky 是国内 NAS 与家庭网络用户熟悉的优秀工具，把 DDNS、STUN 穿透、反向代理等能力整合在一起，也为 Gately 提供了产品思路，没有 Lucky 就不会有 Gately。Lucky 以闭源方式免费分发，Gately 选择完全开源，让用户和社区可以直接审阅源码与配置。对于运行在网络边界、手握 DNS 密钥和证书私钥的软件，开源让用户能够看到它在做什么。Gately 的承诺：

- 源码以 Apache-2.0 开源，接受社区审查
- 默认不连接任何项目方服务器，不上报设备信息；第三方登录使用你自己的 OAuth 应用，不经过中转
- 所有对外地址（公网 IP 查询、STUN 服务器等）可见、可改、可关
- 配置为明文 JSON，可直接纳入 git 做审计和回滚

## 功能

| 模块 | 能力 |
| --- | --- |
| 动态域名 | 阿里云、Cloudflare、腾讯云、DNSPod、华为云、GoDaddy、Porkbun、dynv6、DuckDNS、自定义回调；URL/网卡/命令取 IP；IPv4/IPv6；Webhook |
| STUN 穿透 | TCP/UDP、保活、UPnP/NAT-PMP、内置转发、地址变化时触发 Webhook 与脚本 |
| 端口转发 | TCP/UDP、IPv4/IPv6、端口段、多目标故障转移、连接数限制、流量统计 |
| SSL 证书 | ACME DNS-01 自动签发与续期（通配符、EAB）、上传、读取文件、映射到目录并执行脚本 |
| Tailscale | 托管 tailscaled（用户态网络，容器友好）或接入系统已有守护进程；认证密钥/登录链接、Headscale、子网路由、出口节点、节点列表与 SOCKS5 代理、固定端口、tailscaled 额外参数与环境变量 |
| frp | frpc 客户端（tcp/udp/http/https/stcp/xtcp 代理，增删即生效）与 frps 服务端（端口白名单、子域名、控制台），可一键导入旧版 ini / 新版 toml 配置 |
| Cloudflare 隧道 | cloudflared 令牌隧道与 trycloudflare 临时隧道，多隧道、自动重启、公网地址展示 |
| Web 服务 | 按域名反向代理/重定向/静态文件/固定文本；HTTPS；WebSocket；PROXY 协议收发；基本认证、网页认证、GitHub/Google/QQ/OIDC 登录；IP 黑白名单、路径规则、临时分享链接、认证失败自动封禁、GeoIP 国家过滤；后端权重/备用/负载均衡策略、主动与被动健康检查、压缩、限流、响应缓存、路径重写；分钟级访问统计与 Prometheus 指标 |
| WAF 防火墙 | Coraza + OWASP CRS v4，可按文件启用规则、自定义 SecLang、拦截/仅检测 |
| Docker 管理 | 容器创建与编辑、日志与终端、Compose、镜像更新检查与一键升级（失败自动回滚）、网络与卷 |
| Web 终端 | SSH（主机密钥首次信任）、Telnet、本机 Shell（默认关闭）；代理、持久会话、SFTP 文件管理、2FA 与协议确认、快捷命令、分屏 |
| 网络诊断 | Ping、路由追踪、端口探测、DNS 查询、测速、局域网扫描 |
| 通知中心 | Webhook、Telegram、Bark、企业微信、钉钉、飞书、邮件；级别、通配与静默去重规则 |
| 配置备份 | 加密备份（scrypt + AES-GCM）、保留策略、定时备份到本地目录或存储挂载点 |
| 命令行与 MCP | `gately ctl` 调用运行中的实例，`gately mcp` 以 stdio 提供 MCP 服务（默认只读） |
| 网络唤醒 | 魔术包广播、在线探测 |
| WebDAV | 多用户、独立目录、只读 |
| FTP | 多用户、独立目录、只读，支持 FTPS 与被动端口范围 |
| 存储管理 | 本地目录与远程 WebDAV 挂载，在线文件管理 |
| 计划任务 | cron 表达式，命令或 HTTP 请求，执行记录 |
| 安全与设置 | 两步验证（TOTP）、登录失败限流、安全入口、仅内网访问、管理界面 HTTPS、多 API 令牌（按模块读写授权）、自定义 DNS（支持 DoH）、时区、配置备份与恢复 |
| 迁移 | 导入 Lucky 的 `.lkcf` 或整站配置 zip：端口转发、DDNS、STUN、证书、反向代理、WAF、网络唤醒、Web 终端、WebDAV、基础设置 |

## 快速开始

### Docker

镜像支持 amd64、arm64、arm/v7：

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep -i "initial password"
```

打开 `http://<设备IP>:16680`，用 `admin` 和日志中的初始密码登录，然后在「设置」中修改密码并开启两步验证。完整的 Compose 示例见 [deploy/docker-compose.yml](deploy/docker-compose.yml)。

端口转发、STUN、Web 服务需要监听宿主机端口，推荐 `--network host`。不需要 Docker 管理时可去掉 `docker.sock` 挂载。

### 二进制

从 [Releases](https://github.com/wangwanjie/gately/releases) 下载对应平台的压缩包（Linux amd64/arm64/arm/mips/mipsle/riscv64、macOS、Windows、FreeBSD），解压后运行：

```bash
./gately -c /etc/gately
```

参数：`-c` 配置目录（也可用环境变量 `GATELY_CONFIG_DIR`），`-listen` 覆盖管理界面监听地址，`-version` 输出版本。

### API 调用

在管理后台“API 令牌”页创建令牌（可建多个，按模块分别授予只读或读写，可设过期时间，创建后明文只显示一次），脚本即可批量调用管理接口：

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks
```

账号、两步验证、令牌管理以及备份、恢复、重启、系统设置接口只能使用登录会话。全部接口、所需权限与数据结构见管理后台“API 文档”页，也可以用 `make api-docs` 导出到 [docs/api](docs/api)（`openapi.json` 与 `API.md`），官网 https://gately.cn 发布时直接引用。

从旧版升级时，原来的全局 API 令牌会自动变成一条“遗留令牌”（值不变，拥有全部模块权限），建议在 API 令牌页收紧权限或换成新令牌。

### Tailscale、frp、cloudflared

这三个模块托管官方的外部程序（自动重启、日志汇总到界面），Gately 本体不内置它们，因此始终可以使用上游的最新版本。程序按以下顺序查找：模块设置中填写的路径、`PATH`、配置目录下的 `bin/` 子目录。Docker 部署时把二进制放进挂载的配置目录即可，例如 `./gately-data/bin/frpc`；Tailscale 需要同时提供 `tailscale` 与 `tailscaled`。配置文件中的令牌与密钥以 0600 权限保存，cloudflared 的令牌通过环境变量传递，不会出现在进程列表中。

### 从 Lucky 迁移

见 [迁移指南](docs/MIGRATION.md)。简要步骤：在「设置 → 从 Lucky 导入」选择 Lucky 配置目录中的 `lucky_*.lkcf` 文件，预览后导入；导入的规则默认停用，按模块逐个在 Lucky 中停用、在 Gately 中启用。

## 开发

依赖：Go 1.26+、Node.js 24+、pnpm。

```bash
make web      # 构建前端（输出到 web/dist，由 go:embed 嵌入）
make build    # 编译 bin/gately
make test     # go test -race ./...
make lint     # gofmt、go vet、前端类型检查
```

前端开发时运行 `cd web && pnpm dev`，`/api` 会代理到本地 `127.0.0.1:16680`。

一些测试默认跳过，需要额外条件：

| 测试 | 启用方式 |
| --- | --- |
| ACME 端到端（Pebble） | 安装 pebble 与 pebble-challtestsrv，设置 `GATELY_PEBBLE_BIN`、`GATELY_PEBBLE_SRC`（CI 中自动运行） |
| 真实公共 STUN 服务器 | `GATELY_NET_TEST=1` |
| 用真实 Lucky 配置验证导入 | `GATELY_LUCKY_SAMPLE` 等，见各模块 `TestImportRealSample` |

目录结构与技术选型见 [docs/ROADMAP.md](docs/ROADMAP.md)。

官网（[gately.cn](https://gately.cn)）源码在 `website/`，基于 VitePress：`cd website && pnpm install && pnpm dev`；`pnpm deploy:site` 构建并同步到服务器。

## 贡献约定

- 与 Lucky 的兼容基于公开可观察的行为、接口和配置格式。请勿提交从 Lucky 二进制或前端产物中复制的代码。
- 新模块需要：配置结构与校验、运行时启停、REST 接口、单元测试；能从 Lucky 迁移的模块实现 `luckyimport.Importer`，希望在总览页展示状态的模块实现 `module.Summarizer`。
- 代码注释使用中文；控制语句必须使用大括号。

## Star 趋势

[![Star History Chart](https://api.star-history.com/svg?repos=wangwanjie/gately&type=Date)](https://star-history.com/#wangwanjie/gately&Date)

## 许可证

[Apache License 2.0](LICENSE)
