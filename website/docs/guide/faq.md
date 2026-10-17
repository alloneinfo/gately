# 常见问题

## Gately 与 Lucky 是什么关系？

Gately 是独立实现的开源项目，与 Lucky 项目方没有隶属关系。Gately 的产品思路来自 Lucky，没有 Lucky 就不会有 Gately，感谢 Lucky 作者和社区的贡献。兼容性基于公开可观察的行为、接口与配置格式，仓库拒收从 Lucky 二进制或前端产物复制的代码。

## 需要连接项目方的服务器吗？

无需连接。Gately 默认不连接任何项目方服务器，也不上报设备信息。公网 IP 查询等对外地址使用第三方公共服务，可以在对应任务中修改。

## 初始密码在哪里？

首次启动时输出在日志中，仅打印一次。Docker 部署执行 `docker logs gately 2>&1 | grep -i "initial password"`。

## 忘记管理员密码怎么办？

停止 Gately，删除配置目录中的 `account.json`，再启动：程序会重新创建 `admin` 账号，并把新的初始密码打印到日志。登录后请检查两步验证等安全设置。

## 为什么推荐 `--network host`？

端口转发、STUN、Web 服务需要直接监听宿主机端口。使用桥接网络时，需要自行映射每一个用到的端口。

## 可以和 Lucky 同时运行吗？

可以。管理端口默认 16680，与 Lucky 的 16601 互不冲突。导入的规则默认停用，避免与仍在运行的 Lucky 争用端口和路由器 UPnP 映射。

## Tailscale、frp、cloudflared 为什么要自己放二进制？

这三个模块托管官方外部程序，Gately 本体不内置它们，因此始终可以使用上游最新版本。放置位置见 [配置与运行参数](/docs/guide/configuration)。

## STUN 规则启用后外网仍无法访问？

路由器 WAN 口位于运营商级 NAT（100.64.0.0/10）时，外部无法访问穿透端口，这是网络环境本身的限制。可改用 [Tailscale / frp / Cloudflare 隧道](/docs/modules/tunnels)。

## 如何反馈问题或参与开发？

在 [GitHub Issues](https://github.com/wangwanjie/gately/issues) 提交问题，开发约定见 [参与贡献](/docs/guide/contributing)。
