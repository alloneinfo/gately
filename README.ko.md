# Gately

[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md) | **한국어** | [Español](README.es.md) | [Français](README.fr.md) | [Deutsch](README.de.md) | [Русский](README.ru.md) | [Português (Brasil)](README.pt-BR.md)

가정 네트워크, NAS, 클라우드 호스트, 각종 서버에서 사용할 수 있는 오픈소스 · 감사 가능한 네트워크 게이트웨이 도구 모음입니다. [Lucky](https://github.com/gdy666/lucky)에서 영감을 받아 같은 종류의 기능을 완전한 오픈소스로 제공합니다. DDNS, STUN 터널링, 포트 포워딩, 리버스 프록시, 인증서, WAF 등 자주 쓰는 기능을 약 22 MB의 단일 파일에 담았으며, Lucky 설정을 한 번에 가져올 수 있습니다.

## Gately를 만든 이유

Lucky는 중국의 NAS 및 가정 네트워크 사용자에게 익숙한 훌륭한 도구로, DDNS, STUN 터널링, 리버스 프록시 등의 기능을 하나로 묶었고 Gately에도 제품 아이디어를 제공했습니다. Lucky가 없었다면 Gately도 없었을 것입니다. Lucky는 비공개 소스로 무료 배포되는 반면 Gately는 완전한 오픈소스를 택하여, 사용자와 커뮤니티가 소스 코드와 설정을 직접 검토할 수 있습니다. 네트워크 경계에서 동작하며 DNS 키와 인증서 개인 키를 다루는 소프트웨어라면, 오픈소스를 통해 사용자가 그 동작을 직접 확인할 수 있어야 합니다. Gately의 약속은 다음과 같습니다.

- 소스 코드는 Apache-2.0으로 공개되며 커뮤니티의 검토를 받습니다
- 기본적으로 프로젝트 측 서버에 연결하지 않고 기기 정보도 전송하지 않습니다. 서드파티 로그인은 직접 만든 OAuth 앱을 사용하며 중계 서버를 거치지 않습니다
- 모든 외부 주소(공인 IP 조회, STUN 서버 등)를 확인하고 변경하고 끌 수 있습니다
- 설정은 평문 JSON이므로 git에 넣어 감사와 롤백에 활용할 수 있습니다

## 기능

| 모듈 | 기능 |
| --- | --- |
| 동적 도메인 | Alibaba Cloud, Cloudflare, Tencent Cloud, DNSPod, Huawei Cloud, GoDaddy, Porkbun, dynv6, DuckDNS, 사용자 지정 콜백. URL/네트워크 카드/명령으로 IP 조회. IPv4/IPv6. Webhook |
| STUN 터널링 | TCP/UDP, keep-alive, UPnP/NAT-PMP, 내장 포워딩, 주소 변경 시 Webhook 및 스크립트 실행 |
| 포트 포워딩 | TCP/UDP, IPv4/IPv6, 포트 범위, 다중 대상 장애 조치, 연결 수 제한, 트래픽 통계 |
| SSL 인증서 | ACME DNS-01 자동 발급 및 갱신(와일드카드, EAB), 업로드, 파일 읽기, 디렉터리 매핑 및 스크립트 실행 |
| Tailscale | tailscaled 관리(유저스페이스 네트워킹, 컨테이너 친화적) 또는 시스템에 이미 있는 데몬에 연결. 인증 키/로그인 링크, Headscale, 서브넷 라우트, 출구 노드, 노드 목록과 SOCKS5 프록시, 고정 포트, tailscaled 추가 인수 및 환경 변수 |
| frp | frpc 클라이언트(tcp/udp/http/https/stcp/xtcp 프록시, 추가·삭제 즉시 적용)와 frps 서버(포트 허용 목록, 서브도메인, 대시보드). 기존 ini / 신규 toml 설정을 한 번에 가져올 수 있음 |
| Cloudflare 터널 | cloudflared 토큰 터널과 trycloudflare 임시 터널, 다중 터널, 자동 재시작, 공개 주소 표시 |
| 웹 서비스 | 도메인별 리버스 프록시/리디렉션/정적 파일/고정 텍스트. HTTPS. WebSocket. PROXY 프로토콜 송수신. 기본 인증, 웹 인증, GitHub/Google/QQ/OIDC 로그인. IP 허용/차단 목록, 경로 규칙, 임시 공유 링크, 인증 실패 시 자동 차단, GeoIP 국가 필터링. 백엔드 가중치/백업/로드 밸런싱 정책, 능동·수동 헬스 체크, 압축, 속도 제한, 응답 캐시, 경로 재작성. 분 단위 접속 통계와 Prometheus 메트릭 |
| WAF 방화벽 | Coraza + OWASP CRS v4, 파일 단위 규칙 활성화, 사용자 지정 SecLang, 차단/탐지 전용 |
| Docker 관리 | 컨테이너 생성 및 편집, 로그와 터미널, Compose, 이미지 업데이트 확인 및 원클릭 업그레이드(실패 시 자동 롤백), 네트워크와 볼륨 |
| 웹 터미널 | SSH(호스트 키 최초 신뢰), Telnet, 로컬 셸(기본 비활성). 프록시, 지속 세션, SFTP 파일 관리, 2FA 및 프로토콜 확인, 빠른 명령, 화면 분할 |
| 네트워크 진단 | Ping, 경로 추적, 포트 탐색, DNS 조회, 속도 측정, LAN 스캔 |
| 알림 센터 | Webhook, Telegram, Bark, WeCom, DingTalk, Feishu, 이메일. 수준, 와일드카드, 무음 및 중복 제거 규칙 |
| 설정 백업 | 암호화 백업(scrypt + AES-GCM), 보존 정책, 로컬 디렉터리 또는 스토리지 마운트 지점으로 예약 백업 |
| 명령줄 및 MCP | `gately ctl`로 실행 중인 인스턴스를 호출하고, `gately mcp`로 stdio MCP 서비스를 제공(기본 읽기 전용) |
| Wake-on-LAN | 매직 패킷 브로드캐스트, 온라인 탐지 |
| WebDAV | 다중 사용자, 사용자별 디렉터리, 읽기 전용 |
| FTP | 다중 사용자, 사용자별 디렉터리, 읽기 전용. FTPS 및 패시브 포트 범위 지원 |
| 스토리지 관리 | 로컬 디렉터리와 원격 WebDAV 마운트, 온라인 파일 관리 |
| 예약 작업 | cron 표현식, 명령 또는 HTTP 요청, 실행 기록 |
| 보안 및 설정 | 2단계 인증(TOTP), 로그인 실패 속도 제한, 보안 진입점, 내부망 전용 접속, 관리 UI의 HTTPS, 다중 API 토큰(모듈별 읽기/쓰기 권한), 사용자 지정 DNS(DoH 지원), 시간대, 설정 백업 및 복원 |
| 마이그레이션 | Lucky의 `.lkcf` 또는 전체 사이트 설정 zip 가져오기: 포트 포워딩, DDNS, STUN, 인증서, 리버스 프록시, WAF, Wake-on-LAN, 웹 터미널, WebDAV, 기본 설정 |

## 빠른 시작

### Docker

이미지는 amd64, arm64, arm/v7을 지원합니다.

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep -i "initial password"
```

`http://<기기 IP>:16680`을 열고 `admin`과 로그에 출력된 초기 비밀번호로 로그인한 뒤, "설정"에서 비밀번호를 변경하고 2단계 인증을 켜세요. 전체 Compose 예시는 [deploy/docker-compose.yml](deploy/docker-compose.yml)을 참고하세요.

포트 포워딩, STUN, 웹 서비스는 호스트 포트를 수신해야 하므로 `--network host`를 권장합니다. Docker 관리가 필요 없다면 `docker.sock` 마운트를 제거해도 됩니다.

### 바이너리

[Releases](https://github.com/wangwanjie/gately/releases)에서 플랫폼에 맞는 압축 파일(Linux amd64/arm64/arm/mips/mipsle/riscv64, macOS, Windows, FreeBSD)을 내려받아 압축을 풀고 실행합니다.

```bash
./gately -c /etc/gately
```

인수: `-c` 설정 디렉터리(환경 변수 `GATELY_CONFIG_DIR`로도 지정 가능), `-listen` 관리 UI 수신 주소 덮어쓰기, `-version` 버전 출력.

### API 호출

관리 콘솔의 "API 토큰" 페이지에서 토큰을 만들면(여러 개 생성 가능, 모듈별로 읽기 전용 또는 읽기/쓰기 권한 부여, 만료 시간 설정 가능, 평문은 생성 직후 한 번만 표시됨) 스크립트로 관리 API를 일괄 호출할 수 있습니다.

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks
```

계정, 2단계 인증, 토큰 관리, 그리고 백업, 복원, 재시작, 시스템 설정 API는 로그인 세션으로만 사용할 수 있습니다. 전체 API, 필요한 권한, 데이터 구조는 관리 콘솔의 "API 문서" 페이지에서 확인할 수 있으며, `make api-docs`로 [docs/api](docs/api)(`openapi.json`과 `API.md`)에 내보낼 수도 있습니다. 공식 사이트 https://gately.cn 에서도 이 파일을 그대로 참조합니다.

이전 버전에서 업그레이드하면 기존 전역 API 토큰은 자동으로 "레거시 토큰"이 됩니다(값은 그대로이며 모든 모듈 권한을 가짐). API 토큰 페이지에서 권한을 줄이거나 새 토큰으로 교체하는 것을 권장합니다.

### Tailscale, frp, cloudflared

이 세 모듈은 공식 외부 프로그램을 관리합니다(자동 재시작, 로그를 UI에 집계). Gately 본체에는 포함되어 있지 않으므로 항상 업스트림의 최신 버전을 사용할 수 있습니다. 프로그램은 다음 순서로 찾습니다. 모듈 설정에 입력한 경로, `PATH`, 설정 디렉터리 아래의 `bin/` 하위 디렉터리. Docker 배포에서는 마운트한 설정 디렉터리에 바이너리를 넣으면 됩니다(예: `./gately-data/bin/frpc`). Tailscale은 `tailscale`과 `tailscaled`를 모두 제공해야 합니다. 설정 파일의 토큰과 키는 0600 권한으로 저장되며, cloudflared 토큰은 환경 변수로 전달되어 프로세스 목록에 나타나지 않습니다.

### Lucky에서 마이그레이션

[마이그레이션 가이드](https://gately.cn/ko/docs/guide/migration)를 참고하세요. 간단한 절차: "설정 → Lucky에서 가져오기"에서 Lucky 설정 디렉터리의 `lucky_*.lkcf` 파일을 선택하고 미리보기 후 가져옵니다. 가져온 규칙은 기본적으로 비활성 상태이므로, 모듈별로 하나씩 Lucky에서 끄고 Gately에서 켜세요.

## 개발

요구 사항: Go 1.26+, Node.js 24+, pnpm.

```bash
make web      # 프런트엔드 빌드(web/dist에 출력되며 go:embed로 포함됨)
make build    # bin/gately 컴파일
make test     # go test -race ./...
make lint     # gofmt, go vet, 프런트엔드 타입 검사
```

프런트엔드 개발 시에는 `cd web && pnpm dev`를 실행하세요. `/api`는 로컬 `127.0.0.1:16680`으로 프록시됩니다.

일부 테스트는 기본적으로 건너뛰며 추가 조건이 필요합니다.

| 테스트 | 활성화 방법 |
| --- | --- |
| ACME 엔드투엔드(Pebble) | pebble과 pebble-challtestsrv를 설치하고 `GATELY_PEBBLE_BIN`, `GATELY_PEBBLE_SRC`를 설정(CI에서는 자동 실행) |
| 실제 공용 STUN 서버 | `GATELY_NET_TEST=1` |
| 실제 Lucky 설정으로 가져오기 검증 | `GATELY_LUCKY_SAMPLE` 등, 각 모듈의 `TestImportRealSample` 참고 |

디렉터리 구조와 기술 선정은 [docs/ROADMAP.md](docs/ROADMAP.md)를 참고하세요.

공식 사이트([gately.cn](https://gately.cn))의 소스는 `website/`에 있으며 VitePress 기반입니다. `cd website && pnpm install && pnpm dev`; `pnpm deploy:site`는 빌드 후 서버에 동기화합니다.

## 기여 규칙

- Lucky와의 호환성은 공개적으로 관찰 가능한 동작, 인터페이스, 설정 형식에 기반합니다. Lucky 바이너리나 프런트엔드 산출물에서 복사한 코드는 제출하지 마세요.
- 새 모듈에는 설정 구조와 검증, 런타임 시작/중지, REST 인터페이스, 단위 테스트가 필요합니다. Lucky에서 마이그레이션할 수 있는 모듈은 `luckyimport.Importer`를 구현하고, 개요 페이지에 상태를 표시하려는 모듈은 `module.Summarizer`를 구현합니다.
- 코드 주석은 중국어로 작성하며, 제어문에는 반드시 중괄호를 사용해야 합니다.

## Star 추이

[![Star History Chart](https://api.star-history.com/svg?repos=wangwanjie/gately&type=Date)](https://star-history.com/#wangwanjie/gately&Date)

## 라이선스

[Apache License 2.0](LICENSE)
