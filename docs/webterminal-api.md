# Web 终端接口约定（对标 Lucky 2.27.2）

所有接口挂载在 `/api/webterminal/` 下，需登录。错误统一为 `{"error": "..."}`。

## 1. 配置结构（`conf` 中 webterminal 文件）

```jsonc
{
  "settings": {
    "allowLocalShell": false,
    "maxSessions": 20,
    "idleTimeout": 0,          // 分钟，会话无输入输出超过该时间自动结束，0 不限
    "bufferSize": 256,         // KB，每个会话保留的输出回放缓冲，重连时回放
    "heartbeatInterval": 30,   // 秒，WebSocket 心跳（服务端 ping / 前端 ping）
    "sessionKeepAlive": 0      // 分钟，浏览器断开后会话继续保留的时间；0 表示断开即结束
  },
  "connections": [Connection],
  "shortcuts": [{ "name": "查看磁盘", "command": "df -h", "sendEnter": true }],
  "splitLayout": null,         // 前端自定义的分屏布局，后端原样存取（json.RawMessage）
  "security": {
    "agreementAccepted": false,
    "twoFAEnable": false,
    "twoFAKey": ""             // TOTP 密钥，接口输出时脱敏
  }
}
```

Connection（`type` 为 `ssh` / `telnet` / `local`）：

```jsonc
{
  "key": "", "name": "", "type": "ssh", "remark": "", "enable": true,
  "quickAccessDirs": ["/etc", "/var/log"],      // SFTP 快速访问目录
  "ssh": { "host", "port", "username", "authType": "password|key", "password", "privateKey",
           "keyFile": "",        // 服务器本机上的私钥文件路径，authType=key 且 privateKey 为空时使用
           "passphrase", "hostKey", "keepAlive", "workDir": "", "timeout": 15,
           "proxy": Proxy },
  "telnet": { "host", "port": 23, "username", "password", "timeout": 15, "proxy": Proxy },
  "local": { "shell", "workDir", "env": {"K":"V"} }
}
Proxy = { "enabled": false, "type": "socks5|http", "host", "port", "username", "password" }
```

密码、私钥、私钥口令、代理密码输出时用 `crud.Mask` 脱敏，修改时用 `crud.KeepSecret` 保留。

## 2. 连接管理

| 方法 | 路径 | 说明 |
|---|---|---|
| GET/POST | `/connections` | 列表 / 新增 |
| PUT/DELETE | `/connections/{key}` | 修改 / 删除 |
| PUT | `/connections-order` | 排序 `{"keys":[...]}` |
| POST | `/connections/{key}/test` | 测试 SSH / Telnet 连通性 |
| POST | `/connections/test` | 测试未保存的连接（body 为 Connection；已保存的密文用 `key` 字段指向原连接合并） |
| DELETE | `/connections/{key}/hostkey` | 重置主机指纹 |
| PUT | `/connections/{key}/quickaccess` | body `{"dirs":["/a","/b"]}` |
| GET | `/shells` | 本机可用 Shell 列表 `["/bin/bash", ...]`（读取 /etc/shells 并过滤存在的） |

补充说明（以实现为准）：
- `POST /connections/test` 与 `POST /connections/{key}/test` 返回 200，`{ok, error?, serverVersion?, newHostKey?, latencyMs}`；`local` 类型返回 400。Telnet 仅验证 TCP（含代理）可达。
- `PUT /connections/{key}/quickaccess` 返回 `{ok, dirs}`（去空白与重复）；编辑连接时若不带 `quickAccessDirs` 则沿用旧值。
- 连接的 `ssh.timeout` / `telnet.timeout` 取值 1~120 秒，缺省 15。`ssh.workDir` 在登录后通过 `cd` 切换。
- Telnet 登录自动填充仅在检测到 `login:` / `username:` / `password:` 等提示时各尝试一次。

## 3. 终端会话（持久会话，可断线重连）

- `GET /connections/{key}/ws?cols=&rows=`：新建会话并升级为 WebSocket。
- `GET /sessions/{id}/ws?cols=&rows=`：附着到已有会话，先回放缓冲再实时转发；同一会话允许多个客户端同时附着（输入共享、输出广播）。

WebSocket 协议：
- 服务端 → 浏览器：**二进制帧**为终端输出；**文本帧**为 JSON：
  `{"t":"s","d":"<sessionId>"}` 会话建立（新建与附着后都会先发一次）；
  `{"t":"n","d":"提示"}` 提示；`{"t":"x","d":"原因"}` 会话已结束；`{"t":"p"}` 心跳响应。
- 浏览器 → 服务端：文本 JSON：`{"t":"i","d":"输入"}`、`{"t":"r","c":列,"r":行}`、`{"t":"p"}` 心跳。
- 浏览器 WebSocket 断开：`sessionKeepAlive > 0` 时会话保留（detached），超时无人附着后关闭；为 0 时立即关闭。
- 会话进程自行退出时发送 `x` 并关闭 WebSocket，从会话表中移除。

- 超过 `maxSessions` 时新建会话返回 HTTP 429；连接失败通过 `n` 提示帧告知后关闭 WebSocket。
- `bytesIn` 为浏览器输入的字节数，`bytesOut` 为终端输出的字节数；`attached` 为当前附着的客户端数量（number）；时间字段为 RFC3339。

REST：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/sessions` | `[{id, connKey, connName, type, remark, createdAt, lastActive, attached, cols, rows, bytesIn, bytesOut}]` |
| GET | `/sessions/{id}` | 单个会话 |
| PUT | `/sessions/{id}/remark` | `{"remark":""}` |
| GET | `/sessions/{id}/stats` | `{bytesIn, bytesOut, durationSec, idleSec, attached, cols, rows}` |
| DELETE | `/sessions/{id}` | 结束会话 |

## 4. 设置、快捷命令、分屏布局

| 方法 | 路径 | 说明 |
|---|---|---|
| GET/PUT | `/settings` | `{settings, activeSessions}` / body 为 Settings |
| GET/PUT | `/globalshortcuts` | 快捷命令数组 |
| GET/PUT/DELETE | `/splitlayout` | 任意 JSON 布局 |

## 5. 安全（用户协议 + 二次验证）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/security` | `{agreementAccepted, twoFAEnable, verified}`，`verified` 表示当前登录令牌已通过二次验证 |
| POST | `/security/agreement` | 接受协议 |
| POST | `/security/2fa/setup` | 生成待启用密钥 `{secret, url}`（不落盘） |
| POST | `/security/2fa/enable` | `{secret, code}` 校验通过后启用 |
| POST | `/security/2fa/disable` | `{code}` 校验通过后关闭 |
| POST | `/security/verify2fa` | `{code}` 通过后当前登录令牌 30 分钟内视为已验证 |
| GET | `/security/check2fa` | `{need: bool}`，是否还需要验证 |

2FA 验证码连续失败 5 次后该登录令牌锁定 1 分钟（HTTP 429）；`2fa/setup` 在已启用时返回 400（需先关闭）；`2fa/enable` 成功后当前令牌视为已验证。
门禁：开启本机 Shell / 新建会话 / 附着会话 / SFTP 所有接口，在协议未接受时返回 403 `{"error":"...","needAgreement":true}`，
启用 2FA 且当前令牌未验证时返回 403 `{"error":"...","need2fa":true}`。其中“开启本机 Shell”指 `PUT /settings` 把 `allowLocalShell` 由 false 改为 true。Go 侧统一调用 `m.requireAccess(w, r) bool`。

## 6. SFTP 文件管理

路径参数 `{key}` 为**连接 key**。SSH 连接走 SFTP（每次请求在复用的 SSH 连接上开一个 SFTP 子系统通道）；`local` 连接直接操作服务器本机文件系统（同一套接口）。
Telnet 等其他类型返回 400；连接不存在返回 404。后端为每个连接复用一条 SSH 连接，空闲 5 分钟关闭（`m.acquireSSH`）。
`local` 连接要求设置里已开启「允许本机 Shell」，否则返回 403（本机文件权限等同本机 Shell）。SSH 连接失败返回 502。

通用约定：
- 所有 `path` / `from` / `to` / `dest` 必须是以 `/` 开头的绝对路径（`list` 的 `path` 为空时取默认目录：SSH 为登录目录，local 为连接的 workDir 或用户主目录）；相对路径返回 400。
- 错误统一 `{"error":"中文文案"}`：不存在 404、权限 403、过大 413、类型不支持 415、目标已存在 409（本节扩展）、参数错误 400、其余 500。
- 成功的写类接口返回 `{"ok":true, ...}`。

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/sftp/{key}/list?path=` | `{path, parent, entries:[{name, path, isDir, isLink, size, mode, modeStr, modTime, owner?}]}`，目录在前、按名称忽略大小写排序。`mode` 为八进制权限串如 `"755"`，`modeStr` 为 `drwxr-xr-x` 形式，`modTime` 为 RFC3339，`owner` 在 local 为用户名、SSH 为数字 uid。符号链接 `isLink=true`，指向目录时 `isDir=true` |
| GET | `/sftp/{key}/read?path=&download=1` | 无 download：文本预览，返回 `{path, name, size, modTime, content}`；超过 1MB 返回 413，含 NUL 或非 UTF-8 的二进制返回 415。download=1：流式下载（`application/octet-stream` + `Content-Disposition: attachment`），支持 `Range` 与 `If-Modified-Since` |
| POST | `/sftp/{key}/write` | `{path, content}` 保存文本（覆盖写，目标为目录返回 400）→ `{ok, path, size}` |
| POST | `/sftp/{key}/upload` | multipart 流式写入：`path`（目标目录，必须已存在，且**必须位于 files 之前**）、`files`（可多个）、可选 `relativePaths`（每个文件一个，按出现顺序与其后的 files 一一对应，必须位于对应 files 之前；也接受 JSON 数组字符串）。未提供 `relativePaths` 时使用 filename 原文（浏览器上传目录时可带子路径）。同名文件覆盖，缺失的中间目录自动创建；包含 `..` 的路径返回 400，保证不会写出目标目录 → `{ok, count, files:[绝对路径]}` |
| POST | `/sftp/{key}/mkdir` | `{path}` 创建目录（含缺失的父目录），已存在 409 |
| POST | `/sftp/{key}/touch` | `{path}` 创建空文件（含缺失的父目录），已存在 409 |
| POST | `/sftp/{key}/rename` | `{from, to}`，`to` 已存在 409，把目录移到自身内部 400，禁止重命名根目录 403；local 跨设备时自动退化为复制后删除 |
| POST | `/sftp/{key}/copy` | `{from:[...], to}`，`to` 是目标**目录**，每项复制到 `to/原名`（递归，符号链接按链接复制，忽略设备/管道等特殊文件）。目标重名时**自动改名**为 `name 副本.ext`、`name 副本 2.ext`……，不报错；复制目录到自身内部 400 → `{ok, paths:[新路径]}` |
| POST | `/sftp/{key}/chmod` | `{path, mode:"755", recursive}`，`mode` 为 0–7777 的八进制串；递归时不跟随符号链接，文件与目录统一设为同一权限 |
| DELETE | `/sftp/{key}/remove` | `{paths:[...]}`（body JSON，递归删除，符号链接只删链接）；任一路径为 `/` 返回 403 且不删除任何项 → `{ok, count}` |
| POST | `/sftp/{key}/compress` | `{paths:[...], dest, format:"zip|tar.gz"}`（format 缺省 zip，`tgz` 视为 tar.gz）。`dest` 为输出压缩包的完整路径；若 `dest` 是已存在的目录或以 `/` 结尾，则在其中自动命名（单个项目用其名称，多个项目用其父目录名，加 `.zip` / `.tar.gz`）。每个项目以自身名称为归档顶层，同名项目 400，输出文件已存在 409，输出文件自身不会被打入 → `{ok, path}` |
| POST | `/sftp/{key}/decompress` | `{path, dest}` 支持 zip / tar / tar.gz / tgz（按后缀识别，其他 415）。`dest` 为空时解压到压缩包所在目录，不存在则创建；同名文件覆盖。含 `..` 的条目整体拒绝（400，防 zip-slip）；符号链接、硬链接等非普通文件条目被跳过；文件权限只保留 rwx 位 → `{ok, dest, count}` |
| GET | `/sftp/{key}/preview-archive?path=` | 压缩包条目列表 `[{name,size,isDir}]`（最多 10000 条，`name` 不含结尾 `/`） |

压缩/解压全部由 Go 通过文件系统接口完成（local 用 os，SSH 用 SFTP 读写），不依赖远端命令。

## 7. 其他

- 日志：模块日志已由全局日志页统一展示，无需单独接口。
- Lucky 导入：补全 telnet、代理、keyFile、workDir、timeout、env、quickAccessDirs、sessionConfig、globalShortcuts、
  securityConfig（用户协议与 2FA 密钥一并迁移）。
