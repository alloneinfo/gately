# 웹 터미널 API 규약

모든 엔드포인트는 `/api/webterminal/` 아래에 마운트되며 로그인이 필요합니다. 오류는 항상 `{"error": "..."}` 형식으로 반환됩니다.

## 1. 설정 구조(`conf`의 webterminal 파일)

```jsonc
{
  "settings": {
    "allowLocalShell": false,
    "maxSessions": 20,
    "idleTimeout": 0,          // 분 단위. 입력과 출력이 이 시간 동안 없으면 세션을 자동으로 종료하며, 0은 제한 없음
    "bufferSize": 256,         // KB 단위. 세션마다 보관하는 출력 재생 버퍼로, 다시 연결할 때 재생됨
    "heartbeatInterval": 30,   // 초 단위. WebSocket 하트비트(서버 ping / 프런트엔드 ping)
    "sessionKeepAlive": 0      // 분 단위. 브라우저 연결이 끊긴 뒤 세션을 유지하는 시간이며, 0이면 끊기는 즉시 종료
  },
  "connections": [Connection],
  "shortcuts": [{ "name": "디스크 사용량 보기", "command": "df -h", "sendEnter": true }],
  "splitLayout": null,         // 프런트엔드가 정의하는 분할 창 레이아웃으로, 백엔드는 그대로 저장하고 반환함(json.RawMessage)
  "security": {
    "agreementAccepted": false,
    "twoFAEnable": false,
    "twoFAKey": ""             // TOTP 시크릿, API 출력에서는 마스킹됨
  },
  "credentials": [Credential]  // SSH 인증 프리셋
}
```

Connection(`type`은 `ssh` / `telnet` / `local`):

```jsonc
{
  "key": "", "name": "", "type": "ssh", "remark": "", "enable": true,
  "quickAccessDirs": ["/etc", "/var/log"],      // SFTP 빠른 접근 디렉터리
  "ssh": { "host", "port", "username", "authType": "password|key|credential",
           "credentialKey": "",  // authType=credential일 때 참조하는 인증 프리셋 key. 이때 username은 비워 둘 수 있으며 프리셋의 기본 사용자 이름을 사용함
           "password", "privateKey",
           "keyFile": "",        // 서버 자체에 있는 개인 키 파일 경로. authType=key이고 privateKey가 비어 있을 때 사용
           "passphrase", "hostKey", "keepAlive", "workDir": "", "timeout": 15,
           "proxy": Proxy },
  "telnet": { "host", "port": 23, "username", "password", "timeout": 15, "proxy": Proxy },
  "local": { "shell", "workDir", "env": {"K":"V"} }
}
Proxy = { "enabled": false, "type": "socks5|http", "host", "port", "username", "password" }
```

Credential(인증 프리셋, 여러 SSH 연결에서 참조 가능):

```jsonc
{ "key": "", "name": "", "authType": "password|key", "username": "", // 선택 사항, 연결에 사용자 이름이 없을 때 사용
  "password", "privateKey", "keyFile", "passphrase", "remark": "", "enable": true }
```

비밀번호, 개인 키, 개인 키 암호, 프록시 비밀번호는 출력 시 `crud.Mask`로 마스킹되고, 수정 시 `crud.KeepSecret`으로 유지됩니다.

## 2. 연결 관리

| 메서드 | 경로 | 설명 |
|---|---|---|
| GET/POST | `/connections` | 목록 / 생성 |
| PUT/DELETE | `/connections/{key}` | 수정 / 삭제 |
| PUT | `/connections-order` | 순서 변경 `{"keys":[...]}` |
| POST | `/connections/{key}/test` | SSH / Telnet 연결 테스트 |
| POST | `/connections/test` | 저장하지 않은 연결 테스트(본문은 Connection이며, `key` 필드로 원래 연결을 가리키면 저장된 비밀 값을 합쳐 사용) |
| DELETE | `/connections/{key}/hostkey` | 호스트 지문 초기화 |
| PUT | `/connections/{key}/quickaccess` | 본문 `{"dirs":["/a","/b"]}` |
| GET/POST | `/credentials` | 인증 프리셋 목록 / 생성 |
| PUT/DELETE | `/credentials/{key}` | 수정 / 삭제(아직 연결에서 참조 중이면 연결 이름을 나열하며 409 반환) |
| PUT | `/credentials-order` | 프리셋 순서 변경 `{"keys":[...]}` |
| GET | `/shells` | 이 기기에서 사용할 수 있는 셸 목록 `["/bin/bash", ...]`(/etc/shells를 읽고 실제로 있는 것만 남김) |

추가 설명(구현이 기준입니다):
- `POST /connections/test`와 `POST /connections/{key}/test`는 200과 함께 `{ok, error?, serverVersion?, newHostKey?, latencyMs}`를 반환합니다. `local` 유형은 400을 반환합니다. Telnet은 TCP 도달 여부(프록시 경유 포함)만 확인합니다.
- `PUT /connections/{key}/quickaccess`는 `{ok, dirs}`를 반환합니다(공백과 중복 제거). 연결을 편집할 때 `quickAccessDirs`가 없으면 기존 값을 유지합니다.
- `ssh.timeout` / `telnet.timeout`은 1~120초를 받으며 기본값은 15입니다. `ssh.workDir`은 로그인 후 `cd`로 적용합니다.
- `authType=credential`인 연결은 저장할 때 연결 안의 비밀번호, 개인 키, 키 파일, 암호를 비우고, 연결 시점에 프리셋에서 자격 증명을 가져옵니다. 프리셋을 수정하면 이를 참조하는 풀링된 SSH 연결이 끊깁니다.
- Telnet 로그인 자동 입력은 `login:` / `username:` / `password:` 같은 프롬프트를 감지했을 때만 각각 한 번씩 시도합니다.

## 3. 터미널 세션(연결이 끊겨도 다시 연결할 수 있는 영구 세션)

- `GET /connections/{key}/ws?cols=&rows=`: 새 세션을 만들고 WebSocket으로 업그레이드합니다.
- `GET /sessions/{id}/ws?cols=&rows=`: 기존 세션에 연결하여 버퍼를 먼저 재생한 뒤 실시간으로 전달합니다. 여러 클라이언트가 같은 세션에 동시에 연결할 수 있습니다(입력은 공유, 출력은 브로드캐스트).

WebSocket 프로토콜:
- 서버 → 브라우저: **바이너리 프레임**은 터미널 출력이고, **텍스트 프레임**은 JSON입니다.
  `{"t":"s","d":"<sessionId>"}` 세션 수립(생성과 연결 후 모두 한 번 전송);
  `{"t":"n","d":"알림"}` 알림; `{"t":"x","d":"사유"}` 세션 종료; `{"t":"p"}` 하트비트 응답.
- 브라우저 → 서버: 텍스트 JSON `{"t":"i","d":"입력"}`, `{"t":"r","c":cols,"r":rows}`, `{"t":"p"}` 하트비트.
- 브라우저 WebSocket 연결이 끊겼을 때: `sessionKeepAlive > 0`이면 세션을 유지하고(detached) 시간이 지나도록 아무도 연결하지 않으면 닫습니다. 0이면 즉시 닫습니다.
- 세션 프로세스가 스스로 종료되면 `x` 메시지를 보내고 WebSocket을 닫은 뒤 세션 테이블에서 제거합니다.

- `maxSessions`를 넘어 세션을 만들면 HTTP 429를 반환합니다. 연결에 실패하면 `n` 알림 프레임으로 알린 뒤 WebSocket을 닫습니다.
- `bytesIn`은 브라우저 입력 바이트 수, `bytesOut`은 터미널 출력 바이트 수, `attached`는 현재 연결된 클라이언트 수(number)입니다. 시간 필드는 RFC3339입니다.

REST:

| 메서드 | 경로 | 설명 |
|---|---|---|
| GET | `/sessions` | `[{id, connKey, connName, type, remark, createdAt, lastActive, attached, cols, rows, bytesIn, bytesOut}]` |
| GET | `/sessions/{id}` | 단일 세션 |
| PUT | `/sessions/{id}/remark` | `{"remark":""}` |
| GET | `/sessions/{id}/stats` | `{bytesIn, bytesOut, durationSec, idleSec, attached, cols, rows}` |
| DELETE | `/sessions/{id}` | 세션 종료 |

## 4. 설정, 바로 가기 명령, 분할 레이아웃

| 메서드 | 경로 | 설명 |
|---|---|---|
| GET/PUT | `/settings` | `{settings, activeSessions}` / 본문은 Settings |
| GET/PUT | `/globalshortcuts` | 바로 가기 명령 배열 |
| GET/PUT/DELETE | `/splitlayout` | 임의의 JSON 레이아웃 |

## 5. 보안(사용자 약관 + 2단계 인증)

| 메서드 | 경로 | 설명 |
|---|---|---|
| GET | `/security` | `{agreementAccepted, twoFAEnable, verified}`. `verified`는 현재 로그인 토큰이 2단계 인증을 통과했는지를 나타냅니다 |
| POST | `/security/agreement` | 약관 동의 |
| POST | `/security/2fa/setup` | 활성화 대기 중인 시크릿 `{secret, url}` 생성(저장하지 않음) |
| POST | `/security/2fa/enable` | `{secret, code}` 인증을 통과하면 2FA 활성화 |
| POST | `/security/2fa/disable` | `{code}` 인증을 통과하면 2FA 비활성화 |
| POST | `/security/verify2fa` | `{code}` 통과하면 현재 로그인 토큰을 30분 동안 인증된 것으로 간주 |
| GET | `/security/check2fa` | `{need: bool}`: 아직 인증이 필요한지 여부 |

2FA 코드가 5번 연속 틀리면 해당 로그인 토큰이 1분 동안 잠깁니다(HTTP 429). 2FA가 이미 활성화된 상태에서 `2fa/setup`은 400을 반환합니다(먼저 비활성화해야 함). `2fa/enable`에 성공하면 현재 토큰은 인증된 것으로 간주합니다.
접근 관문: 로컬 셸 활성화, 세션 생성, 세션 연결, 모든 SFTP 엔드포인트는 약관에 동의하지 않았으면 403 `{"error":"...","needAgreement":true}`를 반환하고,
2FA가 활성화되어 있는데 현재 토큰이 인증되지 않았으면 403 `{"error":"...","need2fa":true}`를 반환합니다. 여기서 "로컬 셸 활성화"는 `PUT /settings`로 `allowLocalShell`을 false에서 true로 바꾸는 것을 말합니다. Go 쪽에서는 모두 `m.requireAccess(w, r) bool`을 호출합니다.

## 6. SFTP 파일 관리

경로 매개변수 `{key}`는 **연결 key**입니다. SSH 연결은 SFTP를 사용하고(요청마다 재사용하는 SSH 연결 위에 SFTP 서브시스템 채널을 엶), `local` 연결은 서버 자체의 파일 시스템을 직접 다룹니다(같은 엔드포인트 사용).
Telnet 등 다른 유형은 400을, 없는 연결은 404를 반환합니다. 백엔드는 연결마다 SSH 연결 하나를 재사용하며 5분 동안 유휴 상태이면 닫습니다(`m.acquireSSH`).
`local` 연결은 설정에서 「로컬 셸 허용」이 켜져 있어야 하며, 그렇지 않으면 403을 반환합니다(로컬 파일 권한은 로컬 셸과 같음). SSH 연결에 실패하면 502를 반환합니다.

공통 규약:
- 모든 `path` / `from` / `to` / `dest`는 `/`로 시작하는 절대 경로여야 합니다(`list`의 `path`가 비어 있으면 기본 디렉터리를 사용합니다. SSH는 로그인 디렉터리, local은 연결의 workDir 또는 사용자 홈 디렉터리). 상대 경로는 400을 반환합니다.
- 오류는 항상 `{"error":"메시지"}`입니다. 없음 404, 권한 거부 403, 너무 큼 413, 지원하지 않는 유형 415, 대상이 이미 있음 409(이 절에서 확장), 잘못된 매개변수 400, 그 외 500.
- 성공한 쓰기 엔드포인트는 `{"ok":true, ...}`를 반환합니다.

| 메서드 | 경로 | 설명 |
|---|---|---|
| GET | `/sftp/{key}/list?path=` | `{path, parent, entries:[{name, path, isDir, isLink, size, mode, modeStr, modTime, owner?}]}`. 디렉터리가 먼저 오고 대소문자를 구분하지 않고 이름순으로 정렬합니다. `mode`는 `"755"` 같은 8진수 권한 문자열, `modeStr`은 `drwxr-xr-x` 형식, `modTime`은 RFC3339이며, `owner`는 local에서는 사용자 이름, SSH에서는 숫자 uid입니다. 심볼릭 링크는 `isLink=true`이고 디렉터리를 가리키면 `isDir=true`입니다 |
| GET | `/sftp/{key}/read?path=&download=1` | download가 없으면 텍스트 미리보기로 `{path, name, size, modTime, content}`를 반환합니다. 1MB를 넘으면 413, 바이너리 내용(NUL 포함 또는 UTF-8이 아님)이면 415를 반환합니다. download=1이면 스트리밍 다운로드(`application/octet-stream` + `Content-Disposition: attachment`)이며 `Range`와 `If-Modified-Since`를 지원합니다 |
| POST | `/sftp/{key}/write` | `{path, content}`로 텍스트를 저장합니다(덮어쓰기, 대상이 디렉터리면 400) → `{ok, path, size}` |
| POST | `/sftp/{key}/upload` | multipart 스트리밍 쓰기: `path`(대상 디렉터리, 이미 있어야 하며 **files보다 앞에 와야 함**), `files`(여러 개 가능), 선택 사항 `relativePaths`(파일마다 하나씩, 나타난 순서대로 뒤따르는 files와 짝지어지며 해당 files보다 앞에 와야 함. JSON 배열 문자열도 허용). `relativePaths`가 없으면 filename을 그대로 사용합니다(브라우저가 디렉터리를 업로드할 때 하위 경로가 포함될 수 있음). 같은 이름의 파일은 덮어쓰고 없는 중간 디렉터리는 자동으로 만듭니다. `..`가 포함된 경로는 400을 반환하므로 대상 디렉터리 밖에는 아무것도 쓰지 않습니다 → `{ok, count, files:[절대 경로]}` |
| POST | `/sftp/{key}/mkdir` | `{path}`로 디렉터리를 만듭니다(없는 상위 디렉터리 포함). 이미 있으면 409 |
| POST | `/sftp/{key}/touch` | `{path}`로 빈 파일을 만듭니다(없는 상위 디렉터리 포함). 이미 있으면 409 |
| POST | `/sftp/{key}/rename` | `{from, to}`. `to`가 이미 있으면 409, 디렉터리를 자기 내부로 옮기면 400, 루트 디렉터리 이름을 바꾸면 403입니다. local에서 장치를 넘나들면 자동으로 복사 후 삭제로 대체합니다 |
| POST | `/sftp/{key}/copy` | `{from:[...], to}`. `to`는 대상 **디렉터리**이며 각 항목은 `to/원래 이름`으로 복사됩니다(재귀, 심볼릭 링크는 링크로 복사, 장치·파이프 같은 특수 파일은 무시). 이름이 겹치면 오류 없이 `name copy.ext`, `name copy 2.ext` 등으로 **자동으로 이름을 바꿉니다**. 디렉터리를 자기 내부로 복사하면 400 → `{ok, paths:[새 경로]}` |
| POST | `/sftp/{key}/chmod` | `{path, mode:"755", recursive}`. `mode`는 0~7777의 8진수 문자열입니다. 재귀 시 심볼릭 링크를 따라가지 않으며 파일과 디렉터리 모두 같은 권한으로 설정합니다 |
| DELETE | `/sftp/{key}/remove` | `{paths:[...]}`(JSON 본문, 재귀 삭제, 심볼릭 링크는 링크만 삭제). 경로 중 하나라도 `/`이면 403을 반환하고 아무것도 삭제하지 않습니다 → `{ok, count}` |
| POST | `/sftp/{key}/compress` | `{paths:[...], dest, format:"zip|tar.gz"}`(format 기본값은 zip, `tgz`는 tar.gz로 처리). `dest`는 출력 압축 파일의 전체 경로입니다. `dest`가 이미 있는 디렉터리이거나 `/`로 끝나면 그 안에서 자동으로 이름을 정합니다(항목이 하나면 그 이름, 여러 개면 상위 디렉터리 이름에 `.zip` / `.tar.gz`를 붙임). 각 항목은 자기 이름으로 압축 파일의 최상위 항목이 됩니다. 이름이 겹치는 항목이 있으면 400, 출력 파일이 이미 있으면 409이며, 출력 파일 자체는 압축에 포함되지 않습니다 → `{ok, path}` |
| POST | `/sftp/{key}/decompress` | `{path, dest}`. zip / tar / tar.gz / tgz를 지원합니다(확장자로 판별, 그 외는 415). `dest`가 비어 있으면 압축 파일이 있는 디렉터리에 풀고, 디렉터리가 없으면 만듭니다. 같은 이름의 파일은 덮어씁니다. `..` 항목이 하나라도 있으면 압축 파일 전체를 거부합니다(400, zip-slip 방지). 심볼릭 링크, 하드 링크 등 일반 파일이 아닌 항목은 건너뛰며 파일 권한은 rwx 비트만 유지합니다 → `{ok, dest, count}` |
| GET | `/sftp/{key}/preview-archive?path=` | 압축 파일 항목 목록 `[{name,size,isDir}]`(최대 10000개, `name`에 끝의 `/` 없음) |

압축과 압축 해제는 모두 Go에서 파일 시스템 인터페이스(local은 os, SSH는 SFTP 읽기·쓰기)로 처리하며 원격 명령에 의존하지 않습니다.

## 7. 기타

- 로그: 모듈 로그는 전역 로그 페이지에 이미 표시되므로 별도 엔드포인트가 필요 없습니다.
- Lucky 가져오기: telnet, 프록시, keyFile, workDir, timeout, env, quickAccessDirs, sessionConfig, globalShortcuts,
  securityConfig(사용자 약관과 2FA 시크릿도 함께 마이그레이션)를 채웁니다.
