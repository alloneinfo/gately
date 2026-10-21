# 자주 묻는 질문

## 프로젝트 서버에 연결해야 하나요?

연결할 필요가 없습니다. Gately는 기본적으로 어떤 프로젝트 서버에도 연결하지 않으며 기기 정보도 보고하지 않습니다. 공인 IP 조회 같은 외부 주소는 서드파티 공개 서비스를 사용하며 해당 작업에서 수정할 수 있습니다.

## 초기 비밀번호는 어디에 있나요?

최초 실행 시 로그에 한 번만 출력됩니다. Docker로 배포했다면 `docker logs gately 2>&1 | grep -i "initial password"`를 실행하세요.

## 관리자 비밀번호를 잊어버렸습니다

Gately를 중지하고 설정 디렉터리의 `account.json`을 삭제한 뒤 다시 시작하세요. 프로그램이 `admin` 계정을 새로 만들고 새 초기 비밀번호를 로그에 출력합니다. 로그인한 뒤에는 2단계 인증 등 보안 설정을 확인하세요.

## fnOS를 지원하나요?

지원합니다. 모든 릴리스에 fnOS용 오프라인 패키지(`.fpk`)가 포함되며 앱 센터에서 수동으로 설치합니다. Gately는 fnOS 통합 게이트웨이에 연결되므로 바탕 화면 아이콘으로 열 수 있고 fnOS 원격 도메인으로도 접속할 수 있습니다. 「설정 → 접근 제어」에서 「fnOS 로그인 상태로 자동 로그인」을 켜면 로그인한 fnOS 관리자는 비밀번호나 2단계 인증 코드 없이 바로 들어갑니다. 자세한 내용은 [fnOS 네이티브 앱](/ko/docs/guide/install#fnos)을 참고하세요.

## OpenWrt / iStoreOS를 지원하나요?

지원합니다. 모든 릴리스에 OpenWrt 패키지가 포함됩니다. OpenWrt 24.10 이하(iStoreOS 포함)는 `.ipk`를 `opkg install`로, OpenWrt 25.12 이상은 `.apk`를 `apk add --allow-untrusted`로 설치합니다. 설치 후 바로 시작되며 설정은 `/etc/gately`에 저장되고, 업그레이드는 새 패키지를 설치하면 됩니다. 바이너리가 약 35 MB이므로 저장 공간이 넉넉한 소프트 라우터에 적합합니다. 자세한 내용은 [OpenWrt](/ko/docs/guide/install#openwrt)를 참고하세요.

## 왜 `--network host`를 권장하나요?

포트 포워딩, STUN, 웹 서비스는 호스트 포트를 직접 수신해야 합니다. 브리지 네트워크를 쓰면 사용하는 포트를 모두 직접 매핑해야 합니다.

## Lucky와 동시에 실행할 수 있나요?

가능합니다. 관리 포트는 기본 16680이라 Lucky의 16601과 충돌하지 않습니다. 가져온 규칙은 기본적으로 비활성화되므로 아직 실행 중인 Lucky와 포트 및 라우터 UPnP 매핑을 두고 충돌하지 않습니다.

## Tailscale, frp, cloudflared는 왜 바이너리를 직접 넣어야 하나요?

이 세 모듈은 공식 외부 프로그램을 관리하며 Gately 본체에는 내장하지 않습니다. 덕분에 항상 업스트림 최신 버전을 쓸 수 있습니다. 배치 위치는 [설정 및 실행 옵션](/ko/docs/guide/configuration)을 참고하세요.

## STUN 규칙을 활성화했는데도 외부에서 접속할 수 없습니다

라우터 WAN 포트가 통신사급 NAT(100.64.0.0/10) 뒤에 있으면 외부에서 통과 포트에 접근할 수 없으며, 이는 네트워크 환경 자체의 제약입니다. [Tailscale / frp / Cloudflare 터널](/ko/docs/modules/tunnels)을 대신 사용해 보세요.

## 문제를 제보하거나 개발에 참여하려면 어떻게 하나요?

[GitHub Issues](https://github.com/wangwanjie/gately/issues)에 문제를 등록하세요. 개발 규칙은 [기여 가이드](/ko/docs/guide/contributing)를 참고하세요.
