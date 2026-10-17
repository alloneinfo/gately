VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS := -s -w \
	-X github.com/wangwanjie/gately/internal/version.Version=$(VERSION) \
	-X github.com/wangwanjie/gately/internal/version.Commit=$(COMMIT) \
	-X github.com/wangwanjie/gately/internal/version.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

.PHONY: all web build test lint run clean docker api-docs

all: web build

web:
	cd web && pnpm install --frozen-lockfile && pnpm build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/gately ./cmd/gately

test:
	go test -race ./...

# 导出 API 文档（openapi.json 与 API.md），供官网 https://gately.cn 发布
api-docs:
	go run ./cmd/gately -api-docs docs/api

lint:
	gofmt -l . | (! grep .)
	go vet ./...
	cd web && pnpm typecheck

run: build
	./bin/gately -c ./data

clean:
	rm -rf bin

docker:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) -t gately:$(VERSION) .
