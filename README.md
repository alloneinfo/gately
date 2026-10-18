# Gately

**English** | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md) | [한국어](README.ko.md) | [Español](README.es.md) | [Français](README.fr.md) | [Deutsch](README.de.md) | [Русский](README.ru.md) | [Português (Brasil)](README.pt-BR.md)

An open-source, auditable network gateway toolkit for home networks, NAS boxes, cloud hosts, and servers of all kinds. DDNS, STUN traversal, port forwarding, reverse proxy, certificates, WAF, and more ship in a single ~22 MB file, and existing Lucky configurations can be imported.

![Gately overview](docs/images/overview.webp)

![Dynamic DNS](docs/images/ddns.webp)

![Docker](docs/images/docker.webp)

## Why Gately

Gately is fully open source, so users and the community can read the source and the configuration directly. For software that sits on your network edge and holds your DNS keys and certificate private keys, open source lets you see exactly what it does. Gately's commitments:

- The source is released under Apache-2.0 and open to community review
- No connections to any project-operated server by default and no device information reported; third-party sign-in uses your own OAuth apps, with no relay in between
- Every outbound address (public IP lookup, STUN servers, and so on) is visible, editable, and can be turned off
- Configuration is plain-text JSON, so you can put it under git for auditing and rollback

## Features

| Module | Capabilities |
| --- | --- |
| Dynamic DNS | Alibaba Cloud, Cloudflare, Tencent Cloud, DNSPod, Huawei Cloud, GoDaddy, Porkbun, dynv6, DuckDNS, custom callbacks; IP discovery via URL, network interface, or command; IPv4/IPv6; webhooks |
| STUN traversal | TCP/UDP, keep-alive, UPnP/NAT-PMP, built-in forwarding, webhooks and scripts triggered on address change |
| Port forwarding | TCP/UDP, IPv4/IPv6, port ranges, multi-target failover, connection limits, traffic statistics |
| SSL certificates | Automatic ACME DNS-01 issuance and renewal (wildcard, EAB), upload, read from file, map to a directory and run a script |
| Tailscale | Managed tailscaled (userspace networking, container-friendly) or attach to an existing system daemon; auth keys / login links, Headscale, subnet routes, exit nodes, peer list and SOCKS5 proxy, fixed port, extra tailscaled arguments and environment variables |
| frp | frpc client (tcp/udp/http/https/stcp/xtcp proxies, changes apply instantly) and frps server (port allowlist, subdomains, dashboard); one-click import of legacy ini and new toml configs |
| Cloudflare Tunnel | cloudflared token tunnels and trycloudflare quick tunnels, multiple tunnels, automatic restart, public address display |
| Web services | Per-domain reverse proxy / redirect / static files / fixed text; HTTPS; WebSocket; PROXY protocol in and out; basic auth, web auth, GitHub/Google/QQ/OIDC sign-in; IP allow/deny lists, path rules, temporary share links, automatic ban on failed auth, GeoIP country filtering; backend weights / standby / load-balancing policies, active and passive health checks, compression, rate limiting, response caching, path rewriting; per-minute access statistics and Prometheus metrics |
| WAF | Coraza + OWASP CRS v4, per-file rule toggles, custom SecLang, block or detection-only mode |
| Docker management | Create and edit containers, logs and terminal, Compose, image update checks with one-click upgrade (automatic rollback on failure), networks and volumes |
| Web terminal | SSH (trust host key on first use), Telnet, local shell (off by default); proxies, persistent sessions, SFTP file manager, 2FA and protocol confirmation, quick commands, split panes |
| Network diagnostics | Ping, traceroute, port probe, DNS lookup, speed test, LAN scan |
| Notification center | Webhook, Telegram, Bark, WeCom, DingTalk, Feishu, email; severity levels, wildcard, and silencing / dedup rules |
| Config backup | Encrypted backups (scrypt + AES-GCM), retention policy, scheduled backups to a local directory or storage mount |
| CLI and MCP | `gately ctl` talks to a running instance; `gately mcp` serves MCP over stdio (read-only by default) |
| Wake-on-LAN | Magic packet broadcast, online detection |
| WebDAV | Multiple users, per-user directories, read-only mode |
| FTP | Multiple users, per-user directories, read-only mode, FTPS and passive port ranges |
| Storage management | Local directories and remote WebDAV mounts, online file manager |
| Scheduled tasks | Cron expressions, commands or HTTP requests, run history |
| Security and settings | Two-factor authentication (TOTP), login rate limiting, secure entry path, LAN-only access, HTTPS for the admin UI, multiple API tokens (per-module read/write grants), custom DNS (DoH supported), time zone, config backup and restore |
| Migration | Import a Lucky `.lkcf` file or a full-site config zip: port forwarding, DDNS, STUN, certificates, reverse proxy, WAF, Wake-on-LAN, web terminal, WebDAV, basic settings |

## Quick Start

### Docker

Images are available for amd64, arm64, and arm/v7:

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep -i "initial password"
```

Open `http://<device-ip>:16680` and sign in as `admin` with the initial password from the log. Then change the password and enable two-factor authentication under Settings. See [deploy/docker-compose.yml](deploy/docker-compose.yml) for a complete Compose example.

Port forwarding, STUN, and web services listen on host ports, so `--network host` is recommended. If you do not need Docker management, drop the `docker.sock` mount.

### Binary

Download the archive for your platform from [Releases](https://github.com/wangwanjie/gately/releases) (Linux amd64/arm64/arm/mips/mipsle/riscv64, macOS, Windows, FreeBSD), extract it, and run:

```bash
./gately -c /etc/gately
```

Flags: `-c` sets the config directory (the `GATELY_CONFIG_DIR` environment variable also works), `-listen` overrides the admin UI listen address, and `-version` prints the version.

### API usage

Create a token on the "API Tokens" page of the admin console. You can create several, grant each read-only or read-write access per module, set an expiry, and the plaintext is shown only once, right after creation. Scripts can then call the admin API in bulk:

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks
```

Account, two-factor authentication, and token management, as well as the backup, restore, restart, and system settings endpoints, accept login sessions only. All endpoints, required permissions, and data schemas are listed on the "API Docs" page of the admin console. You can also export them to [docs/api/en](docs/api/en) (`openapi.json` and `API.md`) with `make api-docs`; the website https://gately.cn references them directly when published.

When upgrading from an older version, the previous global API token becomes a "legacy token" automatically (same value, full access to every module). Tighten its permissions or replace it with a new token on the API Tokens page.

### Tailscale, frp, cloudflared

These three modules manage the official external programs (automatic restart, logs aggregated in the UI). Gately does not bundle them, so you can always run the latest upstream release. Programs are located in this order: the path set in the module settings, `PATH`, then the `bin/` subdirectory of the config directory. For Docker deployments, drop the binary into the mounted config directory, for example `./gately-data/bin/frpc`; Tailscale needs both `tailscale` and `tailscaled`. Tokens and keys in config files are stored with 0600 permissions, and the cloudflared token is passed via an environment variable so it never appears in the process list.

### Migrating from Lucky

See the [migration guide](docs/en/MIGRATION.md). In short: under "Settings → Import from Lucky", choose the `lucky_*.lkcf` file from your Lucky config directory, preview it, then import. Imported rules are disabled by default; go module by module, disabling in Lucky and enabling in Gately.

## Development

Requirements: Go 1.26+, Node.js 24+, pnpm.

```bash
make web      # build the frontend (outputs to web/dist, embedded via go:embed)
make build    # compile bin/gately
make test     # go test -race ./...
make lint     # gofmt, go vet, frontend type check
```

For frontend development, run `cd web && pnpm dev`; `/api` is proxied to local `127.0.0.1:16680`.

Some tests are skipped by default and need extra setup:

| Test | How to enable |
| --- | --- |
| ACME end-to-end (Pebble) | Install pebble and pebble-challtestsrv, set `GATELY_PEBBLE_BIN` and `GATELY_PEBBLE_SRC` (runs automatically in CI) |
| Real public STUN servers | `GATELY_NET_TEST=1` |
| Import validation with a real Lucky config | `GATELY_LUCKY_SAMPLE` and others, see `TestImportRealSample` in each module |

For the directory layout and technology choices, see [docs/ROADMAP.md](docs/ROADMAP.md).

The website ([gately.cn](https://gately.cn)) source lives in `website/` and is built with VitePress: `cd website && pnpm install && pnpm dev`; `pnpm deploy:site` builds and syncs it to the server.

## Contributing

- Lucky compatibility is based on publicly observable behavior, interfaces, and configuration formats. Do not submit code copied from Lucky binaries or frontend bundles.
- A new module needs a config schema with validation, runtime start/stop, REST endpoints, and unit tests. Modules that can be migrated from Lucky implement `luckyimport.Importer`; modules that want to show status on the overview page implement `module.Summarizer`.
- Code comments are written in Chinese; control statements must use braces.

## Community

Questions, feedback and usage tips are welcome in the Telegram group: [t.me/gately_chat](https://t.me/gately_chat). Please report bugs in [GitHub Issues](https://github.com/wangwanjie/gately/issues) and attach a [diagnostics bundle](docs/en/DIAGNOSTICS.md) (Settings → Backup & Migration); it is generated locally and contains no passwords, tokens, domains or real IPs.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=wangwanjie/gately&type=Date)](https://star-history.com/#wangwanjie/gately&Date)

## License

[Apache License 2.0](LICENSE)
