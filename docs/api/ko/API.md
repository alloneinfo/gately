# Gately 관리 API

Gately 관리 API는 스크립트와 자동화를 통한 일괄 관리에 사용합니다.

인증: 관리 콘솔의 "API 토큰" 페이지에서 토큰을 생성하고 요청 헤더에 `Authorization: Bearer gtl_xxxx`를 포함하세요.

권한: 각 토큰은 생성 시 선택한 권한만 가지며 형식은 `<모듈>:read` 또는 `<모듈>:write`입니다. 쓰기 권한은 읽기 권한을 포함합니다. 계정, 2단계 인증, 토큰 관리, 백업, 복원, 재시작, 시스템 설정 인터페이스는 로그인 세션으로만 사용할 수 있습니다.

요청 본문은 `Content-Type: application/json`을 사용하며, 오류는 `{"error": "설명"}` 형식으로 반환됩니다.

공식 사이트: https://gately.cn

기계가 읽을 수 있는 설명은 같은 디렉터리의 `openapi.json`(OpenAPI 3.0)을 참고하세요.

## 빠른 시작

```bash
# 시스템 상태 조회 (status:read 필요)
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/status

# 모든 DDNS 작업 나열 (ddns:read 필요)
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks

# DDNS 작업 즉시 동기화 (ddns:write 필요, key는 이전 단계에서 반환된 목록에서 가져옴)
curl -X POST -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks/<key>/sync
```

## 권한

| 권한 영역 | 이름 | 사용 가능 권한 |
| --- | --- | --- |
| `portforward` | 포트 포워딩 | `portforward:read`、`portforward:write` |
| `ddns` | 동적 도메인 | `ddns:read`、`ddns:write` |
| `stun` | STUN 터널링 | `stun:read`、`stun:write` |
| `tailscale` | Tailscale 네트워킹 | `tailscale:read`、`tailscale:write` |
| `frp` | frp 내부망 터널링 | `frp:read`、`frp:write` |
| `cloudflared` | Cloudflare 터널 | `cloudflared:read`、`cloudflared:write` |
| `ssl` | SSL 인증서 | `ssl:read`、`ssl:write` |
| `waf` | WAF 방화벽 | `waf:read`、`waf:write` |
| `webservice` | Web 서비스 | `webservice:read`、`webservice:write` |
| `wol` | Wake-on-LAN | `wol:read`、`wol:write` |
| `webterminal` | Web 터미널 | `webterminal:read`、`webterminal:write` |
| `cron` | 예약 작업 | `cron:read`、`cron:write` |
| `docker` | Docker 관리 | `docker:read`、`docker:write` |
| `webdav` | WebDAV | `webdav:read`、`webdav:write` |
| `ftp` | FTP 서비스 | `ftp:read`、`ftp:write` |
| `storage` | 스토리지 관리 | `storage:read`、`storage:write` |
| `notify` | 알림 센터 | `notify:read`、`notify:write` |
| `nettools` | 네트워크 진단 | `nettools:read`、`nettools:write` |
| `backup` | 설정 백업 | `backup:read`、`backup:write` |
| `status` | 시스템 상태 | `status:read` |
| `logs` | 시스템 로그 | `logs:read` |
| `import` | Lucky 설정 가져오기 | `import:write` |

와일드카드: `*:read`는 모든 모듈 읽기 전용, `*:write` 또는 `*`는 모든 모듈 읽기/쓰기를 의미합니다.

## 포트 포워딩

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/portforward/logs` | `portforward:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/portforward/logs` | `portforward:write` | 모듈 실행 로그 비우기 |
| GET | `/api/portforward/rules` | `portforward:read` | 포트 포워딩 규칙 목록 조회 |
| POST | `/api/portforward/rules` | `portforward:write` | 포트 포워딩 규칙 추가 |
| PUT | `/api/portforward/rules/{key}` | `portforward:write` | 포트 포워딩 규칙 수정 |
| DELETE | `/api/portforward/rules/{key}` | `portforward:write` | 포트 포워딩 규칙 삭제 |
| PUT | `/api/portforward/rules/{key}/enable` | `portforward:write` | 포트 포워딩 규칙 사용 설정/해제 |
| PUT | `/api/portforward/rules-order` | `portforward:write` | 포트 포워딩 규칙 순서 변경 |
| GET | `/api/portforward/settings` | `portforward:read` | 포트 포워딩 전역 설정 읽기 |
| PUT | `/api/portforward/settings` | `portforward:write` | 포트 포워딩 전역 설정 저장 |

## 동적 도메인

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/ddns/logs` | `ddns:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/ddns/logs` | `ddns:write` | 모듈 실행 로그 비우기 |
| GET | `/api/ddns/tasks` | `ddns:read` | DDNS 작업 목록 조회 |
| POST | `/api/ddns/tasks` | `ddns:write` | DDNS 작업 추가 |
| PUT | `/api/ddns/tasks/{key}` | `ddns:write` | DDNS 작업 수정 |
| DELETE | `/api/ddns/tasks/{key}` | `ddns:write` | DDNS 작업 삭제 |
| PUT | `/api/ddns/tasks/{key}/enable` | `ddns:write` | DDNS 작업 사용 설정/해제 |
| PUT | `/api/ddns/tasks-order` | `ddns:write` | DDNS 작업 순서 변경 |
| POST | `/api/ddns/tasks/{key}/sync` | `ddns:write` | DDNS 작업 즉시 동기화 |
| GET | `/api/ddns/providers` | `ddns:read` | 지원되는 DNS 서비스 제공자와 필요한 매개변수 목록 조회 |
| GET | `/api/ddns/netinterfaces` | `ddns:read` | 이 장치의 네트워크 카드 및 주소 목록 조회 |
| POST | `/api/ddns/iptest` | `ddns:write` | 공인 IP 가져오기 테스트 |
| GET | `/api/ddns/defaults` | `ddns:read` | DDNS 기본값 읽기 |
| GET | `/api/ddns/settings` | `ddns:read` | DDNS 전역 설정 읽기 |
| PUT | `/api/ddns/settings` | `ddns:write` | DDNS 전역 설정 저장 |

## STUN 터널링

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/stun/logs` | `stun:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/stun/logs` | `stun:write` | 모듈 실행 로그 비우기 |
| GET | `/api/stun/rules` | `stun:read` | STUN 규칙 목록 조회 |
| POST | `/api/stun/rules` | `stun:write` | STUN 규칙 추가 |
| PUT | `/api/stun/rules/{key}` | `stun:write` | STUN 규칙 수정 |
| DELETE | `/api/stun/rules/{key}` | `stun:write` | STUN 규칙 삭제 |
| PUT | `/api/stun/rules/{key}/enable` | `stun:write` | STUN 규칙 사용 설정/해제 |
| PUT | `/api/stun/rules-order` | `stun:write` | STUN 규칙 순서 변경 |
| POST | `/api/stun/rules/{key}/restart` | `stun:write` | STUN 규칙 재시작 |
| GET | `/api/stun/netinterfaces` | `stun:read` | 이 장치의 네트워크 카드 및 주소 목록 조회 |
| GET | `/api/stun/settings` | `stun:read` | STUN 전역 설정 읽기 |
| PUT | `/api/stun/settings` | `stun:write` | STUN 전역 설정 저장 |

## Tailscale 네트워킹

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/tailscale/logs` | `tailscale:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/tailscale/logs` | `tailscale:write` | 모듈 실행 로그 비우기 |
| GET | `/api/tailscale/settings` | `tailscale:read` | Tailscale 설정 읽기 |
| PUT | `/api/tailscale/settings` | `tailscale:write` | Tailscale 설정 저장 |
| GET | `/api/tailscale/status` | `tailscale:read` | Tailscale 연결 상태 보기 |
| POST | `/api/tailscale/restart` | `tailscale:write` | Tailscale 재시작 |
| POST | `/api/tailscale/logout` | `tailscale:write` | Tailscale 로그아웃 |

## frp 내부망 터널링

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/frp/logs` | `frp:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/frp/logs` | `frp:write` | 모듈 실행 로그 비우기 |
| GET | `/api/frp/proxies` | `frp:read` | 항목 목록 조회 |
| POST | `/api/frp/proxies` | `frp:write` | 항목 추가 |
| PUT | `/api/frp/proxies/{key}` | `frp:write` | 항목 수정 |
| DELETE | `/api/frp/proxies/{key}` | `frp:write` | 항목 삭제 |
| PUT | `/api/frp/proxies/{key}/enable` | `frp:write` | 항목 사용 설정/해제 |
| PUT | `/api/frp/proxies-order` | `frp:write` | 항목 순서 변경 |
| POST | `/api/frp/import` | `frp:write` | 기존 frpc / frps 설정 가져오기 (구버전 ini와 신버전 toml 지원). apply가 false이면 미리보기만 수행 |
| GET | `/api/frp/client` | `frp:read` | frp 클라이언트 설정 읽기 |
| PUT | `/api/frp/client` | `frp:write` | frp 클라이언트 설정 저장 |
| GET | `/api/frp/server` | `frp:read` | frp 서버 설정 읽기 |
| PUT | `/api/frp/server` | `frp:write` | frp 서버 설정 저장 |

## Cloudflare 터널

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/cloudflared/logs` | `cloudflared:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/cloudflared/logs` | `cloudflared:write` | 모듈 실행 로그 비우기 |
| GET | `/api/cloudflared/tunnels` | `cloudflared:read` | Cloudflare 터널 목록 조회 |
| POST | `/api/cloudflared/tunnels` | `cloudflared:write` | Cloudflare 터널 추가 |
| PUT | `/api/cloudflared/tunnels/{key}` | `cloudflared:write` | Cloudflare 터널 수정 |
| DELETE | `/api/cloudflared/tunnels/{key}` | `cloudflared:write` | Cloudflare 터널 삭제 |
| PUT | `/api/cloudflared/tunnels/{key}/enable` | `cloudflared:write` | Cloudflare 터널 사용 설정/해제 |
| PUT | `/api/cloudflared/tunnels-order` | `cloudflared:write` | Cloudflare 터널 순서 변경 |
| POST | `/api/cloudflared/tunnels/{key}/restart` | `cloudflared:write` | Cloudflare 터널 재시작 |
| GET | `/api/cloudflared/settings` | `cloudflared:read` | Cloudflare 터널 설정 읽기 |
| PUT | `/api/cloudflared/settings` | `cloudflared:write` | Cloudflare 터널 설정 저장 |

## SSL 인증서

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/ssl/logs` | `ssl:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/ssl/logs` | `ssl:write` | 모듈 실행 로그 비우기 |
| GET | `/api/ssl/certs` | `ssl:read` | 인증서 목록 조회 |
| POST | `/api/ssl/certs` | `ssl:write` | 인증서 추가 |
| PUT | `/api/ssl/certs/{key}` | `ssl:write` | 인증서 수정 |
| DELETE | `/api/ssl/certs/{key}` | `ssl:write` | 인증서 삭제 |
| PUT | `/api/ssl/certs/{key}/enable` | `ssl:write` | 인증서 사용 설정/해제 |
| PUT | `/api/ssl/certs-order` | `ssl:write` | 인증서 순서 변경 |
| POST | `/api/ssl/certs/{key}/renew` | `ssl:write` | 인증서 즉시 발급 또는 갱신 |
| GET | `/api/ssl/certs/{key}/download` | `ssl:write` | 인증서와 개인 키 다운로드 |
| GET | `/api/ssl/providers` | `ssl:read` | 지원되는 DNS 검증 서비스 제공자 목록 조회 |
| GET | `/api/ssl/settings` | `ssl:read` | 인증서 전역 설정 읽기 |
| PUT | `/api/ssl/settings` | `ssl:write` | 인증서 전역 설정 저장 |

## WAF 방화벽

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/waf/logs` | `waf:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/waf/logs` | `waf:write` | 모듈 실행 로그 비우기 |
| GET | `/api/waf/instances` | `waf:read` | WAF 인스턴스 목록 조회 |
| POST | `/api/waf/instances` | `waf:write` | WAF 인스턴스 추가 |
| PUT | `/api/waf/instances/{key}` | `waf:write` | WAF 인스턴스 수정 |
| DELETE | `/api/waf/instances/{key}` | `waf:write` | WAF 인스턴스 삭제 |
| PUT | `/api/waf/instances/{key}/enable` | `waf:write` | WAF 인스턴스 사용 설정/해제 |
| PUT | `/api/waf/instances-order` | `waf:write` | WAF 인스턴스 순서 변경 |
| GET | `/api/waf/crs` | `waf:read` | OWASP CRS 규칙 파일 목록 조회 |
| GET | `/api/waf/crs/{file}` | `waf:read` | CRS 규칙 파일 하나 읽기 |

## Web 서비스

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/webservice/logs` | `webservice:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/webservice/logs` | `webservice:write` | 모듈 실행 로그 비우기 |
| GET | `/api/webservice/listeners` | `webservice:read` | Web 서비스 수신 대기 목록 조회 |
| POST | `/api/webservice/listeners` | `webservice:write` | Web 서비스 수신 대기 추가 |
| PUT | `/api/webservice/listeners/{key}` | `webservice:write` | Web 서비스 수신 대기 수정 |
| DELETE | `/api/webservice/listeners/{key}` | `webservice:write` | Web 서비스 수신 대기 삭제 |
| PUT | `/api/webservice/listeners/{key}/enable` | `webservice:write` | Web 서비스 수신 대기 사용 설정/해제 |
| PUT | `/api/webservice/listeners-order` | `webservice:write` | Web 서비스 수신 대기 순서 변경 |
| GET | `/api/webservice/settings` | `webservice:read` | Web 서비스 전역 설정 읽기 |
| PUT | `/api/webservice/settings` | `webservice:write` | Web 서비스 전역 설정 저장 |
| GET | `/api/webservice/stats` | `webservice:read` | 사이트 분 단위 통계 (?listener=&site=&minutes=) |
| GET | `/api/webservice/metrics` | `webservice:read` | Prometheus 텍스트 형식 지표 |
| GET | `/api/webservice/bans` | `webservice:read` | 현재 차단된 IP |
| POST | `/api/webservice/bans` | `webservice:write` | IP 수동 차단 |
| DELETE | `/api/webservice/bans` | `webservice:write` | 차단 해제 (body는 {"key": 주소}) |
| GET | `/api/webservice/geoip` | `webservice:read` | GeoIP 데이터베이스 상태 |
| POST | `/api/webservice/geoip/upload` | `webservice:write` | GeoIP 데이터베이스 업로드 (multipart 필드 file, .mmdb) |
| POST | `/api/webservice/geoip/download` | `webservice:write` | 주소에서 GeoIP 데이터베이스(.mmdb) 다운로드 |
| DELETE | `/api/webservice/geoip` | `webservice:write` | GeoIP 데이터베이스 제거 |

## Wake-on-LAN

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/wol/logs` | `wol:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/wol/logs` | `wol:write` | 모듈 실행 로그 비우기 |
| GET | `/api/wol/devices` | `wol:read` | 장치 깨우기 목록 조회 |
| POST | `/api/wol/devices` | `wol:write` | 장치 깨우기 추가 |
| PUT | `/api/wol/devices/{key}` | `wol:write` | 장치 깨우기 수정 |
| DELETE | `/api/wol/devices/{key}` | `wol:write` | 장치 깨우기 삭제 |
| PUT | `/api/wol/devices/{key}/enable` | `wol:write` | 장치 깨우기 사용 설정/해제 |
| PUT | `/api/wol/devices-order` | `wol:write` | 장치 깨우기 순서 변경 |
| POST | `/api/wol/devices/{key}/wake` | `wol:write` | 장치에 깨우기 매직 패킷 전송 |
| GET | `/api/wol/status` | `wol:read` | 장치 온라인 상태 보기 |

## Web 터미널

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/webterminal/logs` | `webterminal:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/webterminal/logs` | `webterminal:write` | 모듈 실행 로그 비우기 |
| GET | `/api/webterminal/connections` | `webterminal:read` | 터미널 연결 목록 조회 |
| POST | `/api/webterminal/connections` | `webterminal:write` | 터미널 연결 추가 |
| PUT | `/api/webterminal/connections/{key}` | `webterminal:write` | 터미널 연결 수정 |
| DELETE | `/api/webterminal/connections/{key}` | `webterminal:write` | 터미널 연결 삭제 |
| PUT | `/api/webterminal/connections/{key}/enable` | `webterminal:write` | 터미널 연결 사용 설정/해제 |
| PUT | `/api/webterminal/connections-order` | `webterminal:write` | 터미널 연결 순서 변경 |
| DELETE | `/api/webterminal/connections/{key}/hostkey` | `webterminal:write` | 기록된 호스트 키 삭제 |
| POST | `/api/webterminal/connections/{key}/test` | `webterminal:write` | 저장된 연결 테스트 |
| POST | `/api/webterminal/connections/test` | `webterminal:write` | 임시 연결 매개변수 테스트 |
| PUT | `/api/webterminal/connections/{key}/quickaccess` | `webterminal:write` | 연결 설정 바로가기 |
| GET | `/api/webterminal/connections/{key}/ws` | `webterminal:write` | 터미널 세션 열기 (WebSocket) |
| GET | `/api/webterminal/sessions/{id}/ws` | `webterminal:write` | 기존 터미널 세션에 연결 (WebSocket) |
| GET | `/api/webterminal/sessions` | `webterminal:read` | 터미널 세션 목록 조회 |
| GET | `/api/webterminal/sessions/{id}` | `webterminal:read` | 터미널 세션 상세 정보 보기 |
| PUT | `/api/webterminal/sessions/{id}/remark` | `webterminal:write` | 터미널 세션 메모 수정 |
| GET | `/api/webterminal/sessions/{id}/stats` | `webterminal:read` | 터미널 세션 통계 보기 |
| DELETE | `/api/webterminal/sessions/{id}` | `webterminal:write` | 터미널 세션 닫기 |
| GET | `/api/webterminal/security` | 로그인 세션 전용 | 터미널 보안 설정 상태 보기 |
| POST | `/api/webterminal/security/agreement` | 로그인 세션 전용 | 터미널 이용 약관 동의 |
| POST | `/api/webterminal/security/2fa/setup` | 로그인 세션 전용 | 터미널 2단계 인증 키 생성 |
| POST | `/api/webterminal/security/2fa/enable` | 로그인 세션 전용 | 터미널 2단계 인증 켜기 |
| POST | `/api/webterminal/security/2fa/disable` | 로그인 세션 전용 | 터미널 2단계 인증 끄기 |
| POST | `/api/webterminal/security/verify2fa` | 로그인 세션 전용 | 터미널 2단계 인증 코드 확인 |
| GET | `/api/webterminal/security/check2fa` | 로그인 세션 전용 | 터미널 2단계 인증 통과 여부 확인 |
| GET | `/api/webterminal/sftp/{key}/list` | `webterminal:read` | 원격 디렉터리 목록 조회 |
| GET | `/api/webterminal/sftp/{key}/read` | `webterminal:write` | 원격 파일 읽기 |
| POST | `/api/webterminal/sftp/{key}/write` | `webterminal:write` | 원격 파일 쓰기 |
| POST | `/api/webterminal/sftp/{key}/upload` | `webterminal:write` | 원격 호스트에 파일 업로드 |
| POST | `/api/webterminal/sftp/{key}/mkdir` | `webterminal:write` | 원격 디렉터리 생성 |
| POST | `/api/webterminal/sftp/{key}/touch` | `webterminal:write` | 원격 빈 파일 생성 |
| POST | `/api/webterminal/sftp/{key}/rename` | `webterminal:write` | 원격 파일 이름 변경 또는 이동 |
| POST | `/api/webterminal/sftp/{key}/copy` | `webterminal:write` | 원격 파일 복사 |
| POST | `/api/webterminal/sftp/{key}/chmod` | `webterminal:write` | 원격 파일 권한 수정 |
| DELETE | `/api/webterminal/sftp/{key}/remove` | `webterminal:write` | 원격 파일 삭제 |
| POST | `/api/webterminal/sftp/{key}/compress` | `webterminal:write` | 원격 호스트에서 파일 압축 |
| POST | `/api/webterminal/sftp/{key}/decompress` | `webterminal:write` | 원격 호스트에서 파일 압축 해제 |
| GET | `/api/webterminal/sftp/{key}/preview-archive` | `webterminal:write` | 원격 압축 파일 내용 미리보기 |
| GET | `/api/webterminal/settings` | `webterminal:read` | Web 터미널 설정 읽기 |
| PUT | `/api/webterminal/settings` | `webterminal:write` | Web 터미널 설정 저장 |
| GET | `/api/webterminal/shells` | `webterminal:read` | 이 장치에서 사용 가능한 Shell 목록 조회 |
| GET | `/api/webterminal/globalshortcuts` | `webterminal:read` | 전역 단축 명령 읽기 |
| PUT | `/api/webterminal/globalshortcuts` | `webterminal:write` | 전역 단축 명령 저장 |
| GET | `/api/webterminal/splitlayout` | `webterminal:read` | 분할 화면 레이아웃 읽기 |
| PUT | `/api/webterminal/splitlayout` | `webterminal:write` | 분할 화면 레이아웃 저장 |
| DELETE | `/api/webterminal/splitlayout` | `webterminal:write` | 분할 화면 레이아웃 초기화 |

## 예약 작업

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/cron/logs` | `cron:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/cron/logs` | `cron:write` | 모듈 실행 로그 비우기 |
| GET | `/api/cron/tasks` | `cron:read` | 예약 작업 목록 조회 |
| POST | `/api/cron/tasks` | `cron:write` | 예약 작업 추가 |
| PUT | `/api/cron/tasks/{key}` | `cron:write` | 예약 작업 수정 |
| DELETE | `/api/cron/tasks/{key}` | `cron:write` | 예약 작업 삭제 |
| PUT | `/api/cron/tasks/{key}/enable` | `cron:write` | 예약 작업 사용 설정/해제 |
| PUT | `/api/cron/tasks-order` | `cron:write` | 예약 작업 순서 변경 |
| POST | `/api/cron/tasks/{key}/run` | `cron:write` | 예약 작업 즉시 실행 |
| POST | `/api/cron/parse` | `cron:write` | Cron 표현식 파싱 및 미리보기 |

## Docker 관리

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/docker/logs` | `docker:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/docker/logs` | `docker:write` | 모듈 실행 로그 비우기 |
| GET | `/api/docker/tasks` | `docker:read` | 백그라운드 작업 목록 조회 |
| GET | `/api/docker/tasks/{id}` | `docker:read` | 백그라운드 작업 진행 상황 및 출력 보기 |
| DELETE | `/api/docker/tasks` | `docker:write` | 종료된 백그라운드 작업 정리 |
| GET | `/api/docker/images` | `docker:read` | 이미지 목록 조회 |
| POST | `/api/docker/images/pull` | `docker:write` | 이미지 가져오기 |
| POST | `/api/docker/images/push` | `docker:write` | 이미지 푸시 |
| POST | `/api/docker/images/search` | `docker:write` | 이미지 검색 |
| POST | `/api/docker/images/load` | `docker:write` | 이미지 파일 가져오기 |
| POST | `/api/docker/images/build` | `docker:write` | 이미지 빌드 |
| GET | `/api/docker/images/{id}` | `docker:read` | 이미지 상세 정보 보기 |
| POST | `/api/docker/images/{id}/tag` | `docker:write` | 이미지에 태그 지정 |
| DELETE | `/api/docker/images/{id}` | `docker:write` | 이미지 삭제 |
| GET | `/api/docker/images/{id}/export` | `docker:write` | 이미지 내보내기 |
| GET | `/api/docker/networks` | `docker:read` | 네트워크 목록 조회 |
| POST | `/api/docker/networks` | `docker:write` | 네트워크 생성 |
| DELETE | `/api/docker/networks/{id}` | `docker:write` | 네트워크 삭제 |
| POST | `/api/docker/networks/{id}/connect` | `docker:write` | 컨테이너를 네트워크에 연결 |
| POST | `/api/docker/networks/{id}/disconnect` | `docker:write` | 컨테이너를 네트워크에서 분리 |
| GET | `/api/docker/volumes` | `docker:read` | 볼륨 목록 조회 |
| POST | `/api/docker/volumes` | `docker:write` | 데이터 볼륨 생성 |
| DELETE | `/api/docker/volumes/{name}` | `docker:write` | 데이터 볼륨 삭제 |
| GET | `/api/docker/volumes/{name}/export` | `docker:write` | 데이터 볼륨 내보내기 |
| POST | `/api/docker/volumes/{name}/import` | `docker:write` | 데이터 볼륨에 데이터 가져오기 |
| GET | `/api/docker/compose/projects` | `docker:read` | Compose 프로젝트 목록 조회 |
| POST | `/api/docker/compose/projects` | `docker:write` | Compose 프로젝트 생성 |
| POST | `/api/docker/compose/{project}/up` | `docker:write` | Compose 프로젝트 시작 |
| POST | `/api/docker/compose/{project}/pull` | `docker:write` | Compose 프로젝트 이미지 가져오기 |
| POST | `/api/docker/compose/{project}/{action}` | `docker:write` | Compose 프로젝트에 작업 수행 |
| GET | `/api/docker/compose/{project}/config` | `docker:write` | Compose 설정 파일 읽기 |
| PUT | `/api/docker/compose/{project}/config` | `docker:write` | Compose 설정 파일 저장 |
| GET | `/api/docker/compose/{project}/logs` | `docker:read` | Compose 프로젝트 로그 보기 |
| DELETE | `/api/docker/compose/{project}` | `docker:write` | Compose 프로젝트 삭제 |
| GET | `/api/docker/info` | `docker:read` | Docker 엔진 정보 보기 |
| GET | `/api/docker/disk-usage` | `docker:read` | Docker 디스크 사용량 보기 |
| POST | `/api/docker/prune` | `docker:write` | 사용하지 않는 Docker 리소스 정리 |
| GET | `/api/docker/settings` | `docker:read` | Docker 모듈 설정 읽기 |
| PUT | `/api/docker/settings` | `docker:write` | Docker 모듈 설정 저장 |
| GET | `/api/docker/containers` | `docker:read` | 컨테이너 목록 조회 |
| POST | `/api/docker/containers` | `docker:write` | 컨테이너 생성 |
| POST | `/api/docker/containers/check-all` | `docker:write` | 모든 컨테이너 이미지 업데이트 확인 |
| POST | `/api/docker/containers/upgrade-batch` | `docker:write` | 컨테이너 일괄 업그레이드 |
| POST | `/api/docker/images/upgrade-check` | `docker:write` | 이미지 새 버전 여부 확인 |
| GET | `/api/docker/containers/{id}` | `docker:read` | 컨테이너 상세 정보 보기 |
| DELETE | `/api/docker/containers/{id}` | `docker:write` | 컨테이너 삭제 |
| POST | `/api/docker/containers/{id}/edit` | `docker:write` | 컨테이너 설정을 수정하고 재생성 |
| POST | `/api/docker/containers/{id}/rename` | `docker:write` | 컨테이너 이름 변경 |
| POST | `/api/docker/containers/{id}/commit` | `docker:write` | 컨테이너를 이미지로 커밋 |
| POST | `/api/docker/containers/{id}/check` | `docker:write` | 컨테이너 이미지 업데이트 여부 확인 |
| POST | `/api/docker/containers/{id}/upgrade` | `docker:write` | 컨테이너 이미지를 업그레이드하고 재생성 |
| POST | `/api/docker/containers/{id}/{action}` | `docker:write` | 컨테이너에 작업 수행 |
| GET | `/api/docker/containers/{id}/logs` | `docker:read` | 컨테이너 로그 보기 |
| GET | `/api/docker/containers/{id}/stats` | `docker:read` | 컨테이너 리소스 사용량 보기 |
| GET | `/api/docker/containers/{id}/processes` | `docker:read` | 컨테이너 내부 프로세스 보기 |
| GET | `/api/docker/containers/{id}/files/list` | `docker:read` | 컨테이너 내부 디렉터리 목록 조회 |
| GET | `/api/docker/containers/{id}/files/read` | `docker:write` | 컨테이너 내부 파일 읽기 |
| GET | `/api/docker/containers/{id}/files/download` | `docker:write` | 컨테이너 내부 파일 다운로드 |
| POST | `/api/docker/containers/{id}/files/write` | `docker:write` | 컨테이너 내부 파일 쓰기 |
| POST | `/api/docker/containers/{id}/files/mkdir` | `docker:write` | 컨테이너 내부에 디렉터리 생성 |
| POST | `/api/docker/containers/{id}/files/rename` | `docker:write` | 컨테이너 내부 파일 이름 변경 |
| POST | `/api/docker/containers/{id}/files/upload` | `docker:write` | 컨테이너에 파일 업로드 |
| DELETE | `/api/docker/containers/{id}/files` | `docker:write` | 컨테이너 내부 파일 삭제 |
| GET | `/api/docker/containers/{id}/exec` | `docker:write` | 컨테이너 내부 터미널 열기 (WebSocket) |

## WebDAV

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/webdav/logs` | `webdav:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/webdav/logs` | `webdav:write` | 모듈 실행 로그 비우기 |
| GET | `/api/webdav/settings` | `webdav:read` | WebDAV 설정 읽기 |
| PUT | `/api/webdav/settings` | `webdav:write` | WebDAV 설정 저장 |

## FTP 서비스

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/ftp/logs` | `ftp:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/ftp/logs` | `ftp:write` | 모듈 실행 로그 비우기 |
| GET | `/api/ftp/settings` | `ftp:read` | FTP 설정 읽기 |
| PUT | `/api/ftp/settings` | `ftp:write` | FTP 설정 저장 |

## 스토리지 관리

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/storage/logs` | `storage:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/storage/logs` | `storage:write` | 모듈 실행 로그 비우기 |
| GET | `/api/storage/mounts` | `storage:read` | 항목 목록 조회 |
| POST | `/api/storage/mounts` | `storage:write` | 항목 추가 |
| PUT | `/api/storage/mounts/{key}` | `storage:write` | 항목 수정 |
| DELETE | `/api/storage/mounts/{key}` | `storage:write` | 항목 삭제 |
| PUT | `/api/storage/mounts/{key}/enable` | `storage:write` | 항목 사용 설정/해제 |
| PUT | `/api/storage/mounts-order` | `storage:write` | 항목 순서 변경 |
| GET | `/api/storage/files/{key}/list` | `storage:read` | 스토리지 디렉터리 목록 조회 |
| GET | `/api/storage/files/{key}/download` | `storage:write` | 스토리지 파일 다운로드 |
| POST | `/api/storage/files/{key}/upload` | `storage:write` | 스토리지에 파일 업로드 |
| POST | `/api/storage/files/{key}/mkdir` | `storage:write` | 스토리지 디렉터리 생성 |
| POST | `/api/storage/files/{key}/rename` | `storage:write` | 스토리지 파일 이름 변경 또는 이동 |
| DELETE | `/api/storage/files/{key}` | `storage:write` | 스토리지 파일 삭제 |

## 알림 센터

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/notify/logs` | `notify:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/notify/logs` | `notify:write` | 모듈 실행 로그 비우기 |
| GET | `/api/notify/channels` | `notify:read` | 알림 채널 목록 조회 |
| POST | `/api/notify/channels` | `notify:write` | 알림 채널 추가 |
| PUT | `/api/notify/channels/{key}` | `notify:write` | 알림 채널 수정 |
| DELETE | `/api/notify/channels/{key}` | `notify:write` | 알림 채널 삭제 |
| PUT | `/api/notify/channels/{key}/enable` | `notify:write` | 알림 채널 사용 설정/해제 |
| PUT | `/api/notify/channels-order` | `notify:write` | 알림 채널 순서 변경 |
| GET | `/api/notify/rules` | `notify:read` | 알림 규칙 목록 조회 |
| POST | `/api/notify/rules` | `notify:write` | 알림 규칙 추가 |
| PUT | `/api/notify/rules/{key}` | `notify:write` | 알림 규칙 수정 |
| DELETE | `/api/notify/rules/{key}` | `notify:write` | 알림 규칙 삭제 |
| PUT | `/api/notify/rules/{key}/enable` | `notify:write` | 알림 규칙 사용 설정/해제 |
| PUT | `/api/notify/rules-order` | `notify:write` | 알림 규칙 순서 변경 |
| GET | `/api/notify/event-types` | `notify:read` | 구독 가능한 이벤트 유형 |
| GET | `/api/notify/history` | `notify:read` | 최근 전달 기록 |
| POST | `/api/notify/channels/{key}/test` | `notify:write` | 채널로 테스트 알림 전송 |

## 네트워크 진단

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/nettools/logs` | `nettools:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/nettools/logs` | `nettools:write` | 모듈 실행 로그 비우기 |
| POST | `/api/nettools/run` | `nettools:write` | 진단 도구를 실행하고 작업을 반환합니다 (tool: ping / traceroute / tcp / dns / speed / scan) |
| GET | `/api/nettools/tasks` | `nettools:read` | 최근 진단 작업 |
| GET | `/api/nettools/tasks/{id}` | `nettools:read` | 작업 상세 및 출력 |
| DELETE | `/api/nettools/tasks/{id}` | `nettools:write` | 실행 중인 작업을 취소하거나 종료된 작업을 삭제 |
| GET | `/api/nettools/settings` | `nettools:read` | 진단 설정 |
| PUT | `/api/nettools/settings` | `nettools:write` | 탐지 금지 목록 수정 |
| GET | `/api/nettools/networks` | `nettools:read` | 이 장치에서 LAN 스캔에 사용할 수 있는 네트워크 대역 |

## 설정 백업

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/backup/logs` | `backup:read` | 모듈 실행 로그 보기 |
| DELETE | `/api/backup/logs` | `backup:write` | 모듈 실행 로그 비우기 |
| GET | `/api/backup/targets` | `backup:read` | 백업 대상 목록 조회 |
| POST | `/api/backup/targets` | `backup:write` | 백업 대상 추가 |
| PUT | `/api/backup/targets/{key}` | `backup:write` | 백업 대상 수정 |
| DELETE | `/api/backup/targets/{key}` | `backup:write` | 백업 대상 삭제 |
| PUT | `/api/backup/targets/{key}/enable` | `backup:write` | 백업 대상 사용 설정/해제 |
| PUT | `/api/backup/targets-order` | `backup:write` | 백업 대상 순서 변경 |
| GET | `/api/backup/settings` | `backup:read` | 백업 설정 |
| PUT | `/api/backup/settings` | `backup:write` | 백업 설정 수정 (암호에 자리 표시자를 전달하면 기존 값 유지) |
| POST | `/api/backup/run` | `backup:write` | 지금 백업 실행 |
| GET | `/api/backup/status` | `backup:read` | 최근 백업 결과와 다음 실행 시간 |
| GET | `/api/backup/mounts` | `backup:read` | 백업 대상으로 사용할 수 있는 스토리지 마운트 지점 |
| GET | `/api/backup/local` | 로그인 세션 전용 | 로컬 백업 파일 목록 |
| GET | `/api/backup/local/{name}` | 로그인 세션 전용 | 로컬 백업 파일 다운로드 |

## 시스템

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/session` | 유효한 모든 토큰 | 현재 인증된 ID 보기 |
| GET | `/api/status` | `status:read` | 시스템 및 각 모듈 상태 보기 |
| GET | `/api/logs` | `logs:read` | 시스템 로그 보기 |
| GET | `/api/system/settings` | 로그인 세션 전용 | 전역 설정 읽기 |
| PUT | `/api/system/settings` | 로그인 세션 전용 | 전역 설정 저장 |
| GET | `/api/system/info` | `status:read` | 시스템 정보 보기 |
| GET | `/api/system/update` | `status:read` | 새 버전 확인 |
| GET | `/api/system/backup` | 로그인 세션 전용 | 설정 백업 다운로드 |
| POST | `/api/system/restore` | 로그인 세션 전용 | 백업에서 설정 복원 |
| POST | `/api/system/restart` | 로그인 세션 전용 | Gately 재시작 |
| GET | `/api/openapi.json` | 유효한 모든 토큰 | 이 API의 OpenAPI 설명 가져오기 |

## 설정 가져오기

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| POST | `/api/import/lucky` | `import:write` | Lucky 설정 가져오기 |

## API 토큰

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/api/tokens` | 로그인 세션 전용 | API 토큰 목록 조회 |
| GET | `/api/tokens/scopes` | 로그인 세션 전용 | 권한을 부여할 수 있는 권한 영역 목록 조회 |
| POST | `/api/tokens` | 로그인 세션 전용 | API 토큰 생성 |
| PUT | `/api/tokens/{id}` | 로그인 세션 전용 | API 토큰의 이름, 권한, 만료 시간 및 활성 상태 수정 |
| POST | `/api/tokens/{id}/regenerate` | 로그인 세션 전용 | 토큰 평문 재생성 |
| DELETE | `/api/tokens/{id}` | 로그인 세션 전용 | API 토큰 삭제 |

## 계정

| 메서드 | 경로 | 필요 권한 | 설명 |
| --- | --- | --- | --- |
| PUT | `/api/account` | 로그인 세션 전용 | 관리자 사용자 이름과 비밀번호 수정 |
| GET | `/api/account/totp` | 로그인 세션 전용 | 2단계 인증 상태 보기 |
| POST | `/api/account/totp/setup` | 로그인 세션 전용 | 2단계 인증 키 생성 |
| POST | `/api/account/totp/enable` | 로그인 세션 전용 | 2단계 인증 켜기 |
| POST | `/api/account/totp/disable` | 로그인 세션 전용 | 2단계 인증 끄기 |

## 데이터 구조

### backup.LocalFile

| 필드 | 타입 |
| --- | --- |
| `modTime` | 시간 (RFC 3339) |
| `name` | string |
| `size` | integer |

### backup.Result

| 필드 | 타입 |
| --- | --- |
| `error` | string |
| `file` | string |
| `ok` | boolean |
| `pruned` | integer |
| `target` | string |

### backup.Settings

| 필드 | 타입 |
| --- | --- |
| `enable` | boolean |
| `keep` | integer |
| `localEnable` | boolean |
| `password` | string |
| `schedule` | string |

### backup.Status

| 필드 | 타입 |
| --- | --- |
| `lastRun` | 시간 (RFC 3339) |
| `manual` | boolean |
| `nextRun` | 시간 (RFC 3339) |
| `results` | Array of `backup.Result` |
| `running` | boolean |

### backup.Target

| 필드 | 타입 |
| --- | --- |
| `dir` | string |
| `enable` | boolean |
| `key` | string |
| `mount` | string |
| `name` | string |

### backup.settingsView

| 필드 | 타입 |
| --- | --- |
| `enable` | boolean |
| `hasPassword` | boolean |
| `keep` | integer |
| `localEnable` | boolean |
| `password` | string |
| `schedule` | string |

### cloudflared.Tunnel

| 필드 | 타입 |
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

| 필드 | 타입 |
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

| 필드 | 타입 |
| --- | --- |
| `enable` | boolean |

### crud.orderRequest

| 필드 | 타입 |
| --- | --- |
| `keys` | Array of string |

### ddns.Record

| 필드 | 타입 |
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

| 필드 | 타입 |
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
| `records` | Array of `ddns.Record` |
| `remark` | string |
| `webhook` | `webhook.Config` |

### dnsprovider.Credentials

| 필드 | 타입 |
| --- | --- |
| `extra` | Map of string |
| `id` | string |
| `secret` | string |

### events.TypeInfo

| 필드 | 타입 |
| --- | --- |
| `level` | string |
| `name` | string |
| `type` | string |

### frp.ClientSettings

| 필드 | 타입 |
| --- | --- |
| `binPath` | string |
| `enable` | boolean |
| `protocol` | string |
| `serverAddr` | string |
| `serverPort` | integer |
| `token` | string |
| `user` | string |

### frp.ImportRequest

| 필드 | 타입 |
| --- | --- |
| `apply` | boolean |
| `content` | string |
| `proxyMode` | string |
| `role` | string |

### frp.ImportResponse

| 필드 | 타입 |
| --- | --- |
| `added` | integer |
| `applied` | boolean |
| `client` | `frp.ClientSettings` |
| `proxies` | Array of `frp.Proxy` |
| `role` | string |
| `server` | `frp.ServerSettings` |
| `warnings` | Array of string |

### frp.Proxy

| 필드 | 타입 |
| --- | --- |
| `customDomains` | Array of string |
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

| 필드 | 타입 |
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

| 필드 | 타입 |
| --- | --- |
| `buildTime` | 시간 (RFC 3339) |
| `loaded` | boolean |
| `size` | integer |
| `type` | string |

### ipsource.Source

| 필드 | 타입 |
| --- | --- |
| `command` | string |
| `enable` | boolean |
| `interface` | string |
| `regex` | string |
| `type` | string |
| `urls` | Array of string |

### nettools.Request

| 필드 | 타입 |
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

| 필드 | 타입 |
| --- | --- |
| `deny` | Array of string |

### nettools.Task

| 필드 | 타입 |
| --- | --- |
| `error` | string |
| `finished` | 시간 (RFC 3339) |
| `id` | string |
| `lines` | Array of string |
| `result` | 모두 |
| `started` | 시간 (RFC 3339) |
| `status` | string |
| `title` | string |
| `tool` | string |

### notify.Channel

| 필드 | 타입 |
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

| 필드 | 타입 |
| --- | --- |
| `channel` | string |
| `error` | string |
| `ok` | boolean |
| `rule` | string |
| `time` | 시간 (RFC 3339) |
| `title` | string |
| `type` | string |

### notify.Rule

| 필드 | 타입 |
| --- | --- |
| `channels` | Array of string |
| `enable` | boolean |
| `events` | Array of string |
| `key` | string |
| `minLevel` | string |
| `name` | string |
| `silence` | integer |

### portforward.Rule

| 필드 | 타입 |
| --- | --- |
| `enable` | boolean |
| `key` | string |
| `listenAddress` | string |
| `listenPorts` | string |
| `maxConnections` | integer |
| `name` | string |
| `protocols` | Array of string |
| `remark` | string |
| `targetAddresses` | Array of string |
| `targetPorts` | string |
| `udpSessionTimeout` | integer |

### ssl.ACMEConfig

| 필드 | 타입 |
| --- | --- |
| `ca` | string |
| `credentials` | `dnsprovider.Credentials` |
| `directoryUrl` | string |
| `disableAuthoritativeCheck` | boolean |
| `domains` | Array of string |
| `eabHmac` | string |
| `eabKid` | string |
| `email` | string |
| `keyType` | string |
| `propagationTimeout` | integer |
| `provider` | string |
| `resolvers` | Array of string |
| `skipPropagationCheck` | boolean |
| `zone` | string |

### ssl.Cert

| 필드 | 타입 |
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
| `updatedAt` | 시간 (RFC 3339) |

### ssl.Mapping

| 필드 | 타입 |
| --- | --- |
| `dir` | string |
| `enable` | boolean |
| `script` | string |

### storage.Mount

| 필드 | 타입 |
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

| 필드 | 타입 |
| --- | --- |
| `enable` | boolean |
| `key` | string |
| `name` | string |
| `readOnly` | boolean |
| `type` | string |

### stun.Bind

| 필드 | 타입 |
| --- | --- |
| `interface` | string |
| `ip` | string |
| `mode` | string |
| `regex` | string |

### stun.Rule

| 필드 | 타입 |
| --- | --- |
| `autoRetry` | boolean |
| `bind` | `stun.Bind` |
| `disableCheck` | boolean |
| `disableForward` | boolean |
| `enable` | boolean |
| `heartbeatInterval` | integer |
| `keepAliveServers` | Array of string |
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
| `stunServers` | Array of string |
| `stunTimeout` | integer |
| `targetAddresses` | Array of string |
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

| 필드 | 타입 |
| --- | --- |
| `enable` | boolean |
| `inboundThreshold` | integer |
| `key` | string |
| `mode` | string |
| `name` | string |
| `outboundThreshold` | integer |
| `paranoia` | integer |
| `remark` | string |
| `ruleSets` | Array of `waf.RuleSet` |

### waf.RuleSet

| 필드 | 타입 |
| --- | --- |
| `enable` | boolean |
| `param` | string |
| `remark` | string |
| `type` | string |

### webhook.Config

| 필드 | 타입 |
| --- | --- |
| `body` | string |
| `enable` | boolean |
| `headers` | Array of string |
| `insecureSkipVerify` | boolean |
| `method` | string |
| `successContent` | Array of string |
| `timeout` | integer |
| `url` | string |

### webservice.BanInfo

| 필드 | 타입 |
| --- | --- |
| `key` | string |
| `manual` | boolean |
| `reason` | string |
| `until` | 시간 (RFC 3339) |

### webservice.CacheConfig

| 필드 | 타입 |
| --- | --- |
| `enable` | boolean |
| `maxMb` | integer |
| `ttl` | integer |

### webservice.HealthCheck

| 필드 | 타입 |
| --- | --- |
| `enable` | boolean |
| `fails` | integer |
| `interval` | integer |
| `mode` | string |
| `passes` | integer |
| `path` | string |
| `timeout` | integer |

### webservice.Listener

| 필드 | 타입 |
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
| `sites` | Array of `webservice.Site` |
| `tls` | boolean |

### webservice.PathRule

| 필드 | 타입 |
| --- | --- |
| `auth` | string |
| `ipList` | Array of string |
| `ipMode` | string |
| `prefix` | string |

### webservice.Point

| 필드 | 타입 |
| --- | --- |
| `bytes` | integer |
| `requests` | integer |
| `s4xx` | integer |
| `s5xx` | integer |
| `t` | integer |

### webservice.RateLimit

| 필드 | 타입 |
| --- | --- |
| `burst` | integer |
| `enable` | boolean |
| `maxConns` | integer |
| `rate` | number |

### webservice.Rewrite

| 필드 | 타입 |
| --- | --- |
| `code` | integer |
| `pattern` | string |
| `redirect` | boolean |
| `to` | string |

### webservice.ShareLink

| 필드 | 타입 |
| --- | --- |
| `expiresAt` | 시간 (RFC 3339) |
| `key` | string |
| `maxUses` | integer |
| `name` | string |
| `pathPrefix` | string |
| `token` | string |

### webservice.Site

| 필드 | 타입 |
| --- | --- |
| `accessLog` | boolean |
| `authOptional` | boolean |
| `basicAuth` | boolean |
| `cache` | `webservice.CacheConfig` |
| `compress` | boolean |
| `contentType` | string |
| `cors` | boolean |
| `countries` | Array of string |
| `countryMode` | string |
| `disableKeepAlives` | boolean |
| `domains` | Array of string |
| `enable` | boolean |
| `forwardedHeaders` | boolean |
| `health` | `webservice.HealthCheck` |
| `insecureSkipVerify` | boolean |
| `ipList` | Array of string |
| `ipMode` | string |
| `keepPath` | boolean |
| `key` | string |
| `lbPolicy` | string |
| `listDir` | boolean |
| `name` | string |
| `oauthUsers` | Array of string |
| `rateLimit` | `webservice.RateLimit` |
| `redirectCode` | integer |
| `redirectTo` | string |
| `remark` | string |
| `requestHeaders` | Array of string |
| `responseHeaders` | Array of string |
| `rewrites` | Array of `webservice.Rewrite` |
| `root` | string |
| `rules` | Array of `webservice.PathRule` |
| `sendProxyProtocol` | boolean |
| `shareLinks` | Array of `webservice.ShareLink` |
| `statusCode` | integer |
| `targets` | Array of string |
| `text` | string |
| `type` | string |
| `useTargetHost` | boolean |
| `users` | Array of string |
| `waf` | string |
| `webAuth` | boolean |

### webservice.SiteStatsView

| 필드 | 타입 |
| --- | --- |
| `bytesOut` | integer |
| `series` | Array of `webservice.Point` |
| `status` | Map of integer |
| `topCountries` | Array of `webservice.TopItem` |
| `topIps` | Array of `webservice.TopItem` |
| `upstreams` | Array of `webservice.UpstreamStatus` |

### webservice.TopItem

| 필드 | 타입 |
| --- | --- |
| `count` | integer |
| `key` | string |

### webservice.UpstreamStatus

| 필드 | 타입 |
| --- | --- |
| `backup` | boolean |
| `conns` | integer |
| `healthy` | boolean |
| `url` | string |
| `weight` | integer |

### webservice.banRequest

| 필드 | 타입 |
| --- | --- |
| `key` | string |
| `seconds` | integer |

### webservice.geoipDownload

| 필드 | 타입 |
| --- | --- |
| `url` | string |

### webterminal.Connection

| 필드 | 타입 |
| --- | --- |
| `enable` | boolean |
| `key` | string |
| `local` | `webterminal.LocalConfig` |
| `name` | string |
| `quickAccessDirs` | Array of string |
| `remark` | string |
| `ssh` | `webterminal.SSHConfig` |
| `telnet` | `webterminal.TelnetConfig` |
| `type` | string |

### webterminal.LocalConfig

| 필드 | 타입 |
| --- | --- |
| `env` | Map of string |
| `shell` | string |
| `workDir` | string |

### webterminal.Proxy

| 필드 | 타입 |
| --- | --- |
| `enabled` | boolean |
| `host` | string |
| `password` | string |
| `port` | integer |
| `type` | string |
| `username` | string |

### webterminal.SSHConfig

| 필드 | 타입 |
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

| 필드 | 타입 |
| --- | --- |
| `host` | string |
| `password` | string |
| `port` | integer |
| `proxy` | `webterminal.Proxy` |
| `timeout` | integer |
| `username` | string |

### wol.Device

| 필드 | 타입 |
| --- | --- |
| `broadcastIps` | Array of string |
| `checkAddr` | string |
| `enable` | boolean |
| `key` | string |
| `macs` | Array of string |
| `name` | string |
| `port` | integer |
| `remark` | string |
| `repeat` | integer |

