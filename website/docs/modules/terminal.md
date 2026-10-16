# Web 终端

在浏览器中使用终端，完成日常运维。

## 连接类型

- **SSH**：连接内网任意主机。主机密钥首次连接时记录指纹并信任，之后核对；连接列表提供「测试连接」。
- **本机 Shell**：直接在运行 Gately 的设备上打开 Shell，**默认关闭**，需要在设置中显式启用。

## 安全建议

- 启用本机 Shell 前，确保管理界面已开启两步验证；
- 搭配「仅内网访问」或 VPN 使用；
- 完整接口说明见 [Web 终端 API](/docs/reference/webterminal-api)。

## 从 Lucky 迁移

SSH 连接可从 `lucky_webterminal.lkcf` 导入，首次连接时记录主机指纹。
