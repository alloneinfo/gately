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

### systemd 示例

```ini
[Unit]
Description=Gately
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/gately -c /etc/gately
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## 初始登录

管理界面默认监听 `16680` 端口，用户名 `admin`，初始密码在首次启动时输出到日志。首次登录后请修改密码并开启两步验证。

## 升级

Gately 不内置在线自更新，升级由部署方式负责：

- Docker：`docker pull` 新镜像后重建容器（配置目录保持挂载即可）；
- 二进制：替换可执行文件并重启服务。

升级前建议导出一份配置备份，或直接把配置目录纳入 git。
