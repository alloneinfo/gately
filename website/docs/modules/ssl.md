# SSL 证书

自动申请并续期证书，供 [Web 服务](/docs/modules/web) 使用，也可以导出给其他软件。

## 能力

- **ACME DNS-01** 自动签发与续期，复用 [动态域名](/docs/modules/ddns) 中的 DNS 服务商，无需开放 80 / 443 端口；
- 通配符证书与 EAB（如 ZeroSSL）；
- 上传已有证书，或从文件读取；
- 映射到目录：证书变化后写入指定目录并执行脚本，方便其他服务重载。

## 证书映射

启用证书后，Gately 会把 `.crt`、`.key`、`.pem` 与 `_issuerCertificate.crt` 写入映射目录，并执行变化脚本。

## 安全提示

私钥保存在配置目录的 `ssl.json` 中，请妥善保管该文件，纳入版本库时确认仓库为私有。

## 从 Lucky 迁移

已签发的证书可原样导入，未到期前不会重新申请。ZeroSSL 沿用 Lucky 中的 EAB 凭据注册新的 ACME 账户。详见 [迁移指南](/docs/guide/migration)。
