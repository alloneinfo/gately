# 参与贡献

Gately 是社区驱动的开源项目，欢迎提交 Issue 与 Pull Request。仓库：<https://github.com/wangwanjie/gately>

## 本地开发

依赖：Go 1.26+、Node.js 24+、pnpm。

```bash
make web      # 构建前端（输出到 web/dist，由 go:embed 嵌入）
make build    # 编译 bin/gately
make test     # go test -race ./...
make lint     # gofmt、go vet、前端类型检查
```

前端开发时运行 `cd web && pnpm dev`，`/api` 会代理到本地 `127.0.0.1:16680`。

## 贡献约定

- 与 Lucky 的兼容基于公开可观察的行为、接口和配置格式。请勿提交从 Lucky 二进制或前端产物中复制的代码。
- 新模块需要：配置结构与校验、运行时启停、REST 接口、单元测试；能从 Lucky 迁移的模块实现 `luckyimport.Importer`，希望在总览页展示状态的模块实现 `module.Summarizer`。
- 代码注释使用中文；控制语句必须使用大括号。

## 额外测试

部分测试默认跳过，需要额外条件：

| 测试 | 启用方式 |
| --- | --- |
| ACME 端到端（Pebble） | 安装 pebble 与 pebble-challtestsrv，设置 `GATELY_PEBBLE_BIN`、`GATELY_PEBBLE_SRC` |
| 真实公共 STUN 服务器 | `GATELY_NET_TEST=1` |
| 用真实 Lucky 配置验证导入 | `GATELY_LUCKY_SAMPLE` 等，见各模块 `TestImportRealSample` |

## 许可证

[Apache License 2.0](https://github.com/wangwanjie/gately/blob/main/LICENSE)
