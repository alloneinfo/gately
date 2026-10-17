# Gately Admin API

The Gately Admin API is for batch management through scripts and automation. This English edition is translated from the Chinese reference; docs/api/openapi.json is the authoritative machine-readable description.

Authentication: create a token on the "API Tokens" page of the admin console, and send it in the request header as `Authorization: Bearer gtl_xxxx`.

Permissions: each token has only the permissions selected when it was created, in the form `<module>:read` or `<module>:write`. Write permission includes read permission. The account, two-factor authentication, and token management endpoints, as well as the backup, restore, restart, and system settings endpoints, can only be used with a login session.

Request bodies use `Content-Type: application/json`. Errors are always returned as `{"error": "message"}`.

Website: https://gately.cn

For a machine-readable description, see `openapi.json` in the same directory (OpenAPI 3.0).

## Quick start

```bash
# View system status (requires status:read)
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/status

# List all DDNS tasks (requires ddns:read)
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks

# Sync a DDNS task immediately (requires ddns:write; take the key from the list returned in the previous step)
curl -X POST -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks/<key>/sync
```

## Permissions

| Scope | Name | Available permissions |
| --- | --- | --- |
| `portforward` | Port Forwarding | `portforward:read`, `portforward:write` |
| `ddns` | Dynamic DNS | `ddns:read`, `ddns:write` |
| `stun` | STUN Traversal | `stun:read`, `stun:write` |
| `tailscale` | Tailscale Networking | `tailscale:read`, `tailscale:write` |
| `frp` | frp Tunneling | `frp:read`, `frp:write` |
| `cloudflared` | Cloudflare Tunnel | `cloudflared:read`, `cloudflared:write` |
| `ssl` | SSL Certificates | `ssl:read`, `ssl:write` |
| `waf` | WAF Firewall | `waf:read`, `waf:write` |
| `webservice` | Web Services | `webservice:read`, `webservice:write` |
| `wol` | Wake-on-LAN | `wol:read`, `wol:write` |
| `webterminal` | Web Terminal | `webterminal:read`, `webterminal:write` |
| `cron` | Scheduled Tasks | `cron:read`, `cron:write` |
| `docker` | Docker Management | `docker:read`, `docker:write` |
| `webdav` | WebDAV | `webdav:read`, `webdav:write` |
| `ftp` | FTP Service | `ftp:read`, `ftp:write` |
| `storage` | Storage | `storage:read`, `storage:write` |
| `notify` | Notification Center | `notify:read`, `notify:write` |
| `nettools` | Network Diagnostics | `nettools:read`, `nettools:write` |
| `backup` | Configuration Backup | `backup:read`, `backup:write` |
| `status` | System Status | `status:read` |
| `logs` | System Logs | `logs:read` |
| `import` | Lucky Configuration Import | `import:write` |

Wildcards: `*:read` means read-only access to all modules, and `*:write` or `*` means read-write access to all modules.

## Port Forwarding

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/portforward/logs` | `portforward:read` | View the module runtime log |
| DELETE | `/api/portforward/logs` | `portforward:write` | Clear the module runtime log |
| GET | `/api/portforward/rules` | `portforward:read` | List port forwarding rules |
| POST | `/api/portforward/rules` | `portforward:write` | Create a port forwarding rule |
| PUT | `/api/portforward/rules/{key}` | `portforward:write` | Update a port forwarding rule |
| DELETE | `/api/portforward/rules/{key}` | `portforward:write` | Delete a port forwarding rule |
| PUT | `/api/portforward/rules/{key}/enable` | `portforward:write` | Enable or disable a port forwarding rule |
| PUT | `/api/portforward/rules-order` | `portforward:write` | Reorder port forwarding rules |
| GET | `/api/portforward/settings` | `portforward:read` | Read the global port forwarding settings |
| PUT | `/api/portforward/settings` | `portforward:write` | Save the global port forwarding settings |

## Dynamic DNS

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/ddns/logs` | `ddns:read` | View the module runtime log |
| DELETE | `/api/ddns/logs` | `ddns:write` | Clear the module runtime log |
| GET | `/api/ddns/tasks` | `ddns:read` | List DDNS tasks |
| POST | `/api/ddns/tasks` | `ddns:write` | Create a DDNS task |
| PUT | `/api/ddns/tasks/{key}` | `ddns:write` | Update a DDNS task |
| DELETE | `/api/ddns/tasks/{key}` | `ddns:write` | Delete a DDNS task |
| PUT | `/api/ddns/tasks/{key}/enable` | `ddns:write` | Enable or disable a DDNS task |
| PUT | `/api/ddns/tasks-order` | `ddns:write` | Reorder DDNS tasks |
| POST | `/api/ddns/tasks/{key}/sync` | `ddns:write` | Sync a DDNS task immediately |
| GET | `/api/ddns/providers` | `ddns:read` | List supported DNS providers and the parameters they require |
| GET | `/api/ddns/netinterfaces` | `ddns:read` | List local network interfaces and their addresses |
| POST | `/api/ddns/iptest` | `ddns:write` | Test public IP detection |
| GET | `/api/ddns/defaults` | `ddns:read` | Read the DDNS defaults |
| GET | `/api/ddns/settings` | `ddns:read` | Read the global DDNS settings |
| PUT | `/api/ddns/settings` | `ddns:write` | Save the global DDNS settings |

## STUN Traversal

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/stun/logs` | `stun:read` | View the module runtime log |
| DELETE | `/api/stun/logs` | `stun:write` | Clear the module runtime log |
| GET | `/api/stun/rules` | `stun:read` | List STUN rules |
| POST | `/api/stun/rules` | `stun:write` | Create a STUN rule |
| PUT | `/api/stun/rules/{key}` | `stun:write` | Update a STUN rule |
| DELETE | `/api/stun/rules/{key}` | `stun:write` | Delete a STUN rule |
| PUT | `/api/stun/rules/{key}/enable` | `stun:write` | Enable or disable a STUN rule |
| PUT | `/api/stun/rules-order` | `stun:write` | Reorder STUN rules |
| POST | `/api/stun/rules/{key}/restart` | `stun:write` | Restart a STUN rule |
| GET | `/api/stun/netinterfaces` | `stun:read` | List local network interfaces and their addresses |
| GET | `/api/stun/settings` | `stun:read` | Read the global STUN settings |
| PUT | `/api/stun/settings` | `stun:write` | Save the global STUN settings |

## Tailscale Networking

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/tailscale/logs` | `tailscale:read` | View the module runtime log |
| DELETE | `/api/tailscale/logs` | `tailscale:write` | Clear the module runtime log |
| GET | `/api/tailscale/settings` | `tailscale:read` | Read the Tailscale settings |
| PUT | `/api/tailscale/settings` | `tailscale:write` | Save the Tailscale settings |
| GET | `/api/tailscale/status` | `tailscale:read` | View the Tailscale connection status |
| POST | `/api/tailscale/restart` | `tailscale:write` | Restart Tailscale |
| POST | `/api/tailscale/logout` | `tailscale:write` | Log out of Tailscale |

## frp Tunneling

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/frp/logs` | `frp:read` | View the module runtime log |
| DELETE | `/api/frp/logs` | `frp:write` | Clear the module runtime log |
| GET | `/api/frp/proxies` | `frp:read` | List entries |
| POST | `/api/frp/proxies` | `frp:write` | Create an entry |
| PUT | `/api/frp/proxies/{key}` | `frp:write` | Update an entry |
| DELETE | `/api/frp/proxies/{key}` | `frp:write` | Delete an entry |
| PUT | `/api/frp/proxies/{key}/enable` | `frp:write` | Enable or disable an entry |
| PUT | `/api/frp/proxies-order` | `frp:write` | Reorder entries |
| POST | `/api/frp/import` | `frp:write` | Import an existing frpc / frps configuration (legacy ini and new toml are supported); when apply is false, only a preview is returned |
| GET | `/api/frp/client` | `frp:read` | Read the frp client configuration |
| PUT | `/api/frp/client` | `frp:write` | Save the frp client configuration |
| GET | `/api/frp/server` | `frp:read` | Read the frp server configuration |
| PUT | `/api/frp/server` | `frp:write` | Save the frp server configuration |

## Cloudflare Tunnel

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/cloudflared/logs` | `cloudflared:read` | View the module runtime log |
| DELETE | `/api/cloudflared/logs` | `cloudflared:write` | Clear the module runtime log |
| GET | `/api/cloudflared/tunnels` | `cloudflared:read` | List Cloudflare tunnels |
| POST | `/api/cloudflared/tunnels` | `cloudflared:write` | Create a Cloudflare tunnel |
| PUT | `/api/cloudflared/tunnels/{key}` | `cloudflared:write` | Update a Cloudflare tunnel |
| DELETE | `/api/cloudflared/tunnels/{key}` | `cloudflared:write` | Delete a Cloudflare tunnel |
| PUT | `/api/cloudflared/tunnels/{key}/enable` | `cloudflared:write` | Enable or disable a Cloudflare tunnel |
| PUT | `/api/cloudflared/tunnels-order` | `cloudflared:write` | Reorder Cloudflare tunnels |
| POST | `/api/cloudflared/tunnels/{key}/restart` | `cloudflared:write` | Restart a Cloudflare tunnel |
| GET | `/api/cloudflared/settings` | `cloudflared:read` | Read the Cloudflare Tunnel settings |
| PUT | `/api/cloudflared/settings` | `cloudflared:write` | Save the Cloudflare Tunnel settings |

## SSL Certificates

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/ssl/logs` | `ssl:read` | View the module runtime log |
| DELETE | `/api/ssl/logs` | `ssl:write` | Clear the module runtime log |
| GET | `/api/ssl/certs` | `ssl:read` | List certificates |
| POST | `/api/ssl/certs` | `ssl:write` | Create a certificate |
| PUT | `/api/ssl/certs/{key}` | `ssl:write` | Update a certificate |
| DELETE | `/api/ssl/certs/{key}` | `ssl:write` | Delete a certificate |
| PUT | `/api/ssl/certs/{key}/enable` | `ssl:write` | Enable or disable a certificate |
| PUT | `/api/ssl/certs-order` | `ssl:write` | Reorder certificates |
| POST | `/api/ssl/certs/{key}/renew` | `ssl:write` | Issue or renew a certificate immediately |
| GET | `/api/ssl/certs/{key}/download` | `ssl:write` | Download the certificate and private key |
| GET | `/api/ssl/providers` | `ssl:read` | List supported DNS validation providers |
| GET | `/api/ssl/settings` | `ssl:read` | Read the global certificate settings |
| PUT | `/api/ssl/settings` | `ssl:write` | Save the global certificate settings |

## WAF Firewall

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/waf/logs` | `waf:read` | View the module runtime log |
| DELETE | `/api/waf/logs` | `waf:write` | Clear the module runtime log |
| GET | `/api/waf/instances` | `waf:read` | List WAF instances |
| POST | `/api/waf/instances` | `waf:write` | Create a WAF instance |
| PUT | `/api/waf/instances/{key}` | `waf:write` | Update a WAF instance |
| DELETE | `/api/waf/instances/{key}` | `waf:write` | Delete a WAF instance |
| PUT | `/api/waf/instances/{key}/enable` | `waf:write` | Enable or disable a WAF instance |
| PUT | `/api/waf/instances-order` | `waf:write` | Reorder WAF instances |
| GET | `/api/waf/crs` | `waf:read` | List OWASP CRS rule files |
| GET | `/api/waf/crs/{file}` | `waf:read` | Read a CRS rule file |

## Web Services

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/webservice/logs` | `webservice:read` | View the module runtime log |
| DELETE | `/api/webservice/logs` | `webservice:write` | Clear the module runtime log |
| GET | `/api/webservice/listeners` | `webservice:read` | List Web Services listeners |
| POST | `/api/webservice/listeners` | `webservice:write` | Create a Web Services listener |
| PUT | `/api/webservice/listeners/{key}` | `webservice:write` | Update a Web Services listener |
| DELETE | `/api/webservice/listeners/{key}` | `webservice:write` | Delete a Web Services listener |
| PUT | `/api/webservice/listeners/{key}/enable` | `webservice:write` | Enable or disable a Web Services listener |
| PUT | `/api/webservice/listeners-order` | `webservice:write` | Reorder Web Services listeners |
| GET | `/api/webservice/settings` | `webservice:read` | Read the global Web Services settings |
| PUT | `/api/webservice/settings` | `webservice:write` | Save the global Web Services settings |
| GET | `/api/webservice/stats` | `webservice:read` | Per-minute site statistics (?listener=&site=&minutes=) |
| GET | `/api/webservice/metrics` | `webservice:read` | Metrics in Prometheus text format |
| GET | `/api/webservice/bans` | `webservice:read` | Currently banned IPs |
| POST | `/api/webservice/bans` | `webservice:write` | Ban an IP manually |
| DELETE | `/api/webservice/bans` | `webservice:write` | Lift a ban (the body is {"key": address}) |
| GET | `/api/webservice/geoip` | `webservice:read` | GeoIP database status |
| POST | `/api/webservice/geoip/upload` | `webservice:write` | Upload a GeoIP database (multipart field file, .mmdb) |
| POST | `/api/webservice/geoip/download` | `webservice:write` | Download a GeoIP database from a URL (.mmdb) |
| DELETE | `/api/webservice/geoip` | `webservice:write` | Remove the GeoIP database |

## Wake-on-LAN

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/wol/logs` | `wol:read` | View the module runtime log |
| DELETE | `/api/wol/logs` | `wol:write` | Clear the module runtime log |
| GET | `/api/wol/devices` | `wol:read` | List wake-up devices |
| POST | `/api/wol/devices` | `wol:write` | Create a wake-up device |
| PUT | `/api/wol/devices/{key}` | `wol:write` | Update a wake-up device |
| DELETE | `/api/wol/devices/{key}` | `wol:write` | Delete a wake-up device |
| PUT | `/api/wol/devices/{key}/enable` | `wol:write` | Enable or disable a wake-up device |
| PUT | `/api/wol/devices-order` | `wol:write` | Reorder wake-up devices |
| POST | `/api/wol/devices/{key}/wake` | `wol:write` | Send a wake-up magic packet to a device |
| GET | `/api/wol/status` | `wol:read` | View device online status |

## Web Terminal

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/webterminal/logs` | `webterminal:read` | View the module runtime log |
| DELETE | `/api/webterminal/logs` | `webterminal:write` | Clear the module runtime log |
| GET | `/api/webterminal/connections` | `webterminal:read` | List terminal connections |
| POST | `/api/webterminal/connections` | `webterminal:write` | Create a terminal connection |
| PUT | `/api/webterminal/connections/{key}` | `webterminal:write` | Update a terminal connection |
| DELETE | `/api/webterminal/connections/{key}` | `webterminal:write` | Delete a terminal connection |
| PUT | `/api/webterminal/connections/{key}/enable` | `webterminal:write` | Enable or disable a terminal connection |
| PUT | `/api/webterminal/connections-order` | `webterminal:write` | Reorder terminal connections |
| DELETE | `/api/webterminal/connections/{key}/hostkey` | `webterminal:write` | Clear the recorded host key |
| POST | `/api/webterminal/connections/{key}/test` | `webterminal:write` | Test a saved connection |
| POST | `/api/webterminal/connections/test` | `webterminal:write` | Test ad-hoc connection parameters |
| PUT | `/api/webterminal/connections/{key}/quickaccess` | `webterminal:write` | Set a connection's quick-access entries |
| GET | `/api/webterminal/connections/{key}/ws` | `webterminal:write` | Open a terminal session (WebSocket) |
| GET | `/api/webterminal/sessions/{id}/ws` | `webterminal:write` | Attach to an existing terminal session (WebSocket) |
| GET | `/api/webterminal/sessions` | `webterminal:read` | List terminal sessions |
| GET | `/api/webterminal/sessions/{id}` | `webterminal:read` | View terminal session details |
| PUT | `/api/webterminal/sessions/{id}/remark` | `webterminal:write` | Update a terminal session's remark |
| GET | `/api/webterminal/sessions/{id}/stats` | `webterminal:read` | View terminal session statistics |
| DELETE | `/api/webterminal/sessions/{id}` | `webterminal:write` | Close a terminal session |
| GET | `/api/webterminal/security` | Login session only | View the terminal security settings status |
| POST | `/api/webterminal/security/agreement` | Login session only | Accept the terminal user agreement |
| POST | `/api/webterminal/security/2fa/setup` | Login session only | Generate a terminal two-factor secret |
| POST | `/api/webterminal/security/2fa/enable` | Login session only | Enable terminal two-factor authentication |
| POST | `/api/webterminal/security/2fa/disable` | Login session only | Disable terminal two-factor authentication |
| POST | `/api/webterminal/security/verify2fa` | Login session only | Verify the terminal two-factor code |
| GET | `/api/webterminal/security/check2fa` | Login session only | Check whether terminal two-factor verification has passed |
| GET | `/api/webterminal/sftp/{key}/list` | `webterminal:read` | List a remote directory |
| GET | `/api/webterminal/sftp/{key}/read` | `webterminal:write` | Read a remote file |
| POST | `/api/webterminal/sftp/{key}/write` | `webterminal:write` | Write a remote file |
| POST | `/api/webterminal/sftp/{key}/upload` | `webterminal:write` | Upload files to the remote host |
| POST | `/api/webterminal/sftp/{key}/mkdir` | `webterminal:write` | Create a remote directory |
| POST | `/api/webterminal/sftp/{key}/touch` | `webterminal:write` | Create an empty remote file |
| POST | `/api/webterminal/sftp/{key}/rename` | `webterminal:write` | Rename or move a remote file |
| POST | `/api/webterminal/sftp/{key}/copy` | `webterminal:write` | Copy remote files |
| POST | `/api/webterminal/sftp/{key}/chmod` | `webterminal:write` | Change remote file permissions |
| DELETE | `/api/webterminal/sftp/{key}/remove` | `webterminal:write` | Delete remote files |
| POST | `/api/webterminal/sftp/{key}/compress` | `webterminal:write` | Compress files on the remote host |
| POST | `/api/webterminal/sftp/{key}/decompress` | `webterminal:write` | Extract files on the remote host |
| GET | `/api/webterminal/sftp/{key}/preview-archive` | `webterminal:write` | Preview the contents of a remote archive |
| GET | `/api/webterminal/settings` | `webterminal:read` | Read the Web Terminal settings |
| PUT | `/api/webterminal/settings` | `webterminal:write` | Save the Web Terminal settings |
| GET | `/api/webterminal/shells` | `webterminal:read` | List shells available on this machine |
| GET | `/api/webterminal/globalshortcuts` | `webterminal:read` | Read the global shortcut commands |
| PUT | `/api/webterminal/globalshortcuts` | `webterminal:write` | Save the global shortcut commands |
| GET | `/api/webterminal/splitlayout` | `webterminal:read` | Read the split-pane layout |
| PUT | `/api/webterminal/splitlayout` | `webterminal:write` | Save the split-pane layout |
| DELETE | `/api/webterminal/splitlayout` | `webterminal:write` | Clear the split-pane layout |

## Scheduled Tasks

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/cron/logs` | `cron:read` | View the module runtime log |
| DELETE | `/api/cron/logs` | `cron:write` | Clear the module runtime log |
| GET | `/api/cron/tasks` | `cron:read` | List scheduled tasks |
| POST | `/api/cron/tasks` | `cron:write` | Create a scheduled task |
| PUT | `/api/cron/tasks/{key}` | `cron:write` | Update a scheduled task |
| DELETE | `/api/cron/tasks/{key}` | `cron:write` | Delete a scheduled task |
| PUT | `/api/cron/tasks/{key}/enable` | `cron:write` | Enable or disable a scheduled task |
| PUT | `/api/cron/tasks-order` | `cron:write` | Reorder scheduled tasks |
| POST | `/api/cron/tasks/{key}/run` | `cron:write` | Run a scheduled task immediately |
| POST | `/api/cron/parse` | `cron:write` | Parse and preview a Cron expression |

## Docker Management

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/docker/logs` | `docker:read` | View the module runtime log |
| DELETE | `/api/docker/logs` | `docker:write` | Clear the module runtime log |
| GET | `/api/docker/tasks` | `docker:read` | List background tasks |
| GET | `/api/docker/tasks/{id}` | `docker:read` | View background task progress and output |
| DELETE | `/api/docker/tasks` | `docker:write` | Clear finished background tasks |
| GET | `/api/docker/images` | `docker:read` | List images |
| POST | `/api/docker/images/pull` | `docker:write` | Pull an image |
| POST | `/api/docker/images/push` | `docker:write` | Push an image |
| POST | `/api/docker/images/search` | `docker:write` | Search for images |
| POST | `/api/docker/images/load` | `docker:write` | Load an image file |
| POST | `/api/docker/images/build` | `docker:write` | Build an image |
| GET | `/api/docker/images/{id}` | `docker:read` | View image details |
| POST | `/api/docker/images/{id}/tag` | `docker:write` | Tag an image |
| DELETE | `/api/docker/images/{id}` | `docker:write` | Delete an image |
| GET | `/api/docker/images/{id}/export` | `docker:write` | Export an image |
| GET | `/api/docker/networks` | `docker:read` | List networks |
| POST | `/api/docker/networks` | `docker:write` | Create a network |
| DELETE | `/api/docker/networks/{id}` | `docker:write` | Delete a network |
| POST | `/api/docker/networks/{id}/connect` | `docker:write` | Connect a container to a network |
| POST | `/api/docker/networks/{id}/disconnect` | `docker:write` | Disconnect a container from a network |
| GET | `/api/docker/volumes` | `docker:read` | List volumes |
| POST | `/api/docker/volumes` | `docker:write` | Create a volume |
| DELETE | `/api/docker/volumes/{name}` | `docker:write` | Delete a volume |
| GET | `/api/docker/volumes/{name}/export` | `docker:write` | Export a volume |
| POST | `/api/docker/volumes/{name}/import` | `docker:write` | Import data into a volume |
| GET | `/api/docker/compose/projects` | `docker:read` | List Compose projects |
| POST | `/api/docker/compose/projects` | `docker:write` | Create a Compose project |
| POST | `/api/docker/compose/{project}/up` | `docker:write` | Start a Compose project |
| POST | `/api/docker/compose/{project}/pull` | `docker:write` | Pull a Compose project's images |
| POST | `/api/docker/compose/{project}/{action}` | `docker:write` | Perform an action on a Compose project |
| GET | `/api/docker/compose/{project}/config` | `docker:write` | Read a Compose configuration file |
| PUT | `/api/docker/compose/{project}/config` | `docker:write` | Save a Compose configuration file |
| GET | `/api/docker/compose/{project}/logs` | `docker:read` | View Compose project logs |
| DELETE | `/api/docker/compose/{project}` | `docker:write` | Delete a Compose project |
| GET | `/api/docker/info` | `docker:read` | View Docker engine information |
| GET | `/api/docker/disk-usage` | `docker:read` | View Docker disk usage |
| POST | `/api/docker/prune` | `docker:write` | Clean up unused Docker resources |
| GET | `/api/docker/settings` | `docker:read` | Read the Docker module settings |
| PUT | `/api/docker/settings` | `docker:write` | Save the Docker module settings |
| GET | `/api/docker/containers` | `docker:read` | List containers |
| POST | `/api/docker/containers` | `docker:write` | Create a container |
| POST | `/api/docker/containers/check-all` | `docker:write` | Check all containers for image updates |
| POST | `/api/docker/containers/upgrade-batch` | `docker:write` | Upgrade containers in batch |
| POST | `/api/docker/images/upgrade-check` | `docker:write` | Check whether images have new versions |
| GET | `/api/docker/containers/{id}` | `docker:read` | View container details |
| DELETE | `/api/docker/containers/{id}` | `docker:write` | Delete a container |
| POST | `/api/docker/containers/{id}/edit` | `docker:write` | Change a container's configuration and recreate it |
| POST | `/api/docker/containers/{id}/rename` | `docker:write` | Rename a container |
| POST | `/api/docker/containers/{id}/commit` | `docker:write` | Commit a container as an image |
| POST | `/api/docker/containers/{id}/check` | `docker:write` | Check whether a container's image has an update |
| POST | `/api/docker/containers/{id}/upgrade` | `docker:write` | Upgrade a container's image and recreate it |
| POST | `/api/docker/containers/{id}/{action}` | `docker:write` | Perform an action on a container |
| GET | `/api/docker/containers/{id}/logs` | `docker:read` | View container logs |
| GET | `/api/docker/containers/{id}/stats` | `docker:read` | View container resource usage |
| GET | `/api/docker/containers/{id}/processes` | `docker:read` | View processes inside a container |
| GET | `/api/docker/containers/{id}/files/list` | `docker:read` | List a directory inside a container |
| GET | `/api/docker/containers/{id}/files/read` | `docker:write` | Read a file inside a container |
| GET | `/api/docker/containers/{id}/files/download` | `docker:write` | Download a file from a container |
| POST | `/api/docker/containers/{id}/files/write` | `docker:write` | Write a file inside a container |
| POST | `/api/docker/containers/{id}/files/mkdir` | `docker:write` | Create a directory inside a container |
| POST | `/api/docker/containers/{id}/files/rename` | `docker:write` | Rename a file inside a container |
| POST | `/api/docker/containers/{id}/files/upload` | `docker:write` | Upload files to a container |
| DELETE | `/api/docker/containers/{id}/files` | `docker:write` | Delete a file inside a container |
| GET | `/api/docker/containers/{id}/exec` | `docker:write` | Open a terminal inside a container (WebSocket) |

## WebDAV

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/webdav/logs` | `webdav:read` | View the module runtime log |
| DELETE | `/api/webdav/logs` | `webdav:write` | Clear the module runtime log |
| GET | `/api/webdav/settings` | `webdav:read` | Read the WebDAV settings |
| PUT | `/api/webdav/settings` | `webdav:write` | Save the WebDAV settings |

## FTP Service

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/ftp/logs` | `ftp:read` | View the module runtime log |
| DELETE | `/api/ftp/logs` | `ftp:write` | Clear the module runtime log |
| GET | `/api/ftp/settings` | `ftp:read` | Read the FTP settings |
| PUT | `/api/ftp/settings` | `ftp:write` | Save the FTP settings |

## Storage

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/storage/logs` | `storage:read` | View the module runtime log |
| DELETE | `/api/storage/logs` | `storage:write` | Clear the module runtime log |
| GET | `/api/storage/mounts` | `storage:read` | List entries |
| POST | `/api/storage/mounts` | `storage:write` | Create an entry |
| PUT | `/api/storage/mounts/{key}` | `storage:write` | Update an entry |
| DELETE | `/api/storage/mounts/{key}` | `storage:write` | Delete an entry |
| PUT | `/api/storage/mounts/{key}/enable` | `storage:write` | Enable or disable an entry |
| PUT | `/api/storage/mounts-order` | `storage:write` | Reorder entries |
| GET | `/api/storage/files/{key}/list` | `storage:read` | List a storage directory |
| GET | `/api/storage/files/{key}/download` | `storage:write` | Download a storage file |
| POST | `/api/storage/files/{key}/upload` | `storage:write` | Upload files to storage |
| POST | `/api/storage/files/{key}/mkdir` | `storage:write` | Create a storage directory |
| POST | `/api/storage/files/{key}/rename` | `storage:write` | Rename or move a storage file |
| DELETE | `/api/storage/files/{key}` | `storage:write` | Delete a storage file |

## Notification Center

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/notify/logs` | `notify:read` | View the module runtime log |
| DELETE | `/api/notify/logs` | `notify:write` | Clear the module runtime log |
| GET | `/api/notify/channels` | `notify:read` | List notification channels |
| POST | `/api/notify/channels` | `notify:write` | Create a notification channel |
| PUT | `/api/notify/channels/{key}` | `notify:write` | Update a notification channel |
| DELETE | `/api/notify/channels/{key}` | `notify:write` | Delete a notification channel |
| PUT | `/api/notify/channels/{key}/enable` | `notify:write` | Enable or disable a notification channel |
| PUT | `/api/notify/channels-order` | `notify:write` | Reorder notification channels |
| GET | `/api/notify/rules` | `notify:read` | List notification rules |
| POST | `/api/notify/rules` | `notify:write` | Create a notification rule |
| PUT | `/api/notify/rules/{key}` | `notify:write` | Update a notification rule |
| DELETE | `/api/notify/rules/{key}` | `notify:write` | Delete a notification rule |
| PUT | `/api/notify/rules/{key}/enable` | `notify:write` | Enable or disable a notification rule |
| PUT | `/api/notify/rules-order` | `notify:write` | Reorder notification rules |
| GET | `/api/notify/event-types` | `notify:read` | Event types that can be subscribed to |
| GET | `/api/notify/history` | `notify:read` | Recent delivery records |
| POST | `/api/notify/channels/{key}/test` | `notify:write` | Send a test notification to a channel |

## Network Diagnostics

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/nettools/logs` | `nettools:read` | View the module runtime log |
| DELETE | `/api/nettools/logs` | `nettools:write` | Clear the module runtime log |
| POST | `/api/nettools/run` | `nettools:write` | Run a diagnostic tool and return a task (tool: ping / traceroute / tcp / dns / speed / scan) |
| GET | `/api/nettools/tasks` | `nettools:read` | Recent diagnostic tasks |
| GET | `/api/nettools/tasks/{id}` | `nettools:read` | Task details and output |
| DELETE | `/api/nettools/tasks/{id}` | `nettools:write` | Cancel a running task, or delete a finished task |
| GET | `/api/nettools/settings` | `nettools:read` | Diagnostic settings |
| PUT | `/api/nettools/settings` | `nettools:write` | Update the probe deny list |
| GET | `/api/nettools/networks` | `nettools:read` | Network segments on this machine that can be used for LAN scanning |

## Configuration Backup

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/backup/logs` | `backup:read` | View the module runtime log |
| DELETE | `/api/backup/logs` | `backup:write` | Clear the module runtime log |
| GET | `/api/backup/targets` | `backup:read` | List backup destinations |
| POST | `/api/backup/targets` | `backup:write` | Create a backup destination |
| PUT | `/api/backup/targets/{key}` | `backup:write` | Update a backup destination |
| DELETE | `/api/backup/targets/{key}` | `backup:write` | Delete a backup destination |
| PUT | `/api/backup/targets/{key}/enable` | `backup:write` | Enable or disable a backup destination |
| PUT | `/api/backup/targets-order` | `backup:write` | Reorder backup destinations |
| GET | `/api/backup/settings` | `backup:read` | Backup settings |
| PUT | `/api/backup/settings` | `backup:write` | Update the backup settings (send the placeholder for the passphrase to keep it unchanged) |
| POST | `/api/backup/run` | `backup:write` | Run a backup now |
| GET | `/api/backup/status` | `backup:read` | Result of the most recent backup and the next scheduled run time |
| GET | `/api/backup/mounts` | `backup:read` | Storage mount points that can be used as backup destinations |
| GET | `/api/backup/local` | Login session only | List local backup files |
| GET | `/api/backup/local/{name}` | Login session only | Download a local backup file |

## System

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/session` | Any valid token | View the current authenticated identity |
| GET | `/api/status` | `status:read` | View the status of the system and each module |
| GET | `/api/logs` | `logs:read` | View the system log |
| GET | `/api/system/settings` | Login session only | Read the global settings |
| PUT | `/api/system/settings` | Login session only | Save the global settings |
| GET | `/api/system/info` | `status:read` | View system information |
| GET | `/api/system/update` | `status:read` | Check for a new version |
| GET | `/api/system/backup` | Login session only | Download a configuration backup |
| POST | `/api/system/restore` | Login session only | Restore the configuration from a backup |
| POST | `/api/system/restart` | Login session only | Restart Gately |
| GET | `/api/openapi.json` | Any valid token | Get the OpenAPI description of this API |

## Configuration Import

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| POST | `/api/import/lucky` | `import:write` | Import a Lucky configuration |

## API Tokens

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| GET | `/api/tokens` | Login session only | List API tokens |
| GET | `/api/tokens/scopes` | Login session only | List the scopes that can be granted |
| POST | `/api/tokens` | Login session only | Create an API token |
| PUT | `/api/tokens/{id}` | Login session only | Update an API token's name, permissions, expiry, and enabled state |
| POST | `/api/tokens/{id}/regenerate` | Login session only | Regenerate the token's plaintext |
| DELETE | `/api/tokens/{id}` | Login session only | Delete an API token |

## Account

| Method | Path | Required permission | Description |
| --- | --- | --- | --- |
| PUT | `/api/account` | Login session only | Change the administrator username and password |
| GET | `/api/account/totp` | Login session only | View the two-factor authentication status |
| POST | `/api/account/totp/setup` | Login session only | Generate a two-factor secret |
| POST | `/api/account/totp/enable` | Login session only | Enable two-factor authentication |
| POST | `/api/account/totp/disable` | Login session only | Disable two-factor authentication |

## Data structures

### backup.LocalFile

| Field | Type |
| --- | --- |
| `modTime` | time (RFC 3339) |
| `name` | string |
| `size` | integer |

### backup.Result

| Field | Type |
| --- | --- |
| `error` | string |
| `file` | string |
| `ok` | boolean |
| `pruned` | integer |
| `target` | string |

### backup.Settings

| Field | Type |
| --- | --- |
| `enable` | boolean |
| `keep` | integer |
| `localEnable` | boolean |
| `password` | string |
| `schedule` | string |

### backup.Status

| Field | Type |
| --- | --- |
| `lastRun` | time (RFC 3339) |
| `manual` | boolean |
| `nextRun` | time (RFC 3339) |
| `results` | array of `backup.Result` |
| `running` | boolean |

### backup.Target

| Field | Type |
| --- | --- |
| `dir` | string |
| `enable` | boolean |
| `key` | string |
| `mount` | string |
| `name` | string |

### backup.settingsView

| Field | Type |
| --- | --- |
| `enable` | boolean |
| `hasPassword` | boolean |
| `keep` | integer |
| `localEnable` | boolean |
| `password` | string |
| `schedule` | string |

### cloudflared.Tunnel

| Field | Type |
| --- | --- |
| `edgeIpVersion` | string |
| `enable` | boolean |
| `key` | string |
| `mode` | string |
| `name` | string |
| `protocol` | string |
| `remark` | string |
| `service` | string |
| `token` | string |

### cron.Task

| Field | Type |
| --- | --- |
| `command` | string |
| `enable` | boolean |
| `http` | `webhook.Config` |
| `key` | string |
| `name` | string |
| `remark` | string |
| `spec` | string |
| `type` | string |

### crud.enableRequest

| Field | Type |
| --- | --- |
| `enable` | boolean |

### crud.orderRequest

| Field | Type |
| --- | --- |
| `keys` | array of string |

### ddns.Record

| Field | Type |
| --- | --- |
| `enable` | boolean |
| `key` | string |
| `line` | string |
| `remark` | string |
| `sub` | string |
| `ttl` | integer |
| `type` | string |
| `value` | string |
| `zone` | string |

### ddns.Task

| Field | Type |
| --- | --- |
| `credentials` | `dnsprovider.Credentials` |
| `enable` | boolean |
| `forceInterval` | integer |
| `httpTimeout` | integer |
| `interval` | integer |
| `ipv4` | `ipsource.Source` |
| `ipv6` | `ipsource.Source` |
| `key` | string |
| `name` | string |
| `provider` | string |
| `records` | array of `ddns.Record` |
| `remark` | string |
| `webhook` | `webhook.Config` |

### dnsprovider.Credentials

| Field | Type |
| --- | --- |
| `extra` | map of string |
| `id` | string |
| `secret` | string |

### events.TypeInfo

| Field | Type |
| --- | --- |
| `level` | string |
| `name` | string |
| `type` | string |

### frp.ClientSettings

| Field | Type |
| --- | --- |
| `binPath` | string |
| `enable` | boolean |
| `protocol` | string |
| `serverAddr` | string |
| `serverPort` | integer |
| `token` | string |
| `user` | string |

### frp.ImportRequest

| Field | Type |
| --- | --- |
| `apply` | boolean |
| `content` | string |
| `proxyMode` | string |
| `role` | string |

### frp.ImportResponse

| Field | Type |
| --- | --- |
| `added` | integer |
| `applied` | boolean |
| `client` | `frp.ClientSettings` |
| `proxies` | array of `frp.Proxy` |
| `role` | string |
| `server` | `frp.ServerSettings` |
| `warnings` | array of string |

### frp.Proxy

| Field | Type |
| --- | --- |
| `customDomains` | array of string |
| `enable` | boolean |
| `key` | string |
| `localIp` | string |
| `localPort` | integer |
| `name` | string |
| `remark` | string |
| `remotePort` | integer |
| `secretKey` | string |
| `subdomain` | string |
| `type` | string |

### frp.ServerSettings

| Field | Type |
| --- | --- |
| `allowPorts` | string |
| `binPath` | string |
| `bindPort` | integer |
| `dashboardAddr` | string |
| `dashboardPort` | integer |
| `dashboardPwd` | string |
| `dashboardUser` | string |
| `enable` | boolean |
| `subdomainHost` | string |
| `token` | string |
| `vhostHttpPort` | integer |
| `vhostHttpsPort` | integer |

### geoip.Info

| Field | Type |
| --- | --- |
| `buildTime` | time (RFC 3339) |
| `loaded` | boolean |
| `size` | integer |
| `type` | string |

### ipsource.Source

| Field | Type |
| --- | --- |
| `command` | string |
| `enable` | boolean |
| `interface` | string |
| `regex` | string |
| `type` | string |
| `urls` | array of string |

### nettools.Request

| Field | Type |
| --- | --- |
| `cidr` | string |
| `count` | integer |
| `ports` | string |
| `seconds` | integer |
| `server` | string |
| `target` | string |
| `tool` | string |
| `type` | string |
| `url` | string |

### nettools.Settings

| Field | Type |
| --- | --- |
| `deny` | array of string |

### nettools.Task

| Field | Type |
| --- | --- |
| `error` | string |
| `finished` | time (RFC 3339) |
| `id` | string |
| `lines` | array of string |
| `result` | any |
| `started` | time (RFC 3339) |
| `status` | string |
| `title` | string |
| `tool` | string |

### notify.Channel

| Field | Type |
| --- | --- |
| `enable` | boolean |
| `from` | string |
| `host` | string |
| `key` | string |
| `kind` | string |
| `name` | string |
| `port` | integer |
| `remark` | string |
| `secret` | string |
| `security` | string |
| `target` | string |
| `url` | string |
| `user` | string |
| `webhook` | `webhook.Config` |

### notify.Record

| Field | Type |
| --- | --- |
| `channel` | string |
| `error` | string |
| `ok` | boolean |
| `rule` | string |
| `time` | time (RFC 3339) |
| `title` | string |
| `type` | string |

### notify.Rule

| Field | Type |
| --- | --- |
| `channels` | array of string |
| `enable` | boolean |
| `events` | array of string |
| `key` | string |
| `minLevel` | string |
| `name` | string |
| `silence` | integer |

### portforward.Rule

| Field | Type |
| --- | --- |
| `enable` | boolean |
| `key` | string |
| `listenAddress` | string |
| `listenPorts` | string |
| `maxConnections` | integer |
| `name` | string |
| `protocols` | array of string |
| `remark` | string |
| `targetAddresses` | array of string |
| `targetPorts` | string |
| `udpSessionTimeout` | integer |

### ssl.ACMEConfig

| Field | Type |
| --- | --- |
| `ca` | string |
| `credentials` | `dnsprovider.Credentials` |
| `directoryUrl` | string |
| `disableAuthoritativeCheck` | boolean |
| `domains` | array of string |
| `eabHmac` | string |
| `eabKid` | string |
| `email` | string |
| `keyType` | string |
| `propagationTimeout` | integer |
| `provider` | string |
| `resolvers` | array of string |
| `skipPropagationCheck` | boolean |
| `zone` | string |

### ssl.Cert

| Field | Type |
| --- | --- |
| `acme` | `ssl.ACMEConfig` |
| `certFile` | string |
| `certPem` | string |
| `enable` | boolean |
| `issuerPem` | string |
| `key` | string |
| `keyFile` | string |
| `keyPem` | string |
| `mapping` | `ssl.Mapping` |
| `remark` | string |
| `source` | string |
| `updatedAt` | time (RFC 3339) |

### ssl.Mapping

| Field | Type |
| --- | --- |
| `dir` | string |
| `enable` | boolean |
| `script` | string |

### storage.Mount

| Field | Type |
| --- | --- |
| `enable` | boolean |
| `key` | string |
| `name` | string |
| `password` | string |
| `path` | string |
| `readOnly` | boolean |
| `remark` | string |
| `skipVerify` | boolean |
| `type` | string |
| `url` | string |
| `username` | string |

### storage.MountInfo

| Field | Type |
| --- | --- |
| `enable` | boolean |
| `key` | string |
| `name` | string |
| `readOnly` | boolean |
| `type` | string |

### stun.Bind

| Field | Type |
| --- | --- |
| `interface` | string |
| `ip` | string |
| `mode` | string |
| `regex` | string |

### stun.Rule

| Field | Type |
| --- | --- |
| `autoRetry` | boolean |
| `bind` | `stun.Bind` |
| `disableCheck` | boolean |
| `disableForward` | boolean |
| `enable` | boolean |
| `heartbeatInterval` | integer |
| `keepAliveServers` | array of string |
| `key` | string |
| `localPort` | integer |
| `maxConnections` | integer |
| `name` | string |
| `natpmp` | boolean |
| `natpmpGateway` | string |
| `protocol` | string |
| `remark` | string |
| `retryInterval` | integer |
| `script` | string |
| `scriptEnable` | boolean |
| `stunServers` | array of string |
| `stunTimeout` | integer |
| `targetAddresses` | array of string |
| `targetPort` | integer |
| `udpSessionTimeout` | integer |
| `upnp` | boolean |
| `upnpControlUrl` | string |
| `upnpGateway` | string |
| `upnpInternalPort` | integer |
| `upnpLocalHost` | string |
| `useGlobalServers` | boolean |
| `useGlobalWebhook` | boolean |
| `webhook` | `webhook.Config` |
| `webhookOnlyOnChange` | boolean |

### waf.Instance

| Field | Type |
| --- | --- |
| `enable` | boolean |
| `inboundThreshold` | integer |
| `key` | string |
| `mode` | string |
| `name` | string |
| `outboundThreshold` | integer |
| `paranoia` | integer |
| `remark` | string |
| `ruleSets` | array of `waf.RuleSet` |

### waf.RuleSet

| Field | Type |
| --- | --- |
| `enable` | boolean |
| `param` | string |
| `remark` | string |
| `type` | string |

### webhook.Config

| Field | Type |
| --- | --- |
| `body` | string |
| `enable` | boolean |
| `headers` | array of string |
| `insecureSkipVerify` | boolean |
| `method` | string |
| `successContent` | array of string |
| `timeout` | integer |
| `url` | string |

### webservice.BanInfo

| Field | Type |
| --- | --- |
| `key` | string |
| `manual` | boolean |
| `reason` | string |
| `until` | time (RFC 3339) |

### webservice.CacheConfig

| Field | Type |
| --- | --- |
| `enable` | boolean |
| `maxMb` | integer |
| `ttl` | integer |

### webservice.HealthCheck

| Field | Type |
| --- | --- |
| `enable` | boolean |
| `fails` | integer |
| `interval` | integer |
| `mode` | string |
| `passes` | integer |
| `path` | string |
| `timeout` | integer |

### webservice.Listener

| Field | Type |
| --- | --- |
| `acceptProxyProtocol` | boolean |
| `default` | `webservice.Site` |
| `enable` | boolean |
| `key` | string |
| `listenIp` | string |
| `minTls` | string |
| `name` | string |
| `network` | string |
| `port` | integer |
| `remark` | string |
| `sites` | array of `webservice.Site` |
| `tls` | boolean |

### webservice.PathRule

| Field | Type |
| --- | --- |
| `auth` | string |
| `ipList` | array of string |
| `ipMode` | string |
| `prefix` | string |

### webservice.Point

| Field | Type |
| --- | --- |
| `bytes` | integer |
| `requests` | integer |
| `s4xx` | integer |
| `s5xx` | integer |
| `t` | integer |

### webservice.RateLimit

| Field | Type |
| --- | --- |
| `burst` | integer |
| `enable` | boolean |
| `maxConns` | integer |
| `rate` | number |

### webservice.Rewrite

| Field | Type |
| --- | --- |
| `code` | integer |
| `pattern` | string |
| `redirect` | boolean |
| `to` | string |

### webservice.ShareLink

| Field | Type |
| --- | --- |
| `expiresAt` | time (RFC 3339) |
| `key` | string |
| `maxUses` | integer |
| `name` | string |
| `pathPrefix` | string |
| `token` | string |

### webservice.Site

| Field | Type |
| --- | --- |
| `accessLog` | boolean |
| `authOptional` | boolean |
| `basicAuth` | boolean |
| `cache` | `webservice.CacheConfig` |
| `compress` | boolean |
| `contentType` | string |
| `cors` | boolean |
| `countries` | array of string |
| `countryMode` | string |
| `disableKeepAlives` | boolean |
| `domains` | array of string |
| `enable` | boolean |
| `forwardedHeaders` | boolean |
| `health` | `webservice.HealthCheck` |
| `insecureSkipVerify` | boolean |
| `ipList` | array of string |
| `ipMode` | string |
| `keepPath` | boolean |
| `key` | string |
| `lbPolicy` | string |
| `listDir` | boolean |
| `name` | string |
| `oauthUsers` | array of string |
| `rateLimit` | `webservice.RateLimit` |
| `redirectCode` | integer |
| `redirectTo` | string |
| `remark` | string |
| `requestHeaders` | array of string |
| `responseHeaders` | array of string |
| `rewrites` | array of `webservice.Rewrite` |
| `root` | string |
| `rules` | array of `webservice.PathRule` |
| `sendProxyProtocol` | boolean |
| `shareLinks` | array of `webservice.ShareLink` |
| `statusCode` | integer |
| `targets` | array of string |
| `text` | string |
| `type` | string |
| `useTargetHost` | boolean |
| `users` | array of string |
| `waf` | string |
| `webAuth` | boolean |

### webservice.SiteStatsView

| Field | Type |
| --- | --- |
| `bytesOut` | integer |
| `series` | array of `webservice.Point` |
| `status` | map of integer |
| `topCountries` | array of `webservice.TopItem` |
| `topIps` | array of `webservice.TopItem` |
| `upstreams` | array of `webservice.UpstreamStatus` |

### webservice.TopItem

| Field | Type |
| --- | --- |
| `count` | integer |
| `key` | string |

### webservice.UpstreamStatus

| Field | Type |
| --- | --- |
| `backup` | boolean |
| `conns` | integer |
| `healthy` | boolean |
| `url` | string |
| `weight` | integer |

### webservice.banRequest

| Field | Type |
| --- | --- |
| `key` | string |
| `seconds` | integer |

### webservice.geoipDownload

| Field | Type |
| --- | --- |
| `url` | string |

### webterminal.Connection

| Field | Type |
| --- | --- |
| `enable` | boolean |
| `key` | string |
| `local` | `webterminal.LocalConfig` |
| `name` | string |
| `quickAccessDirs` | array of string |
| `remark` | string |
| `ssh` | `webterminal.SSHConfig` |
| `telnet` | `webterminal.TelnetConfig` |
| `type` | string |

### webterminal.LocalConfig

| Field | Type |
| --- | --- |
| `env` | map of string |
| `shell` | string |
| `workDir` | string |

### webterminal.Proxy

| Field | Type |
| --- | --- |
| `enabled` | boolean |
| `host` | string |
| `password` | string |
| `port` | integer |
| `type` | string |
| `username` | string |

### webterminal.SSHConfig

| Field | Type |
| --- | --- |
| `authType` | string |
| `host` | string |
| `hostKey` | string |
| `keepAlive` | integer |
| `keyFile` | string |
| `passphrase` | string |
| `password` | string |
| `port` | integer |
| `privateKey` | string |
| `proxy` | `webterminal.Proxy` |
| `timeout` | integer |
| `username` | string |
| `workDir` | string |

### webterminal.TelnetConfig

| Field | Type |
| --- | --- |
| `host` | string |
| `password` | string |
| `port` | integer |
| `proxy` | `webterminal.Proxy` |
| `timeout` | integer |
| `username` | string |

### wol.Device

| Field | Type |
| --- | --- |
| `broadcastIps` | array of string |
| `checkAddr` | string |
| `enable` | boolean |
| `key` | string |
| `macs` | array of string |
| `name` | string |
| `port` | integer |
| `remark` | string |
| `repeat` | integer |

