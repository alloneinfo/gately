# Contributing

Gately is a community-driven open-source project. Issues and pull requests are welcome. Repository: <https://github.com/wangwanjie/gately>

## Local development

Requirements: Go 1.26+, Node.js 24+, pnpm.

```bash
make web      # build the frontend (outputs to web/dist, embedded via go:embed)
make build    # compile bin/gately
make test     # go test -race ./...
make lint     # gofmt, go vet, frontend type check
```

For frontend development, run `cd web && pnpm dev`; `/api` is proxied to local `127.0.0.1:16680`.

## Conventions

- Compatibility with Lucky is based on publicly observable behavior, interfaces and configuration formats. Do not submit code copied from Lucky binaries or frontend artifacts.
- A new module needs: a config structure with validation, runtime start/stop, REST endpoints and unit tests. Modules that can be migrated from Lucky implement `luckyimport.Importer`, and modules that should show status on the overview page implement `module.Summarizer`.
- Code comments are written in Chinese; control statements must use braces.

## Extra tests

Some tests are skipped by default and need extra setup:

| Test | How to enable |
| --- | --- |
| ACME end-to-end (Pebble) | Install pebble and pebble-challtestsrv, and set `GATELY_PEBBLE_BIN` and `GATELY_PEBBLE_SRC` |
| Real public STUN servers | `GATELY_NET_TEST=1` |
| Import verification with a real Lucky config | `GATELY_LUCKY_SAMPLE` and others; see each module's `TestImportRealSample` |

## License

[Apache License 2.0](https://github.com/wangwanjie/gately/blob/main/LICENSE)
