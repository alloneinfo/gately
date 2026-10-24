# WAF 防火墙

基于 [Coraza](https://coraza.io/) 与 OWASP CRS v4 的 Web 应用防火墙，可挂接到 [Web 服务](/docs/modules/web) 的站点。

## 能力

- 内置 OWASP CRS v4 规则集，可按规则文件启用；
- 支持自定义 SecLang 规则；
- 规则组可调整顺序；命中日志记录客户端 IP、域名、路径与规则 ID，可在日志中按域名、IP 或关键词筛选；
- 拦截模式与仅检测模式；
- 请求体检查上限可调，超过上限时拒绝请求或只检查前面部分；可按路径跳过请求体检查；
- WebDAV 兼容模式：放行网盘、WebDAV 用到的方法、请求头与任意上传内容类型；
- 多个实例，站点按需引用。

## 使用建议

1. 新站点先使用「仅检测」，观察一段时间的日志，确认没有误报；
2. 针对误报调整规则文件或添加自定义例外；
3. 再切换为「拦截」。

## 规则组顺序

实例里的规则组从上到下加载，可用上移、下移按钮调整。同一阶段内排在前面的规则先执行：

- 拦截规则（deny 等）放在哪里都会生效；
- 运行期的放行或排除规则（`allow`、`ctl:ruleRemoveById`、`ctl:ruleRemoveTargetById` 等）要放在 OWASP CRS 规则组之前，否则请求可能已被 CRS 拦截；
- `SecRuleRemoveById`、`SecRuleUpdateTargetById` 这类配置期指令只作用于已经加载的规则，要放在 CRS 规则组之后，放在之前不起作用。

## 上传大文件的站点

WAF 默认只检查不超过 12.5 MB 的请求体，更大的请求会被拒绝（413），网盘、WebDAV、文件管理这类站点因此无法上传大文件。在实例的「请求体检查」中可以调整：

- **检查上限**：0 表示默认的 12.5 MB，最大 1024 MB；
- **超过上限时**：「拒绝请求」与之前一致；「只检查前面部分」只检查不超过上限的部分，其余内容原样转发给后端，适合上传大文件的站点；
- **兼容 WebDAV 与上传**：OWASP CRS 默认只允许 GET、HEAD、POST、OPTIONS，并拦截 `Lock-Token`、`If`、`Content-Range` 请求头与白名单以外的内容类型，WebDAV 客户端因此无法列目录、上传和加锁。开启后放行 PUT、DELETE、PROPFIND、PROPPATCH、MKCOL、COPY、MOVE、LOCK、UNLOCK 等方法与这些请求头，不再限制上传的内容类型；PUT 与 `application/octet-stream` 的请求体是文件内容，不再按表单参数检查。注入、路径穿越等攻击检查照常生效；
- **跳过请求体的路径**：每行一个路径前缀（如 `/api/upload`、`/remote.php/dav/`），匹配的请求不检查请求体，请求头、URL 与参数仍照常检查。

推荐为网盘、WebDAV 这类站点单独建一个实例：开启「兼容 WebDAV 与上传」并选择「只检查前面部分」，必要时把上传接口加入跳过路径，其他站点继续使用默认设置。

curl、rclone 等客户端上传较大的内容时会带 `Expect: 100-continue` 请求头，CRS 4 默认会拦截它。Gately 的所有实例都放行该请求头（由 Gately 自行处理 100-continue），不需要额外设置。

## 验证

WAF 使用真实攻击载荷测试，覆盖路径穿越等攻击的拦截。
