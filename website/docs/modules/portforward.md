# 端口转发

把本机监听的端口转发到另一台设备或服务。

## 能力

- TCP 与 UDP；
- IPv4 与 IPv6，v4 / v6 之间可互相转发；
- 端口段与多端口；
- 多目标与故障转移；
- 连接数限制；
- 流量统计与运行日志；
- 向目标发送 PROXY 协议（v1 / v2），把真实客户端地址交给下游。

## PROXY 协议

端口转发只做 TCP 中转，下游默认只能看到 Gately 自身的地址。在规则中把「PROXY 协议」设为 v1 或 v2 后，Gately 连上目标后会先发送 PROXY 头，下游即可取得真实客户端地址，用于日志、限流与访问控制。

目标必须开启对应的接收选项，否则连接会被拒绝，例如：

- haproxy：`bind 192.168.1.8:4434 accept-proxy`；
- nginx：`listen 4435 ssl proxy_protocol;`；
- Gately Web 服务：监听中开启「接收 PROXY 协议」，并把来源加入可信 PROXY 来源。

该选项只作用于 TCP，UDP 转发不受影响。

## 典型场景

- 把 NAS 的服务端口暴露到统一入口；
- 在 IPv6 与 IPv4 之间做互通转发；
- 为一个服务配置主备两个后端。

## 部署提示

端口转发直接监听宿主机端口，Docker 部署推荐使用 `--network host`。

## 从 Lucky 迁移

规则可从 `lucky_portforward.lkcf` 导入；TLS / 流加密、限速等选项暂不支持，导入时会给出提示。
