# Compared with Lucky

Gately is inspired by Lucky, covers the common capabilities in an open-source way and can import existing configuration. Lucky leads in feature breadth and user base, and without Lucky there would be no Gately. Thank you.

## Philosophy

| Item | Gately | Lucky |
| --- | --- | --- |
| Source code | Open source (Apache-2.0) | Closed source |
| Config file | Plain JSON, git-friendly | Proprietary format (`.lkcf`) |
| Official services | None by default; all outbound addresses are editable and can be turned off | Online updates, third-party sign-in relay and other official services |
| Third-party sign-in | Your own OAuth apps | Through the official relay service |
| Upgrades | Docker / package manager | Built-in online self-update |
| Feature breadth | Covers the common core features | Broader, e.g. DLNA and cloud-drive mounts |

## Feature coverage

| Capability | Gately |
| --- | --- |
| Dynamic DNS | ✅ |
| STUN traversal | ✅ |
| Port forwarding | ✅ |
| SSL certificates | ✅ |
| Web services (reverse proxy) | ✅ |
| WAF | ✅ |
| Tailscale / frp / cloudflared | ✅ |
| Docker management | ✅ |
| Web terminal | ✅ |
| WebDAV / FTP / storage management | ✅ |
| Wake-on-LAN / scheduled tasks | ✅ |
| Notification center / configuration backup / network diagnostics | ✅ |
| API tokens / OpenAPI / CLI / MCP | ✅ |
| DLNA, cloud-drive mounts, rclone, global IP filtering, HTTP/3 | Planned, scheduled based on community demand |

## Deliberate differences

- No built-in online self-update that overwrites the binary; upgrades are left to your package manager or Docker.
- No remote configuration push and no device reporting.
- Configuration is stored in plain text, and sensitive fields are masked in API output.

## Migration

The two can run side by side, switch over module by module and roll back at any time. See the [Migration guide](/en/docs/guide/migration).
