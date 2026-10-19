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

## 初始登录

管理界面默认监听 `16680` 端口，用户名 `admin`，初始密码在首次启动时输出到日志。自动化部署时可用环境变量 `GATELY_INITIAL_PASSWORD` 指定初始密码，此时不会生成随机密码。

首次登录会出现启动引导，依次提示修改密码、开启两步验证或 Passkey、运行网络自检、从 Lucky 导入与配置通知。

## 升级

Gately 不内置在线自更新，升级由部署方式负责：

- Docker：`docker pull` 新镜像后重建容器（配置目录保持挂载即可）；
- 二进制：替换可执行文件并重启服务。

升级前建议导出一份配置备份，或直接把配置目录纳入 git。下载的压缩包与镜像可按 [校验发布产物](/docs/guide/verify) 核对签名。
