# File and Ops Tools

## WebDAV

Configure it under "WebDAV Service" in the sidebar "File Services" group.

| Field | Default | Description |
| --- | --- | --- |
| `enable` | Off | Enable the service; at least one user is required when enabled |
| `listenIp` | Empty | Listen address; leave empty to listen on all addresses |
| `port` | 18800 | Listen port |
| `tls` | Off | Enable HTTPS; the certificate is provided per SNI by the [SSL Certificates](/en/docs/modules/ssl) module |
| `users` | None | Multiple users. Each has `name`, `password`, `root` (an absolute directory path) and `readOnly`, or uses `mounts` to mount several directories |

Each user can map to one directory, or mount several directories:

- **Single directory**: set `root` and `readOnly`. After login, the root is that directory.
- **Multiple directories**: click "Mount multiple directories" in the UI, or fill in several entries in `mounts`. Each entry has `name` (display name, cannot contain `/`, unique within one user), `path` (an absolute path) and `readOnly`. After login, the root lists the directories by name. The root itself is not writable, and files cannot be moved between different directories.

When migrating from Lucky, a user's mount list is converted into multiple directories, and users that appear more than once in Lucky with the same name and password are merged into one user.

## FTP

Configure it under "FTP Service" in the "File Services" group. FTP and FTPS are supported.

| Field | Default | Description |
| --- | --- | --- |
| `enable` | Off | Enable the service; at least one user is required when enabled |
| `listenIp` | Empty | Listen address; leave empty to listen on all addresses |
| `port` | 2121 | Listen port |
| `passiveStart` / `passiveEnd` | 0 | Passive-mode data port range. When both are 0 the system assigns ports randomly; when set, 1 ≤ start ≤ end ≤ 65535 must hold, and the range must be allowed in the firewall and port mappings |
| `publicHost` | Empty | IP advertised externally in passive mode. Behind NAT, enter the public or forwarded IP. Only IP addresses are supported; leave empty to use the connection address automatically |
| `tlsMode` | 0 | 0 allows both plain text and FTPS, 1 requires explicit FTPS (`AUTH TLS`), 2 is implicit FTPS (encrypted from connect). When non-zero, the certificate is provided per SNI by the [SSL Certificates](/en/docs/modules/ssl) module |
| `idleTimeout` | 900 | Seconds of idle time before a connection is closed |
| `connectionTimeout` | 30 | Timeout in seconds for establishing a data connection |
| `disableActiveMode` | Off | Use passive mode only, which suits NAT environments better |
| `users` | None | Multiple users, with the same fields as WebDAV: `name`, `password`, `root`, `readOnly` or `mounts`; user names must be unique |

Service parameters and users can be imported from Lucky's FTP configuration (see [Migrating from Lucky](/en/docs/guide/migration)).

## Storage Mounts

Add mounts under "Storage Mounts" in the "File Services" group, and manage files online on the page.

| Field | Description |
| --- | --- |
| `name`, `remark`, `enable` | Name, remark and enable switch |
| `type` | `local` (local directory), `webdav` (remote WebDAV) or `smb` (SMB share) |
| `path` | Local directory, must be an existing absolute path (used by `local`) |
| `url`, `username`, `password` | Remote address and account; the password is masked in the UI (used by `webdav` and `smb`). A WebDAV address starts with `http://` or `https://`; an SMB address looks like `smb://host[:port]/share[/subdir]`, SMB2/3 is supported and an empty username connects as guest |
| `skipVerify` | Skip HTTPS certificate verification (used by `webdav`) |
| `domain` | Domain or workgroup for SMB authentication, usually empty (used by `smb`) |
| `readOnly` | Mount read-only; the file manager rejects every write operation |

Local directories and WebDAV storages can be imported from Lucky's storage management. Alibaba Cloud Drive and system mounts are not migrated.

The online file manager supports browsing directories, downloading files, streaming uploads (the whole file is not loaded into memory), creating directories, renaming and deleting. A mount can serve as the target location of a backup task: backup files are written to the specified directory in the mount, and old files are pruned according to the retention policy.

## Wake-on-LAN

Manage devices under "Wake-on-LAN", and send magic packets to them (6 bytes of `0xFF` followed by the MAC repeated 16 times).

| Field | Default | Description |
| --- | --- | --- |
| `name` | None | Device name, required |
| `macs` | None | MAC addresses, one per line, at least one; you can enter several for one device |
| `broadcastIps` | Empty | IPv4 broadcast addresses. When empty, packets are sent to `255.255.255.255` and the broadcast address of each enabled network interface |
| `port` | 9 | Target UDP port |
| `repeat` | 3 | Number of times each MAC is sent, up to 20 |
| `checkAddr` | Empty | Online check address in the form `host:port`, for example `192.168.1.10:3389`; leave empty to skip the check |
| `remark` | Empty | Remark |

Click Wake in the device list to send the magic packet, and the page shows whether the device is online according to `checkAddr`.

## Scheduled Tasks

Run commands or send HTTP requests on a cron schedule.

- `spec` supports standard 5-field expressions, an optional seconds field (6 fields, such as `*/30 * * * * *`), and descriptors such as `@every 10m` and `@daily`;
- When `type` is `command`, `command` runs through `sh -c` (a temporary `.bat` on Windows), and the environment gets `taskName` plus an uppercase variable of the same name with the `GATELY_` prefix;
- When `type` is `http`, a request is sent according to the `http` config, with the fields `url`, `method` (GET, POST, PUT, PATCH, DELETE), `headers` (each as `Key: Value`), `body`, `successContent` (the response body must contain any one of its entries to count as success; when empty, only a 2xx status is checked), `timeout` and `insecureSkipVerify`;
- Tasks can be enabled or disabled, the page shows the next run time, and a task can be run immediately by hand;
- If the previous run of the same task has not finished, the new trigger is skipped;
- Each task keeps its latest 20 run records, including start time, duration, success or failure, whether it was triggered manually, and the output (a summary of up to 4000 characters);
- The creation form validates and parses the expression.

## Overview and System Logs

- The "Overview" page shows host information: CPU model, usage and load, memory, disk, network interface addresses and live rates, public IPv4 and IPv6 (click to copy), the service status of each module, and for the Gately process its version, commit, build time, start time, PID, CPU, resident memory, Go heap and goroutine count;
- The "System Logs" page collects the logs of all modules, including the runtime records of modules such as Docker, FTP and Scheduled Tasks.

## Themes

The UI offers several themes: Emerald, Ocean, Fresh, Sakura, Sunset, Dark, Midnight, Punk, and follow system (Dark when the system is dark, otherwise Emerald). The theme choice is stored locally in the browser.
