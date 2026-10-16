# 模块总览

Gately 的每项能力是一个独立模块，可单独启停、单独配置，配置各自保存为一个明文 JSON 文件。

| 模块 | 能力 | 文档 |
| --- | --- | --- |
| 动态域名 | 11 家 DNS 服务商，URL / 网卡 / 命令取 IP，IPv4 / IPv6，Webhook | [查看](/docs/modules/ddns) |
| STUN 穿透 | TCP / UDP、保活、UPnP / NAT-PMP、内置转发、变化时触发 Webhook 与脚本 | [查看](/docs/modules/stun) |
| 端口转发 | TCP / UDP、IPv4 / IPv6、端口段、多目标故障转移、连接数限制、流量统计 | [查看](/docs/modules/portforward) |
| SSL 证书 | ACME DNS-01 自动签发与续期、上传、映射到目录并执行脚本 | [查看](/docs/modules/ssl) |
| Web 服务 | 反向代理、重定向、静态文件、HTTPS、认证、IP 黑白名单 | [查看](/docs/modules/web) |
| WAF 防火墙 | Coraza + OWASP CRS v4 | [查看](/docs/modules/waf) |
| Tailscale / frp / cloudflared | 托管官方二进制的组网与内网穿透 | [查看](/docs/modules/tunnels) |
| Docker 管理 | 容器启停、日志、镜像更新与一键升级 | [查看](/docs/modules/docker) |
| Web 终端 | SSH 与本机 Shell | [查看](/docs/modules/terminal) |
| 文件与运维工具 | WebDAV、FTP、存储管理、网络唤醒、计划任务 | [查看](/docs/modules/tools) |
