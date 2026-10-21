# 常见问题

## 需要连接项目方的服务器吗？

无需连接。Gately 默认不连接任何项目方服务器，也不上报设备信息。公网 IP 查询等对外地址使用第三方公共服务，可以在对应任务中修改。

## 初始密码在哪里？

首次启动时输出在日志中，仅打印一次。Docker 部署执行 `docker logs gately 2>&1 | grep -i "initial password"`。

## 忘记管理员密码怎么办？

停止 Gately，删除配置目录中的 `account.json`，再启动：程序会重新创建 `admin` 账号，并把新的初始密码打印到日志。登录后请检查两步验证等安全设置。

## 支持飞牛 fnOS 吗？

支持。每个版本附带飞牛离线安装包（`.fpk`），在应用中心手动安装。它接入飞牛统一网关，点击桌面图标即可打开，远程访问飞牛的域名同样可用；在「设置 → 访问控制」开启「飞牛登录态自动登录」后，已登录的飞牛管理员可直接进入，无需再输入密码与两步验证码。详见 [飞牛 fnOS 原生应用](/docs/guide/install#fnos)。

## 支持 OpenWrt / iStoreOS 吗？

支持。每个版本附带 OpenWrt 安装包：OpenWrt 24.10 及以前（含 iStoreOS）用 `.ipk`，执行 `opkg install` 安装；OpenWrt 25.12 起用 `.apk`，执行 `apk add --allow-untrusted` 安装。安装后自动启动，配置保存在 `/etc/gately`，升级时安装新版安装包即可。程序约 35 MB，适合存储较大的软路由。详见 [OpenWrt](/docs/guide/install#openwrt)。

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
