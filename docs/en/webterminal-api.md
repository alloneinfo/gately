# Web Terminal API Conventions (targeting Lucky 2.27.2)

All endpoints are mounted under `/api/webterminal/` and require login. Errors are always returned as `{"error": "..."}`.

## 1. Configuration structure (the webterminal file in `conf`)

```jsonc
{
  "settings": {
    "allowLocalShell": false,
    "maxSessions": 20,
    "idleTimeout": 0,          // minutes; a session with no input or output for this long is ended automatically; 0 means unlimited
    "bufferSize": 256,         // KB; output replay buffer kept per session and replayed on reconnect
    "heartbeatInterval": 30,   // seconds; WebSocket heartbeat (server ping / frontend ping)
    "sessionKeepAlive": 0      // minutes; how long a session is kept after the browser disconnects; 0 ends it immediately on disconnect
  },
  "connections": [Connection],
  "shortcuts": [{ "name": "Show disk usage", "command": "df -h", "sendEnter": true }],
  "splitLayout": null,         // split-pane layout customized by the frontend; the backend stores and returns it as-is (json.RawMessage)
  "security": {
    "agreementAccepted": false,
    "twoFAEnable": false,
    "twoFAKey": ""             // TOTP secret; masked in API output
  },
  "credentials": [Credential]  // SSH credential presets
}
```

Connection (`type` is `ssh` / `telnet` / `local`):

```jsonc
{
  "key": "", "name": "", "type": "ssh", "remark": "", "enable": true,
  "quickAccessDirs": ["/etc", "/var/log"],      // SFTP quick-access directories
  "ssh": { "host", "port", "username", "authType": "password|key|credential",
           "credentialKey": "",  // credential preset key when authType=credential; username may then be empty and falls back to the preset's default
           "password", "privateKey",
           "keyFile": "",        // path to a private key file on the server itself; used when authType=key and privateKey is empty
           "passphrase", "hostKey", "keepAlive", "workDir": "", "timeout": 15,
           "proxy": Proxy },
  "telnet": { "host", "port": 23, "username", "password", "timeout": 15, "proxy": Proxy },
  "local": { "shell", "workDir", "env": {"K":"V"} }
}
Proxy = { "enabled": false, "type": "socks5|http", "host", "port", "username", "password" }
```

Credential (credential preset, can be referenced by many SSH connections):

```jsonc
{ "key": "", "name": "", "authType": "password|key", "username": "", // optional, used when the connection has no username
  "password", "privateKey", "keyFile", "passphrase", "remark": "", "enable": true }
```

Passwords, private keys, private key passphrases, and proxy passwords are masked with `crud.Mask` in output and preserved with `crud.KeepSecret` on update.

## 2. Connection management

| Method | Path | Description |
|---|---|---|
| GET/POST | `/connections` | List / create |
| PUT/DELETE | `/connections/{key}` | Update / delete |
| PUT | `/connections-order` | Reorder `{"keys":[...]}` |
| POST | `/connections/{key}/test` | Test SSH / Telnet connectivity |
| POST | `/connections/test` | Test an unsaved connection (the body is a Connection; saved secrets are merged in by pointing the `key` field at the original connection) |
| DELETE | `/connections/{key}/hostkey` | Reset the host fingerprint |
| PUT | `/connections/{key}/quickaccess` | Body `{"dirs":["/a","/b"]}` |
| GET/POST | `/credentials` | List / create credential presets |
| PUT/DELETE | `/credentials/{key}` | Update / delete (returns 409 listing connection names while still referenced) |
| PUT | `/credentials-order` | Reorder presets `{"keys":[...]}` |
| GET | `/shells` | List of shells available on this machine `["/bin/bash", ...]` (reads /etc/shells and filters to those that exist) |

Additional notes (the implementation is authoritative):
- `POST /connections/test` and `POST /connections/{key}/test` return 200 with `{ok, error?, serverVersion?, newHostKey?, latencyMs}`; the `local` type returns 400. For Telnet, only TCP reachability (including through the proxy) is verified.
- `PUT /connections/{key}/quickaccess` returns `{ok, dirs}` (whitespace and duplicates removed). When editing a connection without `quickAccessDirs`, the old value is kept.
- `ssh.timeout` / `telnet.timeout` accept 1 to 120 seconds, defaulting to 15. `ssh.workDir` is applied with `cd` after login.
- Connections with `authType=credential` have their inline password, private key, key file and passphrase cleared on save; credentials are resolved from the preset at connect time. Updating a preset drops pooled SSH connections that reference it.
- Telnet login auto-fill tries once each, and only when prompts such as `login:` / `username:` / `password:` are detected.

## 3. Terminal sessions (persistent sessions that can reconnect after a disconnect)

- `GET /connections/{key}/ws?cols=&rows=`: creates a new session and upgrades to WebSocket.
- `GET /sessions/{id}/ws?cols=&rows=`: attaches to an existing session, replays the buffer first, then forwards in real time. Multiple clients may attach to the same session at once (input is shared, output is broadcast).

WebSocket protocol:
- Server → browser: **binary frames** carry terminal output; **text frames** carry JSON:
  `{"t":"s","d":"<sessionId>"}` session established (sent once after both creating and attaching);
  `{"t":"n","d":"notice"}` notice; `{"t":"x","d":"reason"}` session ended; `{"t":"p"}` heartbeat reply.
- Browser → server: text JSON: `{"t":"i","d":"input"}`, `{"t":"r","c":cols,"r":rows}`, `{"t":"p"}` heartbeat.
- When the browser WebSocket disconnects: if `sessionKeepAlive > 0`, the session is kept (detached) and closed after the timeout if nobody attaches; if 0, it is closed immediately.
- When the session process exits on its own, an `x` message is sent, the WebSocket is closed, and the session is removed from the session table.

- Creating a session beyond `maxSessions` returns HTTP 429; a connection failure is reported through an `n` notice frame, after which the WebSocket is closed.
- `bytesIn` is the number of bytes of browser input, `bytesOut` is the number of bytes of terminal output, and `attached` is the number of currently attached clients (a number); time fields are RFC3339.

REST:

| Method | Path | Description |
|---|---|---|
| GET | `/sessions` | `[{id, connKey, connName, type, remark, createdAt, lastActive, attached, cols, rows, bytesIn, bytesOut}]` |
| GET | `/sessions/{id}` | A single session |
| PUT | `/sessions/{id}/remark` | `{"remark":""}` |
| GET | `/sessions/{id}/stats` | `{bytesIn, bytesOut, durationSec, idleSec, attached, cols, rows}` |
| DELETE | `/sessions/{id}` | End the session |

## 4. Settings, shortcuts, and split layout

| Method | Path | Description |
|---|---|---|
| GET/PUT | `/settings` | `{settings, activeSessions}` / the body is Settings |
| GET/PUT | `/globalshortcuts` | Array of shortcut commands |
| GET/PUT/DELETE | `/splitlayout` | Arbitrary JSON layout |

## 5. Security (user agreement + two-factor verification)

| Method | Path | Description |
|---|---|---|
| GET | `/security` | `{agreementAccepted, twoFAEnable, verified}`; `verified` indicates that the current login token has passed two-factor verification |
| POST | `/security/agreement` | Accept the agreement |
| POST | `/security/2fa/setup` | Generate a pending secret `{secret, url}` (not persisted) |
| POST | `/security/2fa/enable` | `{secret, code}` enables 2FA once verification passes |
| POST | `/security/2fa/disable` | `{code}` disables 2FA once verification passes |
| POST | `/security/verify2fa` | `{code}`; once it passes, the current login token counts as verified for 30 minutes |
| GET | `/security/check2fa` | `{need: bool}`: whether verification is still required |

After 5 consecutive failed 2FA codes, the login token is locked for 1 minute (HTTP 429). `2fa/setup` returns 400 when 2FA is already enabled (disable it first). After a successful `2fa/enable`, the current token counts as verified.
Gate: enabling the local shell, creating a session, attaching to a session, and every SFTP endpoint return 403 `{"error":"...","needAgreement":true}` when the agreement has not been accepted,
and return 403 `{"error":"...","need2fa":true}` when 2FA is enabled and the current token has not been verified. "Enabling the local shell" here means `PUT /settings` changing `allowLocalShell` from false to true. On the Go side, all of these call `m.requireAccess(w, r) bool`.

## 6. SFTP file management

The path parameter `{key}` is the **connection key**. SSH connections use SFTP (each request opens an SFTP subsystem channel on a reused SSH connection); `local` connections operate directly on the server's own filesystem (through the same endpoints).
Telnet and other types return 400; a missing connection returns 404. The backend reuses one SSH connection per connection and closes it after 5 minutes idle (`m.acquireSSH`).
A `local` connection requires "Allow local shell" to be enabled in settings, otherwise it returns 403 (local file permissions are equivalent to a local shell). An SSH connection failure returns 502.

General conventions:
- Every `path` / `from` / `to` / `dest` must be an absolute path starting with `/` (when the `path` of `list` is empty, the default directory is used: the login directory for SSH, and the connection's workDir or the user's home directory for local). Relative paths return 400.
- Errors are always `{"error":"message"}`: not found 404, permission denied 403, too large 413, unsupported type 415, target already exists 409 (extended in this section), bad parameters 400, everything else 500.
- Successful write endpoints return `{"ok":true, ...}`.

| Method | Path | Description |
|---|---|---|
| GET | `/sftp/{key}/list?path=` | `{path, parent, entries:[{name, path, isDir, isLink, size, mode, modeStr, modTime, owner?}]}`; directories first, then sorted by name case-insensitively. `mode` is an octal permission string such as `"755"`, `modeStr` is in the `drwxr-xr-x` form, `modTime` is RFC3339, and `owner` is the user name for local and the numeric uid for SSH. Symbolic links have `isLink=true`, and `isDir=true` when they point to a directory |
| GET | `/sftp/{key}/read?path=&download=1` | Without download: text preview, returns `{path, name, size, modTime, content}`; over 1MB returns 413, and binary content (containing NUL or non-UTF-8) returns 415. With download=1: streaming download (`application/octet-stream` + `Content-Disposition: attachment`), supporting `Range` and `If-Modified-Since` |
| POST | `/sftp/{key}/write` | `{path, content}` saves text (overwrite; a directory target returns 400) → `{ok, path, size}` |
| POST | `/sftp/{key}/upload` | multipart streaming write: `path` (the target directory, which must already exist and **must come before files**), `files` (may repeat), and optional `relativePaths` (one per file; matched in order of appearance with the files that follow, and must come before the corresponding files; a JSON array string is also accepted). Without `relativePaths`, the filename is used verbatim (it may carry a subpath when a browser uploads a directory). Files with the same name are overwritten and missing intermediate directories are created automatically; paths containing `..` return 400, guaranteeing nothing is written outside the target directory → `{ok, count, files:[absolute paths]}` |
| POST | `/sftp/{key}/mkdir` | `{path}` creates a directory (including missing parents); 409 if it already exists |
| POST | `/sftp/{key}/touch` | `{path}` creates an empty file (including missing parents); 409 if it already exists |
| POST | `/sftp/{key}/rename` | `{from, to}`; 409 if `to` already exists, 400 when moving a directory into itself, 403 when renaming the root directory; for local across devices, it automatically falls back to copy-then-delete |
| POST | `/sftp/{key}/copy` | `{from:[...], to}`; `to` is the target **directory**, and each item is copied to `to/original-name` (recursive; symbolic links are copied as links; special files such as devices and pipes are ignored). On a name collision the copy is **renamed automatically** to `name copy.ext`, `name copy 2.ext`, and so on, without an error; copying a directory into itself returns 400 → `{ok, paths:[new paths]}` |
| POST | `/sftp/{key}/chmod` | `{path, mode:"755", recursive}`; `mode` is an octal string from 0 to 7777; recursion does not follow symbolic links, and files and directories all get the same permission |
| DELETE | `/sftp/{key}/remove` | `{paths:[...]}` (JSON body; recursive delete; for symbolic links only the link is removed); if any path is `/`, returns 403 and nothing is deleted → `{ok, count}` |
| POST | `/sftp/{key}/compress` | `{paths:[...], dest, format:"zip|tar.gz"}` (format defaults to zip; `tgz` is treated as tar.gz). `dest` is the full path of the output archive; if `dest` is an existing directory or ends with `/`, the file is named automatically inside it (the item's own name for a single item, or the parent directory's name for multiple items, plus `.zip` / `.tar.gz`). Each item becomes a top-level entry of the archive under its own name; items with duplicate names return 400, an existing output file returns 409, and the output file is never packed into itself → `{ok, path}` |
| POST | `/sftp/{key}/decompress` | `{path, dest}` supports zip / tar / tar.gz / tgz (detected by suffix; anything else returns 415). When `dest` is empty, it extracts into the directory containing the archive, creating it if needed; files with the same name are overwritten. An archive containing any `..` entry is rejected entirely (400, zip-slip protection); entries that are not regular files, such as symbolic links and hard links, are skipped; file permissions keep only the rwx bits → `{ok, dest, count}` |
| GET | `/sftp/{key}/preview-archive?path=` | Archive entry list `[{name,size,isDir}]` (at most 10000 entries; `name` has no trailing `/`) |

Compression and decompression are done entirely in Go through the filesystem interface (os for local, SFTP reads and writes for SSH), with no dependency on remote commands.

## 7. Other

- Logs: module logs are already shown on the global logs page, so no separate endpoint is needed.
- Lucky import: fills in telnet, proxy, keyFile, workDir, timeout, env, quickAccessDirs, sessionConfig, globalShortcuts,
  and securityConfig (the user agreement and the 2FA secret are migrated together).
