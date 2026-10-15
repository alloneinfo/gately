# 待决事项

自主开发过程中遇到、需要维护者决定或验收的问题，按里程碑累积。处理完毕后在条目前标记 ✅ 并记录结论。

## M2 DDNS

1. **所有导入的规则/任务默认停用**（端口转发、DDNS、STUN）。原因：在测试机上导入会把线上域名解析到测试机 IP、意外开放端口，或与仍在运行的 Lucky 争用端口和 UPnP 映射。代价：迁移后需要逐个手动启用。可选方案：导入页面增加"导入后立即启用"开关。
2. **导入时移除 Lucky 官方 IP 查询地址**（`66666.host` / `666666.host`）。你的两个任务各有 3~4 个这类地址，移除后仍保留 ipw.cn、ipip.net 等第三方地址。如需保留请告知。
3. **DNS 服务商验证程度**：阿里云签名已用官方文档示例校验通过；Cloudflare、腾讯云、DNSPod、华为云、GoDaddy、Porkbun、dynv6、DuckDNS、自定义回调只用模拟服务器验证了请求格式，尚未对真实账号联调。验收时建议用一个测试子域名实际跑一次。
4. **服务商覆盖**：Lucky 支持 25 家，Gately 目前 11 家（含你在用的阿里云）。未实现：aliesa、baiducloud、desec、dnsla、dynadot、edgeone、freemyip、jdcloud、namecom、namesilo、spaceship、vercel、volcengine、west。是否需要优先补齐某几家？

## M3 STUN 穿透

5. **真实网络验证结果**：在你家里的 Mac（经 192.168.18.1 路由器）上实测，TCP 穿透成功：经 stun.nextcloud.com 获得公网地址，UPnP 映射建立，外部访问经内置转发回到本机测试服务；删除规则后已确认路由器上无残留映射。UDP 穿透只做了本地端到端测试。
6. **心跳语义与 Lucky 的差异**：Gately 的"心跳间隔"用于保活连接（向保活服务器发 HEAD 请求，默认 3 秒）；重新查询 STUN 公网地址的周期是全局"地址检测周期"（默认 60 秒），保活连接中断时会立即重新查询。Lucky 可能每次心跳都查询 STUN。如需更快发现端口变化，可调小地址检测周期。
7. **默认 STUN 服务器**：只保留 2026-10 实测支持 TCP 的 5 个（nextcloud、hot-chilli、blackberry、radiojar、freeswitch）。导入 Lucky 配置时会改用你在 Lucky 里配置的 91 个全局服务器。
8. **脚本路径**：你的 17 条规则脚本调用 `/etc/lucky/custom_scripts/*.sh`。Gately 的 Docker 部署需要把该目录挂载到相同路径，或在 Gately 中改成新路径。脚本变量 `${ruleName}` `${port}` 等通过环境变量传入，现有脚本无需修改。
9. **自测期间的一次意外**：第一次导入你的端口转发配置时（当时导入还保留启用状态），25 条规则在 Mac 上监听了不到 1 分钟（转发到你局域网内的主机），已立即停止并改为导入默认停用。影响范围仅限局域网。
10. **浏览器预览工具**：Claude 应用内的预览启动器在启动新编译的二进制时卡住，很可能是 macOS 弹出了"访问文稿文件夹"之类的授权框等待点击。之后的界面自测改用无头 Chrome 截图完成。若看到该弹窗，可点击拒绝或允许。

## M4 SSL 证书

11. **ACME 验证方式只支持 DNS-01**（与你的使用方式一致）。HTTP-01 需要占用 80 端口，暂未实现；如有需要请告知。
12. **DNS 验证支持 9 家服务商**（阿里云、Cloudflare、腾讯云、DNSPod 国内/国际、华为云、GoDaddy、Porkbun、dynv6）。阿里云 TXT 增删的签名与 DDNS 共用、已经官方示例校验；其余只用模拟服务器验证。
13. **账户邮箱**：Lucky 默认使用 `*@acme.lucky666.cn` 形式的邮箱，你的证书配置的是自己的邮箱，导入后沿用。
14. **ZeroSSL 证书（gohome.ddns）**：沿用 Lucky 配置里的 EAB Key ID / HMAC。Gately 会用这组 EAB 注册新的 ACME 账户（Lucky 的账户私钥未迁移），一般可以直接续签；如果 ZeroSSL 拒绝重复绑定，需要在 ZeroSSL 后台生成新的 EAB。
15. **证书私钥的存储位置**：与 Lucky 一样保存在配置文件 `ssl.json` 中（权限 0600）。如果你用 `one_key.sh` 把 Gately 配置同步进 git，私钥也会进入仓库，需要评估；可选方案是把证书内容拆到单独的文件并在备份时排除。

## M5 Web 服务

16. **Coraza WAF**：你的 27 个站点全部挂了 WAF 实例，Gately 暂未实现，迁移后这些站点不经过 WAF。已列入 M7 优先项（计划用 coraza v3 + OWASP CRS）。在 WAF 完成前切换到 Gately，等于暂时关闭了这层防护，请评估。
17. **第三方登录（OAuth）**：download 站点在 Lucky 中允许 GitHub 等第三方账号登录，Gately 暂只支持站点用户（网页认证/基本认证）。你的默认站点只配置了第三方登录、没有站点用户，导入后被停用，需要添加用户后再启用。
18. **HTTP/3、ECH、缓存、限速、按 UA 过滤、单 IP 并发限制、404 计数封禁** 等 Lucky 高级选项暂未实现；你的配置中均未启用（MaxContinuous404Count 仅 1 个站点为 1）。
19. **全局 IP 黑白名单**：Lucky 站点的 SafeIPMode 引用全局 IP 过滤模块，Gately 目前是站点级 IP 列表。你的站点都是黑名单模式且全局黑名单为空，等价于不过滤。
20. **网页认证与 Lucky 的差异**：Gately 的登录页由 Gately 自己渲染，会话 7 天，绑定域名；用户名密码沿用 Lucky 配置。Lucky 的网页认证与管理员登录体系打通，Gately 的站点用户与管理员账号是分开的。
