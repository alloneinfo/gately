# 待决事项

自主开发过程中遇到、需要维护者决定或验收的问题，按里程碑累积。处理完毕后在条目前标记 ✅ 并记录结论。

## M2 DDNS

1. **导入的 DDNS 任务默认停用**。原因：如果在测试机上导入并立即运行，会把线上域名解析到测试机 IP。代价：迁移后需要逐个手动启用。可选方案：导入页面增加"导入后立即启用"开关。
2. **导入时移除 Lucky 官方 IP 查询地址**（`66666.host` / `666666.host`）。你的两个任务各有 3~4 个这类地址，移除后仍保留 ipw.cn、ipip.net 等第三方地址。如需保留请告知。
3. **DNS 服务商验证程度**：阿里云签名已用官方文档示例校验通过；Cloudflare、腾讯云、DNSPod、华为云、GoDaddy、Porkbun、dynv6、DuckDNS、自定义回调只用模拟服务器验证了请求格式，尚未对真实账号联调。验收时建议用一个测试子域名实际跑一次。
4. **服务商覆盖**：Lucky 支持 25 家，Gately 目前 11 家（含你在用的阿里云）。未实现：aliesa、baiducloud、desec、dnsla、dynadot、edgeone、freemyip、jdcloud、namecom、namesilo、spaceship、vercel、volcengine、west。是否需要优先补齐某几家？
