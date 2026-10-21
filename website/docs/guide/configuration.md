# 配置与运行参数

## 命令行参数

| 参数 | 说明 |
| --- | --- |
| `-c` | 配置目录，也可以用环境变量 `GATELY_CONFIG_DIR` 指定 |
| `-listen` | 覆盖管理界面监听地址，例如 `:16680` |
| `-apply` | 声明式配置目录，也可以用环境变量 `GATELY_APPLY_DIR` 指定，见下文 |
| `-version` | 输出版本信息 |

| 环境变量 | 说明 |
| --- | --- |
| `GATELY_INITIAL_PASSWORD` | 首次启动时的管理员初始密码（至少 8 位），不设置时为默认密码 `gately.cn`；账号已存在时不起作用 |

## 配置目录

所有配置保存在配置目录中，每个模块一个 `<name>.json` 明文文件，原子写入，文件权限 0600。因此：

- 可以直接用文本编辑器查看；
- 可以把目录纳入 git，用 diff 审计每次变更并随时回滚；
- 备份只需复制目录。

::: warning 注意
`ssl.json` 中含有证书私钥，纳入版本库前请确认仓库为私有，或把该文件加入忽略列表。
:::

## 声明式配置

指定 `-apply <目录>` 后，每次启动会把该目录下的 `<名称>.json` 覆盖到配置目录，内容相同的文件不会重写。适合把配置放在 git 仓库、Kubernetes ConfigMap 或部署脚本中统一下发。

- 写入前校验是合法 JSON，任一文件无效则拒绝启动；
- `account.json` 与 `tokens.json` 不允许覆盖，账号与令牌只能在管理界面管理；
- 被覆盖的旧内容会保存在 [配置历史](/docs/guide/audit) 中，可随时回滚。

## 外部程序目录

Tailscale、frp、cloudflared 这三个模块托管官方的外部程序，按以下顺序查找可执行文件：

1. 模块设置中填写的路径；
2. `PATH`；
3. 配置目录下的 `bin/` 子目录。

Docker 部署时把二进制放进挂载的配置目录即可，例如 `./gately-data/bin/frpc`。Tailscale 需要同时提供 `tailscale` 与 `tailscaled`。详见 [Tailscale / frp / cloudflared](/docs/modules/tunnels)。

## 备份与恢复

「设置」中提供配置备份与恢复，并承担从 Lucky 导入配置的入口。
