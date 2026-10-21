# API 令牌与 OpenAPI

Gately 的管理界面使用的全部接口都可以通过 HTTP 调用。脚本与自动化使用 API 令牌鉴权，接口说明由程序根据已登记的路由自动生成。

## API 令牌

管理界面路径：侧边栏「系统」分组 → 「API 令牌」。

### 创建令牌

点击「新建令牌」，填写：

| 项目 | 说明 |
| --- | --- |
| 名称 | 必填，最长 64 个字符，用于标识用途，如「监控脚本」 |
| 过期时间 | 可选「永不过期」、7 天、30 天、90 天、1 年或自定义日期；自定义时间必须晚于当前时间 |
| 权限 | 必填，至少选择一项。新建时默认不授予任何权限 |

单个实例最多保存 100 个令牌。

创建成功后弹窗只显示一次令牌明文，格式为 `gtl_` 加 48 位十六进制字符。磁盘上只保存 SHA-256 哈希，遗失后只能「重新生成」，旧值立即失效。列表中以 `gtl_xxxx…yyyy` 形式的提示辨认令牌，并显示最近使用时间与来源 IP（落盘最多滞后一分钟）。

编辑令牌可以修改名称、权限、过期时间和启用开关，修改立即生效，令牌明文保持不变。已过期的令牌可以保留原过期时间并只修改其他字段。

### 鉴权方式

请求头携带 `Authorization: Bearer <令牌>`：

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/status
```

请求体使用 `Content-Type: application/json`，错误统一返回 `{"error": "说明"}`。令牌被禁用、过期或不存在时，请求继续按登录会话校验，最终返回未登录错误。

### 权限范围

权限格式为 `<权限域>:read` 或 `<权限域>:write`，写权限包含读权限。权限域是功能模块的 ID（如 `ddns`、`portforward`、`notify`、`backup`）加上三个内置域：

| 权限域 | 名称 | 可用权限 | 对应接口 |
| --- | --- | --- | --- |
| `status` | 系统状态 | `status:read` | `/api/status`、`/api/system/info`、`/api/system/update` |
| `logs` | 系统日志 | `logs:read` | `/api/logs` |
| `import` | Lucky 配置导入 | `import:write` | `/api/import/lucky` |

通配写法：

| 写法 | 含义 |
| --- | --- |
| `*:read` | 全部模块只读 |
| `*:write` 或 `*` | 全部模块读写 |
| `ddns:read` | 仅 DDNS 只读 |

接口所需权限由路径和方法决定：

- 路径 `/api/<模块>/...` 需要 `<模块>` 域的权限，GET 与 HEAD 要求 `read`，其余方法要求 `write`。
- 模块日志 `GET /api/<模块>/logs` 属于只读，清空日志（DELETE）属于写操作。
- 下列 GET 接口会返回敏感内容或建立交互通道，按 `write` 处理：路径中含有 `ws`（WebSocket 终端）、`exec`、`download`、`export`、`read`、`config`、`preview-archive` 段的接口，例如证书下载、容器内执行命令与文件读取。
- `GET /api/session` 与 `GET /api/openapi.json` 接受任意有效令牌。

### 令牌禁止的操作

以下接口只接受登录会话，令牌访问一律返回 403：

- 账号与两步验证：`/api/account`
- 令牌管理：`/api/tokens`
- 系统设置、系统令牌、重启、在线升级：`/api/system/settings`、`/api/system/token`、`/api/system/restart`、`/api/system/upgrade`
- 配置备份下载与恢复：`/api/system/backup`、`/api/system/restore`
- 本地备份文件下载：`/api/backup/local`
- Web 终端安全设置：`/api/webterminal/security`
- 登录与登出：`/api/login`、`/api/logout`

权限不足的请求返回 `令牌缺少权限 <权限>`，并在系统日志中记录一条警告，包含令牌名称、来源 IP 与被拒绝的接口。

### 遗留令牌

旧版本的单个全局令牌在升级时自动迁移为名为「遗留令牌（由旧版迁移）」的令牌，权限为 `*`，列表中带「遗留」标记。建议创建按需授权的新令牌后删除它；对遗留令牌「重新生成」会同时去掉遗留标记。

::: tip 建议
为每个脚本创建单独的令牌，只勾选它需要的权限，并设置过期时间。令牌泄露时删除或重新生成对应的一个即可。
:::

## OpenAPI 与接口文档

### 管理界面里的文档页

侧边栏「系统」分组 → 「API 文档」。页面展示 Base URL、认证说明，按功能分组列出全部接口，支持按路径或摘要搜索。展开一个接口可以看到所需权限、参数、请求体字段和可复制的 curl 示例。需要登录会话的接口标记为「仅登录会话」，不提供 curl 示例。

### 获取 OpenAPI 文档

实例运行时，`GET /api/openapi.json` 返回 OpenAPI 3.0 文档，任意有效令牌或登录会话均可读取：

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/openapi.json -o openapi.json
```

文档使用 `bearerAuth` 安全方案，并为每个接口增加扩展字段：

| 扩展字段 | 含义 |
| --- | --- |
| `x-required-scope` | 访问该接口所需的令牌权限 |
| `x-token-allowed` | 令牌是否允许访问，`false` 表示仅登录会话 |
| `x-scope-domains`（文档根） | 全部权限域列表 |

[命令行与 MCP](/docs/guide/cli-mcp) 的接口清单读取自这个文档。

### 命令行导出

`-api-docs` 参数在临时配置目录中构建应用（不监听端口、不影响现有配置），把文档写入指定目录后退出：

```bash
gately -api-docs ./api-docs
```

输出两个文件：

- `openapi.json`：OpenAPI 3.0 描述
- `API.md`：Markdown 版接口文档，含快速开始、权限域表和按分组列出的接口表

## 调用示例

```bash
# 查看系统状态（需要 status:read）
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/status

# 列出全部 DDNS 任务（需要 ddns:read）
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks

# 立即同步一个 DDNS 任务（需要 ddns:write，key 取自上一步返回的列表）
curl -X POST -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks/KEY/sync

# 只拉取序号大于 10 的模块日志（需要 ddns:read）
curl -H "Authorization: Bearer gtl_xxxx" "http://localhost:16680/api/ddns/logs?since=10"
```

## 完整接口清单

全部接口、参数与所需权限见 [API 接口清单](/docs/reference/api-reference)。
