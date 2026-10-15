# Gately

开源、可审计的家庭网络网关工具集，定位为 [Lucky](https://github.com/gdy666/lucky) 的开源替代：端口转发、DDNS、STUN 穿透、反向代理与证书管理，单文件部署，支持一键导入 Lucky 配置。

> 当前状态：早期开发。已完成模块见下表，其余见 [路线图](docs/ROADMAP.md)。

## 为什么做 Gately

Lucky 功能强大，但闭源、二进制加壳，配置文件经过混淆，并包含与官方服务器通信的逻辑（详见 [逆向分析报告](docs/analysis/lucky-2.27.2.md)）。对于运行在网络边界、手握 DNS 密钥和证书私钥的软件，用户应当能看到它在做什么。Gately 的承诺：

- 源码以 Apache-2.0 开源，接受社区审查
- 默认不连接任何项目方服务器，不上报设备信息
- 所有对外地址（公网 IP 查询、STUN 服务器等）可见、可改、可关
- 配置为明文 JSON，可直接纳入 git 做审计和回滚

## 功能

| 模块 | 状态 |
| --- | --- |
| 端口转发（TCP/UDP、IPv4/IPv6、端口段、多目标故障转移、连接数限制、流量统计） | ✅ |
| 动态域名 DDNS（11 家服务商、URL/网卡/命令取 IP、IPv4/IPv6、Webhook） | ✅ |
| 从 Lucky 导入（`.lkcf` 解密与字段映射） | ✅ 端口转发、DDNS |
| STUN 内网穿透 | 规划中（M3） |
| SSL 证书（ACME） | 规划中（M4） |
| Web 服务 / 反向代理 | 规划中（M5） |

## 快速开始

### Docker

```bash
docker build -t gately .
docker run -d --name gately --network host -v ./gately-data:/data gately
docker logs gately 2>&1 | grep 初始密码
```

打开 `http://<设备IP>:16680`，用 `admin` 和日志中的初始密码登录，然后在"设置"中修改密码。端口转发需要监听宿主机端口，推荐使用 `--network host`。

### 二进制

```bash
./gately -c /etc/gately
```

参数：`-c` 配置目录（也可用环境变量 `GATELY_CONFIG_DIR`），`-listen` 覆盖管理界面监听地址，`-version` 输出版本。

### 从 Lucky 迁移

在"设置 → 从 Lucky 导入"中选择 Lucky 配置目录（Docker 默认 `/app/conf`）里的 `lucky_*.lkcf` 文件，先预览再导入。暂不支持的选项（如 TLS/流加密、限速）会逐条列出。

## 开发

依赖：Go 1.26+、Node.js 24+、pnpm。

```bash
make web      # 构建前端（输出到 web/dist，由 go:embed 嵌入）
make build    # 编译 bin/gately
make test     # go test -race ./...
make lint     # gofmt、go vet、前端类型检查
```

前端开发时运行 `cd web && pnpm dev`，`/api` 会代理到本地 `127.0.0.1:16680`。

目录结构与技术选型见 [docs/ROADMAP.md](docs/ROADMAP.md)。

## 贡献约定

- 与 Lucky 的兼容基于公开可观察的行为、接口和配置格式（见 `docs/analysis/`）。请勿提交从 Lucky 二进制或前端产物中复制的代码。
- 新模块需要：配置结构 + 校验、运行时启停、REST 接口、单元测试；能从 Lucky 迁移的模块实现 `luckyimport.Importer`。
- 代码注释使用中文；控制语句必须使用大括号。

## 许可证

[Apache License 2.0](LICENSE)
