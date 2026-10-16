# 从 Lucky 迁移到 Gately

Gately 可以直接读取 Lucky 的 `.lkcf` 配置文件或 Lucky「备份配置」导出的整站 zip，把规则、任务、证书迁移过来。整个过程不需要卸载 Lucky，可以两边并存、逐个模块切换。

## 1. 准备

1. 找到 Lucky 的配置目录：Docker 部署通常是映射到容器 `/app/conf` 的宿主机目录，里面有 `lucky_base.lkcf`、`lucky_ddns.lkcf` 等文件。
2. 部署 Gately（见 README）。建议：
   - 使用 `--network host`，与 Lucky 一样直接监听宿主机端口；
   - 管理端口默认 16680，与 Lucky 的 16601 不冲突；
   - 如果 Lucky 的脚本或证书映射用到了容器内路径（例如 `/etc/lucky/custom_scripts`、`/data/nginx/...`），在 Gately 容器中挂载到**相同路径**，导入后无需修改。

## 2. 导入

打开 Gately「设置 → 备份与迁移」，选择全部 `lucky_*.lkcf` 文件，或直接选择 Lucky 导出的整站配置 zip（压缩包内的 `.lkcf` 会被自动识别，其他文件忽略），先点「预览」查看每个模块能导入多少条、有哪些提示，确认后点「导入」。

重复导入是安全的：与已有条目同名的规则、任务、证书会被跳过。

支持的文件：

| 文件 | 导入到 | 说明 |
| --- | --- | --- |
| lucky_portforward.lkcf | 端口转发 | TLS/流加密、限速选项暂不支持，会提示 |
| lucky_ddns.lkcf | 动态域名 | 支持 11 家服务商；Lucky 官方 IP 查询地址会被移除 |
| lucky_stun.lkcf | STUN 穿透 | 含全局 STUN 服务器列表与全局 Webhook |
| lucky_ssl.lkcf | SSL 证书 | 已签发的证书原样导入，未到期前不会重新申请 |
| lucky_reverseproxy.lkcf | Web 服务 | 站点、认证、WAF 引用（子规则 WAF 为 main 时沿用监听规则的实例） |
| lucky_coraza.lkcf | WAF 防火墙 | 沿用实例 key，站点引用保持有效 |
| lucky_wol.lkcf | 网络唤醒 | |
| lucky_webterminal.lkcf | Web 终端 | SSH 连接；首次连接时记录主机指纹 |
| lucky_webdav.lkcf | WebDAV | |
| lucky_base.lkcf | 全局设置、Web 服务 | 仅允许内网访问、登录失败上限、会话时长、时区、自定义 DNS（含 doh@ 写法）、GC 参数、PROXY 协议可信来源。管理端口、安全入口、管理员账号与两步验证不迁移 |

Lucky 的计划任务、IP 过滤、Docker 设置在常见配置中为空，暂不迁移；有内容时请手动配置。

**导入的规则默认停用**：防止在测试机上导入后把线上域名解析到错误地址，或与仍在运行的 Lucky 争用端口和路由器 UPnP 映射。确认 Gately 就运行在目标设备上、且 Lucky 已停止时，可以勾选「导入后保持 Lucky 中的启用状态」，规则会按 Lucky 中的启用状态（包括各模块的总开关）立即运行。

## 3. 切换

建议按模块逐个切换，每切一个验证一个：

1. **证书**：在 Gately 中启用证书。启用后会把证书写入映射目录并执行变化脚本（与 Lucky 相同的文件名），确认目录可写。
2. **DDNS**：在 Lucky 中停用任务 → 在 Gately 中启用对应任务 → 点「强制同步」确认记录已更新。
3. **端口转发 / STUN**：先在 Lucky 中停用规则（释放端口和 UPnP 映射），再在 Gately 中启用。STUN 规则启用后公网端口可能变化，脚本与 Webhook 会自动通知。
4. **Web 服务**：停用 Lucky 的监听规则 → 在 Gately 中启用。HTTPS 站点需要证书模块中已有覆盖该域名的启用证书。
5. 全部确认无误后再停止 Lucky 容器。

## 4. 已在真实配置上验证的行为

以下结论来自一份包含数十条规则的真实 Lucky 2.27.2 配置，在同一台设备上让 Gately 与 Lucky 并行运行对比得出：

- **DDNS**：阿里云任务用导入的密钥完成鉴权，全部记录与线上值一致，未产生任何写入。
- **端口转发**：每条规则的转发结果与直连目标一致；目标离线的规则在两边表现相同。
- **Web 服务**：每个站点经 Gately 与经 Lucky 返回相同的状态码、页面与证书；WAF 对路径穿越的拦截结果一致。
- **证书映射**：写出的 `.crt/.key/.pem/_issuerCertificate.crt` 与 Lucky 写出的文件逐字节相同。
- **Web 终端**：可达主机均认证成功；不可达或密钥失效的连接两边表现相同（可用「测试连接」快速检查）。
- **STUN**：规则配置迁移正确。路由器 WAN 口为运营商级 NAT（100.64.0.0/10）时，外部无法访问穿透端口，这是网络本身的限制，Lucky 同样无法穿透。

## 5. Docker 部署的挂载

Lucky 的脚本与证书映射使用的是 Lucky 容器内的路径，Gately 容器需要把相同的宿主机目录挂载到相同的容器内路径。对照 Lucky 的 compose 逐条添加，例如：

```yaml
volumes:
  - ./config:/etc/gately
  - /var/run/docker.sock:/var/run/docker.sock
  # Lucky 中证书映射到 /data/nginx/ssl，则挂载同一个宿主机目录到同一路径
  - /srv/nginx/ssl:/data/nginx/ssl
  # 规则中调用的脚本（注意核对 Lucky 中填写的路径在 Lucky 容器内是否真实存在）
  - /srv/lucky/custom_scripts:/etc/lucky/custom_scripts:ro
```

Gately 镜像内置 `sh`、`bash` 与 `curl`，常见的通知脚本可以直接运行。

## 6. 需要人工处理的项目

- **第三方登录**：Lucky 的 GitHub/QQ/Google 登录经过 Lucky 官方中转服务器。Gately 需要你自己申请 OAuth 应用（GitHub OAuth App 或 Google OIDC），在「Web 服务 → 全局设置」中填写，并在站点上配置允许的身份。只配置了第三方登录的站点导入后保持停用。
- **ZeroSSL 证书**：沿用 Lucky 中的 EAB 凭据注册新的 ACME 账户；若 ZeroSSL 拒绝，在 ZeroSSL 后台重新生成 EAB。
- **Lucky 特有的功能**：FRP、Cloudflared、FTP、DLNA、rclone、存储管理、IP 库等暂未实现，导入时会跳过。
- **Lucky 官方服务器**：Gately 不会连接 `66666.host`、`lucky666.cn` 等地址，迁移后公网 IP 查询使用第三方公共服务（可在 DDNS 任务中修改）。

## 7. 回退

Gately 不修改 Lucky 的任何文件。如需回退，停用 Gately 中对应的规则，再在 Lucky 中重新启用即可。
