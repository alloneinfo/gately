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
# curl 与 bash 供用户脚本使用（证书变化脚本、STUN 通知脚本等，与 Lucky 镜像保持一致）
# docker-cli 与 docker-cli-compose 供 Docker 管理模块执行 docker compose up/pull/build
RUN apk add --no-cache ca-certificates tzdata curl bash docker-cli docker-cli-compose
COPY --from=build /out/gately /usr/local/bin/gately
# 配置目录使用 /etc/gately，避免与迁移自 Lucky 的 /data/... 挂载路径冲突
ENV GATELY_CONFIG_DIR=/etc/gately
VOLUME ["/etc/gately"]
EXPOSE 16680
ENTRYPOINT ["/usr/local/bin/gately"]
