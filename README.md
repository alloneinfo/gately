# Gately

开源、可审计的家庭网络网关工具集，定位为 [Lucky](https://github.com/gdy666/lucky) 的开源替代。DDNS、STUN 穿透、端口转发、反向代理、证书、WAF 等常用能力集成在一个约 22 MB 的单文件中，并可一键导入 Lucky 配置。

## 为什么做 Gately

Lucky 功能强大，但闭源、二进制加壳，配置文件经过混淆，并包含与官方服务器通信的逻辑（详见 [逆向分析报告](docs/analysis/lucky-2.27.2.md)）。对于运行在网络边界、手握 DNS 密钥和证书私钥的软件，用户应当能看到它在做什么。Gately 的承诺：

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
| Web 服务 | 按域名反向代理/重定向/静态文件/固定文本；HTTPS；WebSocket；PROXY 协议收发；基本认证、网页认证、GitHub/OIDC 登录；IP 黑白名单 |
| WAF 防火墙 | Coraza + OWASP CRS v4，可按文件启用规则、自定义 SecLang、拦截/仅检测 |
| Docker 管理 | 容器启停、日志、镜像更新检查与一键升级（失败自动回滚） |
| Web 终端 | SSH（主机密钥首次信任）、本机 Shell（默认关闭） |
| 网络唤醒 | 魔术包广播、在线探测 |
| WebDAV | 多用户、独立目录、只读 |
| 计划任务 | cron 表达式，命令或 HTTP 请求，执行记录 |
| 安全 | 管理员两步验证（TOTP）、登录失败限流、密钥脱敏 |
| 迁移 | 导入 Lucky 的端口转发、DDNS、STUN、证书、反向代理、WAF、网络唤醒、Web 终端、WebDAV 配置 |

## 快速开始

### Docker

镜像支持 amd64、arm64、arm/v7：

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep 初始密码
```

打开 `http://<设备IP>:16680`，用 `admin` 和日志中的初始密码登录，然后在「设置」中修改密码并开启两步验证。完整的 Compose 示例见 [deploy/docker-compose.yml](deploy/docker-compose.yml)。

端口转发、STUN、Web 服务需要监听宿主机端口，推荐 `--network host`。不需要 Docker 管理时可去掉 `docker.sock` 挂载。

### 二进制

从 [Releases](https://github.com/wangwanjie/gately/releases) 下载对应平台的压缩包（Linux amd64/arm64/arm/mips/mipsle/riscv64、macOS、Windows、FreeBSD），解压后运行：

```bash
./gately -c /etc/gately
```

参数：`-c` 配置目录（也可用环境变量 `GATELY_CONFIG_DIR`），`-listen` 覆盖管理界面监听地址，`-version` 输出版本。

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

## 贡献约定

- 与 Lucky 的兼容基于公开可观察的行为、接口和配置格式（见 `docs/analysis/`）。请勿提交从 Lucky 二进制或前端产物中复制的代码。
- 新模块需要：配置结构与校验、运行时启停、REST 接口、单元测试；能从 Lucky 迁移的模块实现 `luckyimport.Importer`，希望在总览页展示状态的模块实现 `module.Summarizer`。
- 代码注释使用中文；控制语句必须使用大括号。

## 许可证

[Apache License 2.0](LICENSE)
