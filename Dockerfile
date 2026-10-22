# 多阶段构建：先构建前端，再把产物嵌入 Go 二进制
# 构建阶段固定运行在构建机平台上，Go 交叉编译到目标平台（node 镜像不提供 arm/v7 等平台）
FROM --platform=$BUILDPLATFORM node:24-alpine AS web
WORKDIR /src/web
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
ARG VERSION=dev
ARG COMMIT=none
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH GOARM=${TARGETVARIANT#v} CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/wangwanjie/gately/internal/version.Version=${VERSION} -X github.com/wangwanjie/gately/internal/version.Commit=${COMMIT} -X github.com/wangwanjie/gately/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /out/gately ./cmd/gately

FROM alpine:3.23
# curl 与 bash 供用户脚本使用（证书变化脚本、STUN 通知脚本等）
# docker-cli 与 docker-cli-compose 供 Docker 管理模块执行 docker compose up/pull/build
# nftables 供入侵防护把封禁同步到宿主机防火墙（需要 host 网络与 NET_ADMIN 权限）。
# 固定使用 Alpine 3.21 的 nftables 1.1.1：1.1.2 起写入的集合带有旧版 nft 不认识的元数据，
# 宿主机上 1.1.1 之前的 nft（如 Debian 12 / 飞牛的 1.0.6）执行 nft list ruleset 时会崩溃
RUN apk add --no-cache ca-certificates tzdata curl bash docker-cli docker-cli-compose \
    && echo "@nft https://dl-cdn.alpinelinux.org/alpine/v3.21/main" >> /etc/apk/repositories \
    && apk add --no-cache nftables@nft \
    && nft --version | grep -q "v1.1.1 "
COPY --from=build /out/gately /usr/local/bin/gately
# 配置目录使用 /etc/gately，避免与迁移时沿用的 /data/... 挂载路径冲突
ENV GATELY_CONFIG_DIR=/etc/gately
VOLUME ["/etc/gately"]
EXPOSE 16680
ENTRYPOINT ["/usr/local/bin/gately"]
