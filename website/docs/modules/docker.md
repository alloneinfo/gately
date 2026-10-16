# Docker 管理

通过 Docker 套接字管理宿主机上的容器。

## 能力

- 容器启停与日志查看；
- 镜像更新检查与一键升级，升级失败自动回滚；
- 完整的接口说明见 [Docker API](/docs/reference/docker-api)。

## 启用方式

挂载宿主机的 Docker 套接字：

```bash
-v /var/run/docker.sock:/var/run/docker.sock
```

::: warning 安全提示
挂载 `docker.sock` 等同于授予宿主机 root 级权限。不需要 Docker 管理时请去掉该挂载，并保护好管理界面（两步验证、安全入口、仅内网访问）。
:::
