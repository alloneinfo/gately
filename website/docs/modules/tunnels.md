# Tailscale / frp / cloudflared

三个模块以进程托管方式集成官方程序：自动重启、日志汇总到界面、配置在界面中完成。Gately 本体不内置这些程序，因此始终可以使用上游最新版本。

## 程序放置

按以下顺序查找可执行文件：

1. 模块设置中填写的路径；
2. `PATH`；
3. 配置目录下的 `bin/` 子目录。

Docker 部署时把二进制放进挂载的配置目录即可，例如 `./gately-data/bin/frpc`。Tailscale 需要同时提供 `tailscale` 与 `tailscaled`。

令牌与密钥以 0600 权限保存，cloudflared 的令牌通过环境变量传递，不会出现在进程列表中。

## Tailscale

- 托管 `tailscaled`（用户态网络，容器友好）或接入系统已有的守护进程；
- 认证密钥或登录链接，支持 Headscale；
- 子网路由与出口节点；
- 节点列表与 SOCKS5 代理；
- 固定 WireGuard UDP 端口，便于在防火墙放行直连；
- tailscaled 额外参数与环境变量。宿主机 iptables 是 nf_tables 后端时，在环境变量中加入 `TS_DEBUG_FIREWALL_MODE=nftables`，子网转发才会生效。

## frp

- **frpc 客户端**：tcp / udp / http / https / stcp / xtcp 代理，增删即生效；
- **frps 服务端**：端口白名单、子域名、控制台；
- **frps 监听地址**：「监听地址」决定 frpc 连接端口监听在哪，「代理监听地址」决定 HTTP / HTTPS 端口和 tcp / udp 代理的远程端口监听在哪，留空都监听所有地址。frps 放在 Gately 等反向代理之后时，把监听地址填 `127.0.0.1`，在 Web 服务中把一个站点反代到 `127.0.0.1:<绑定端口>`，frpc 的传输协议选 websocket 或 wss 并连接该站点；代理监听地址保持留空，tcp 代理的远程端口照常对外开放。导入 frps 配置时，原配置未写 `proxyBindAddr` 会沿用 `bindAddr`，与 frps 的行为一致；
- **导入已有配置**：页头的「导入配置」支持旧版 `frpc.ini` / `frps.ini`（含 `[common]` 段）和 frp 0.52+ 的 `toml`，可选择文件或粘贴内容。先预览解析结果与被忽略的字段，再确认导入；客户端可选择追加（同名代理保持不变）或替换全部代理。导入不会改变启用状态，也可通过接口 `POST /api/frp/import` 调用。

## Cloudflare 隧道

- 令牌隧道与 trycloudflare 临时隧道；
- 多隧道、自动重启；
- 展示公网地址。

## 如何选择

| 场景 | 推荐 |
| --- | --- |
| 在多台设备之间组成私有网络 | Tailscale |
| 自己有公网服务器，把内网服务映射出去 | frp |
| 没有公网 IP 也没有服务器，快速暴露 Web 服务 | Cloudflare 隧道 |
| 运营商给了可打洞的 NAT | [STUN 穿透](/docs/modules/stun) |
