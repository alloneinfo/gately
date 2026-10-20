# 安装部署

## Docker

镜像地址 `ghcr.io/wangwanjie/gately`，支持 amd64、arm64、arm/v7。

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
```

### Docker Compose

```yaml
services:
  gately:
    image: ghcr.io/wangwanjie/gately:latest
    container_name: gately
    restart: unless-stopped
    # 端口转发、STUN 穿透、Web 服务都需要直接监听宿主机端口
    network_mode: host
    volumes:
      # 配置目录（明文 JSON，可纳入 git 备份；ssl.json 中含证书私钥）
      - ./data:/etc/gately
      # Docker 管理（不需要可删除）
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TZ=Asia/Shanghai
```

镜像内置 `sh`、`bash` 与 `curl`，常见的通知脚本可以直接运行。

## 二进制

1. 从 [下载页](/download) 获取对应平台的压缩包：Linux（amd64 / arm64 / arm / mips / mipsle / riscv64）、macOS、Windows、FreeBSD。
2. 解压后运行：

```bash
./gately -c /etc/gately
```

### systemd

仓库提供完整的服务文件 [deploy/systemd/gately.service](https://github.com/wangwanjie/gately/blob/main/deploy/systemd/gately.service)，已授予监听低位端口所需的能力：

```bash
sudo install -m 0755 gately /usr/local/bin/gately
sudo install -m 0644 gately.service /etc/systemd/system/gately.service
sudo systemctl daemon-reload && sudo systemctl enable --now gately
journalctl -u gately | grep -i "initial password"
```

### OpenWrt

使用 procd 启动脚本 [deploy/openwrt/gately.init](https://github.com/wangwanjie/gately/blob/main/deploy/openwrt/gately.init)：

```bash
cp gately /usr/bin/gately && chmod +x /usr/bin/gately
cp gately.init /etc/init.d/gately && chmod +x /etc/init.d/gately
/etc/init.d/gately enable && /etc/init.d/gately start
```

## 飞牛 fnOS 原生应用 {#fnos}

每个版本附带飞牛 fnOS 的离线安装包：`gately_<版本>_fnos_x86.fpk`（Intel / AMD）与 `gately_<版本>_fnos_arm.fpk`（ARM）。安装包没有上架官方应用中心，需要手动安装：

1. 从 [下载页](/download#fnos) 获取与设备架构对应的 `.fpk`；
2. 在飞牛打开「应用中心 → 手动安装」（左下角按钮），选择文件并完成向导；
3. 点击飞牛桌面上的 Gately 图标打开管理界面。

### 支持飞牛统一网关

桌面图标经飞牛统一网关打开管理界面，地址为访问飞牛所用的地址加 `/app/gately/`，与飞牛共用同一个访问入口。局域网地址、远程访问飞牛的域名都能直接使用，无需额外开放或映射 16680 端口。

经网关访问时由飞牛负责访问控制，只有飞牛管理员能打开；安全入口与「仅允许内网访问」只约束直接访问 16680 端口。局域网内仍可直接访问 `http://<NAS 地址>:16680`。

### 支持飞牛管理员登录态直接登录

在「设置 → 访问控制」开启「飞牛登录态自动登录」后，已登录飞牛的管理员点击桌面图标即可直接进入 Gately，无需再输入密码与两步验证码：

- 只有飞牛管理员会以 Gately 管理员身份登录，普通飞牛用户仍需输入账号密码；
- 只对经统一网关的访问生效，直接访问 16680 端口时仍需正常登录；
- 主动退出后不会静默重新登录，登录页会显示「使用飞牛账号 &lt;用户名&gt; 登录」按钮；
- 该开关默认关闭，只在飞牛安装包中显示。

### 运行说明

- 应用以 `root` 身份运行，可以监听 80/443、通过 `docker.sock` 管理 Docker、使用真实的 Web 终端。请只安装在自己掌控的 NAS 上。
- 首次启动时 `admin` 的初始密码写在 `/var/apps/gately/var/gately.log`：`grep -i "initial password" /var/apps/gately/var/gately.log`。
- 配置保存在应用的 `var` 目录，卸载时默认保留；在卸载向导中选择「全部删除」才会清除。
- 安装包同时提供命令行入口 `/usr/local/bin/gately`，可使用 `gately ctl` 与 `gately mcp`。
- 16680 端口与 Docker 镜像相同，启动原生应用前请先停止 `gately` 容器。

## 初始登录

管理界面默认监听 `16680` 端口，用户名 `admin`，初始密码在首次启动时输出到日志。自动化部署时可用环境变量 `GATELY_INITIAL_PASSWORD` 指定初始密码，此时不会生成随机密码。

首次登录会出现启动引导，依次提示修改密码、开启两步验证或 Passkey、运行网络自检、从 Lucky 导入与配置通知。

## 升级

Gately 不内置在线自更新，升级由部署方式负责：

- Docker：`docker pull` 新镜像后重建容器（配置目录保持挂载即可）；
- 二进制：替换可执行文件并重启服务；
- 飞牛 fnOS：在应用中心手动安装新版 `.fpk`，配置自动保留。

升级前建议导出一份配置备份，或直接把配置目录纳入 git。下载的压缩包与镜像可按 [校验发布产物](/docs/guide/verify) 核对签名。
