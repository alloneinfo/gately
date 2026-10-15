# 多阶段构建：先构建前端，再把产物嵌入 Go 二进制
FROM node:24-alpine AS web
WORKDIR /src/web
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
ARG VERSION=dev
ARG COMMIT=none
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/wangwanjie/gately/internal/version.Version=${VERSION} -X github.com/wangwanjie/gately/internal/version.Commit=${COMMIT} -X github.com/wangwanjie/gately/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /out/gately ./cmd/gately

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/gately /usr/local/bin/gately
ENV GATELY_CONFIG_DIR=/data
VOLUME ["/data"]
EXPOSE 16680
ENTRYPOINT ["/usr/local/bin/gately"]
