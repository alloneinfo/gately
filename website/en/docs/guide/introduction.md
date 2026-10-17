# Introduction

Gately is an open-source, auditable network gateway toolkit for home networks, NAS boxes, cloud hosts and servers of all kinds. It is inspired by [Lucky](https://github.com/gdy666/lucky) and delivers similar capabilities in a fully open-source form. DDNS, STUN NAT traversal, port forwarding, reverse proxy, certificates, WAF and more are bundled into a single ~22 MB binary, and Lucky configurations can be imported in one click.

## Why Gately

Lucky is a well-regarded tool among NAS and home-network users, especially in China. It brings DDNS, STUN traversal, reverse proxy and more together, and it gave Gately its product direction. Without Lucky there would be no Gately. Lucky is distributed free of charge as closed source; Gately is fully open source, so users and the community can review the code and configuration directly. For software that sits on the network edge and holds DNS keys and certificate private keys, open source lets you see exactly what it does. Gately's promises:

- Source code is open under Apache-2.0 and welcomes community review
- No connections to project servers and no device reporting by default; third-party sign-in uses your own OAuth apps, with no relay
- Every outbound address (public IP lookup, STUN servers, etc.) is visible, editable and can be turned off
- Configuration is plain JSON, ready for git-based auditing and rollback

## Design principles

1. **Transparency**: no connections to project servers by default; every outbound address is visible, editable and can be turned off.
2. **Migratable**: configuration fields stay as close to Lucky's as practical, with `.lkcf` import.
3. **Auditable configuration**: plain JSON (file mode 0600) that goes straight into git for diffs and rollbacks.
4. **Single-file deployment**: one static binary with the frontend embedded, supporting Docker, systemd and OpenWrt.

## Tech stack

| Layer | Choice |
| --- | --- |
| Backend | Go, statically compiled, cross-compiles to arm / arm64 / mips / riscv routers and NAS devices |
| HTTP | Standard library `net/http` |
| Storage | One plain JSON file per module, written atomically |
| Authentication | bcrypt passwords + HMAC-signed session tokens, login rate limiting, TOTP two-factor authentication |
| Frontend | Vue 3 + TypeScript + Vite + Element Plus, bundled into the binary via `go:embed` |

## Feature overview

Dynamic DNS, STUN traversal, port forwarding, SSL certificates, web services (reverse proxy, access control, health checks, caching and rate limiting, traffic stats), WAF firewall, Tailscale, frp, Cloudflare Tunnel, Docker management, web terminal, network diagnostics, Wake-on-LAN, WebDAV, FTP, storage management, scheduled tasks, notification center and configuration backup, plus two-factor authentication, secure entrance path, API tokens, CLI and MCP. See the [Modules Overview](/en/docs/modules/) for details.

## Next steps

- Want to try it right away: [Quick Start](/en/docs/guide/quick-start)
- Already using Lucky: [Migration guide](/en/docs/guide/migration)
- Care about privacy and security: [Security Model & Privacy Promise](/en/docs/guide/security)
