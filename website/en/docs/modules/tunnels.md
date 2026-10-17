# Tailscale / frp / cloudflared

These three modules integrate the official programs as managed processes: automatic restart, logs collected in the UI, and configuration done in the UI. Gately does not bundle these programs, so you can always use the latest upstream release.

## Program location

Executables are looked up in this order:

1. The path set in the module settings;
2. `PATH`;
3. The `bin/` subdirectory of the configuration directory.

For Docker deployments, place the binary in the mounted configuration directory, for example `./gately-data/bin/frpc`. Tailscale needs both `tailscale` and `tailscaled`.

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
- **Import existing config**: "Import config" in the page header supports legacy `frpc.ini` / `frps.ini` (including the `[common]` section) and frp 0.52+ `toml`. You can choose a file or paste the content. Preview the parsed result and the ignored fields first, then confirm the import. For the client you can choose to append (proxies with the same name stay unchanged) or replace all proxies. Importing does not change the enabled state, and it can also be called through the API at `POST /api/frp/import`.
- **Migrating from Lucky**: the first client instance and the first server instance in `lucky_frp.lkcf` can be imported directly. See [Migrating from Lucky](/en/docs/guide/migration).

## Cloudflare Tunnel

- Token tunnels and temporary trycloudflare tunnels;
- Multiple tunnels with automatic restart;
- Shows the public address.
- Tunnels that use a tunnel token can be imported from Lucky.

## How to choose

| Scenario | Recommendation |
| --- | --- |
| Build a private network across multiple devices | Tailscale |
| You own a public server and want to map internal services out through it | frp |
| No public IP and no server, and you want to expose a web service quickly | Cloudflare Tunnel |
| Your ISP provides a NAT that allows hole punching | [STUN Traversal](/en/docs/modules/stun) |
