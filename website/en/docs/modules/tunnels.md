# Tailscale / frp / cloudflared

These three modules integrate the official programs as managed processes: automatic restart, logs collected in the UI, and configuration done in the UI. Gately does not bundle these programs; you can install, update and roll them back online from the page.

## Online install, update and rollback {#install}

The frp, Cloudflare Tunnel, Tailscale and [File Browser](/en/docs/modules/tools#filebrowser) pages each have an "online install" card at the bottom:

- **Download and install / Update to latest**: downloads the latest stable release from upstream, verifies it against the SHA-256 recorded upstream, checks that it runs on this machine, installs it into `bin/` under the configuration directory and then restarts the processes that use the program. Nothing is downloaded again when the latest version is already installed;
- **Install a version**: enter a version number (for example `v0.65.0` for frp, `2025.8.0` for cloudflared, `1.90.2` for Tailscale); this also works for downgrading;
- **Download proxy**: can be set per program, supports http, https and socks5, and is saved when an install starts; when empty, the update proxy from "Settings" is used;
- **Previous versions and rollback**: versions replaced by an update or rollback are kept in `bin/.versions/`, at most 3. If a new version has problems, click "Roll back" in the list to switch to an older one; the current version is kept as well, so you can switch back to it later;
- **Uninstall**: deletes the program installed in `bin/` and all previous versions; the module settings are kept. Processes using the program then restart and use a program of the same name in `PATH` if there is one; otherwise they cannot run.

| Program | Source | Installed files |
| --- | --- | --- |
| frp | GitHub fatedier/frp | `frpc`, `frps` |
| cloudflared | GitHub cloudflare/cloudflared | `cloudflared` |
| Tailscale | pkgs.tailscale.com static packages, Linux only | `tailscale`, `tailscaled` |
| FileBrowser Quantum | GitHub gtsteffaniak/filebrowser | `filebrowser` |

Tailscale has no ready-to-run packages for macOS and Windows; use the official installer and enter its path in the settings. The card tells you when the current platform is unsupported.

## Restore defaults {#reset}

"Restore defaults" in the page header is separate from uninstalling: it stops all processes of the module, resets the settings (including the program path and download proxy) to their defaults and deletes the files Gately generated for it. Installed programs are kept.

| Module | What is cleared |
| --- | --- |
| frp | Client, server and all proxy settings, plus the config files generated under `frp/` |
| Cloudflare Tunnel | All tunnels and program settings |
| Tailscale | All settings and the login state in the default state directory; a running Gately-managed tailscaled logs out first. Custom state directories and an existing system tailscaled are left alone |
| File Browser | All instances and program settings, plus each instance's users, share links and caches; files in the shared directories stay untouched |

The old settings are kept in the [config history](/en/docs/guide/audit), so you can restore them after a mistake. To remove a module completely, uninstall the program first, then restore defaults. The API endpoints are `DELETE /api/{module}/install` (uninstall) and `POST /api/{module}/reset` (restore defaults).

## Program location

Executables are looked up in this order:

1. The path set in the module settings;
2. The `bin/` subdirectory of the configuration directory (where online installs go);
3. `PATH`.

When a path is set, the online-installed version is not used, and the card points this out. You can also put the binary into the configuration directory yourself, for example `./gately-data/bin/frpc` in Docker deployments. Tailscale needs both `tailscale` and `tailscaled`; when you use an existing system tailscaled, keep the installed `tailscale` command at the same version as the daemon.

Tokens and keys are stored with 0600 permissions. The cloudflared token is passed through an environment variable, so it does not appear in the process list.

## Tailscale

- Manages `tailscaled` (userspace networking, container-friendly) or attaches to an existing system daemon;
- Auth key or login link, with Headscale supported;
- Subnet routes and exit nodes;
- Node list and SOCKS5 proxy;
- Fixed WireGuard UDP port, making it easy to allow direct connections through a firewall;
- Extra tailscaled arguments and environment variables. If the host's iptables uses the nf_tables backend, add `TS_DEBUG_FIREWALL_MODE=nftables` to the environment variables, otherwise subnet forwarding will not work.

## frp

- **frpc client**: tcp / udp / http / https / stcp / xtcp proxies; adding or removing one takes effect immediately;
- **frps server**: port allowlist, subdomains, dashboard;
- **frps listen addresses**: "Bind address" sets where the frpc connection port listens, and "Proxy bind address" sets where the HTTP / HTTPS ports and the remote ports of tcp / udp proxies listen; leave either empty to listen on all addresses. To put frps behind a reverse proxy such as Gately, set the bind address to `127.0.0.1`, reverse proxy a site in Web Service to `127.0.0.1:<bind port>`, and set the frpc transport protocol to websocket or wss pointing at that site; leave the proxy bind address empty so the remote ports of tcp proxies stay reachable from outside. When importing a frps config without `proxyBindAddr`, it follows `bindAddr`, just like frps does;
- **Import existing config**: "Import config" in the page header supports legacy `frpc.ini` / `frps.ini` (including the `[common]` section) and frp 0.52+ `toml`. You can choose a file or paste the content. Preview the parsed result and the ignored fields first, then confirm the import. For the client you can choose to append (proxies with the same name stay unchanged) or replace all proxies. Importing does not change the enabled state, and it can also be called through the API at `POST /api/frp/import`.

## Cloudflare Tunnel

- Token tunnels and temporary trycloudflare tunnels;
- Multiple tunnels with automatic restart;
- Shows the public address.

## How to choose

| Scenario | Recommendation |
| --- | --- |
| Build a private network across multiple devices | Tailscale |
| You own a public server and want to map internal services out through it | frp |
| No public IP and no server, and you want to expose a web service quickly | Cloudflare Tunnel |
| Your ISP provides a NAT that allows hole punching | [STUN Traversal](/en/docs/modules/stun) |
