# Web 服务

按域名提供反向代理、重定向、静态文件与固定文本，是 Gately 的统一 Web 入口。

## 站点类型

- 反向代理：把域名转发到内网服务，支持 WebSocket；
- 重定向；
- 静态文件；
- 固定文本。

## 传输与协议

- HTTPS：使用 [SSL 证书](/docs/modules/ssl) 模块中覆盖该域名的证书；
- WebSocket；
- PROXY 协议收发。

## 访问控制

- 基本认证、网页认证；
- GitHub、Google、QQ、OIDC 第三方登录（使用自己的 OAuth 应用）；
- IP 黑白名单；
- 可搭配 [WAF 防火墙](/docs/modules/waf)。

## 部署提示

Web 服务需要监听宿主机的 80 / 443 等端口，Docker 部署推荐使用 `--network host`。

## 第三方登录

站点的登录方式在「Web 服务 → 全局设置」中配置：填写自己的 Client ID / Secret 与回调地址并启用后生效。管理后台的第三方登录在「设置 → 账号与安全」中单独配置。
