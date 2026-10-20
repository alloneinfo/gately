# 설치 및 배포

## Docker

이미지 주소는 `ghcr.io/wangwanjie/gately`이며 amd64, arm64, arm/v7을 지원합니다.

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
```

### Docker Compose

```yaml
services:
  gately:
    image: ghcr.io/wangwanjie/gately:latest
    container_name: gately
    restart: unless-stopped
    # 포트 포워딩, STUN 통과, 웹 서비스는 모두 호스트 포트를 직접 수신해야 합니다
    network_mode: host
    volumes:
      # 설정 디렉터리 (일반 JSON, git 백업 가능. ssl.json에는 인증서 개인 키가 들어 있습니다)
      - ./data:/etc/gately
      # Docker 관리 (필요 없으면 삭제)
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TZ=Asia/Shanghai
```

이미지에는 `sh`, `bash`, `curl`이 내장되어 있어 일반적인 알림 스크립트를 바로 실행할 수 있습니다.

## 바이너리

1. [다운로드 페이지](/ko/download)에서 사용하는 플랫폼용 압축 파일을 받습니다. Linux(amd64 / arm64 / arm / mips / mipsle / riscv64), macOS, Windows, FreeBSD를 지원합니다.
2. 압축을 풀고 실행합니다.

```bash
./gately -c /etc/gately
```

### systemd

저장소에 완성된 서비스 파일 [deploy/systemd/gately.service](https://github.com/wangwanjie/gately/blob/main/deploy/systemd/gately.service)가 있으며, 낮은 번호의 포트를 수신하는 데 필요한 권한(capability)이 이미 부여되어 있습니다.

```bash
sudo install -m 0755 gately /usr/local/bin/gately
sudo install -m 0644 gately.service /etc/systemd/system/gately.service
sudo systemctl daemon-reload && sudo systemctl enable --now gately
journalctl -u gately | grep -i "initial password"
```

### OpenWrt

procd 시작 스크립트 [deploy/openwrt/gately.init](https://github.com/wangwanjie/gately/blob/main/deploy/openwrt/gately.init)를 사용합니다.

```bash
cp gately /usr/bin/gately && chmod +x /usr/bin/gately
cp gately.init /etc/init.d/gately && chmod +x /etc/init.d/gately
/etc/init.d/gately enable && /etc/init.d/gately start
```

## fnOS 네이티브 앱 {#fnos}

모든 릴리스에는 fnOS용 오프라인 패키지 `gately_<version>_fnos_x86.fpk`(Intel / AMD)와 `gately_<version>_fnos_arm.fpk`(ARM)가 포함됩니다. 공식 앱 센터에는 등록되어 있지 않으므로 다음과 같이 직접 설치합니다.

1. [다운로드 페이지](/ko/download#fnos)에서 장치 아키텍처에 맞는 `.fpk`를 받습니다.
2. fnOS에서 「앱 센터 → 수동 설치」(왼쪽 아래 버튼)를 열고 파일을 선택한 뒤 마법사를 완료합니다.
3. fnOS 바탕 화면의 Gately 아이콘을 클릭해 관리 UI를 엽니다.

### fnOS 통합 게이트웨이 연동

바탕 화면 아이콘은 fnOS 통합 게이트웨이를 거쳐 fnOS에 접속하는 주소의 `/app/gately/`에서 관리 UI를 열며, 같은 진입점을 함께 씁니다. 내부망 주소와 fnOS 원격 도메인 모두 사용할 수 있고 16680 포트를 열거나 포워딩할 필요가 없습니다.

게이트웨이를 통한 접근은 fnOS가 제어하며 fnOS 관리자만 열 수 있습니다. 보안 진입 경로와 「내부망 접근만 허용」은 16680 포트로 직접 접속할 때만 적용됩니다. 내부망에서는 `http://<NAS 주소>:16680`도 계속 사용할 수 있습니다.

### fnOS 관리자 로그인 상태로 로그인

「설정 → 접근 제어」에서 「fnOS 로그인 상태로 자동 로그인」을 켜면 fnOS에 로그인한 관리자는 비밀번호나 2단계 인증 코드 없이 바탕 화면 아이콘에서 바로 Gately에 들어갑니다.

- fnOS 관리자만 Gately 관리자로 로그인됩니다. 일반 fnOS 사용자는 여전히 사용자 이름과 비밀번호를 입력해야 합니다.
- 통합 게이트웨이를 통한 접근에만 적용됩니다. 16680 포트로 직접 접속할 때는 일반 로그인이 필요합니다.
- 로그아웃한 뒤에는 조용히 다시 로그인시키지 않고, 로그인 페이지에 「fnOS 계정 &lt;이름&gt;(으)로 로그인」 버튼을 표시합니다.
- 이 설정은 기본적으로 꺼져 있으며 fnOS 패키지에서만 표시됩니다.

### 참고 사항

- 앱은 `root`로 실행되므로 80/443을 수신하고 `docker.sock`으로 Docker를 관리하며 실제 웹 터미널을 열 수 있습니다. 직접 관리하는 NAS에만 설치하세요.
- 최초 실행 시 `admin`의 초기 비밀번호가 `/var/apps/gately/var/gately.log`에 기록됩니다: `grep -i "initial password" /var/apps/gately/var/gately.log`
- 설정은 앱의 `var` 디렉터리에 저장되며, 제거 마법사에서 「모두 삭제」를 선택하지 않는 한 제거해도 유지됩니다.
- 패키지는 `gately ctl`과 `gately mcp`용 `/usr/local/bin/gately` 명령도 추가합니다.
- 16680 포트는 Docker 이미지와 같습니다. 네이티브 앱을 시작하기 전에 `gately` 컨테이너를 중지하세요.

## 최초 로그인

관리 UI는 기본적으로 `16680` 포트에서 대기하며 사용자 이름은 `admin`입니다. 초기 비밀번호는 최초 실행 시 로그에 출력됩니다. 자동화 배포에서는 환경 변수 `GATELY_INITIAL_PASSWORD`로 초기 비밀번호를 지정할 수 있으며, 이때는 무작위 비밀번호를 생성하지 않습니다.

처음 로그인하면 시작 안내가 나타나 비밀번호 변경, 2단계 인증 또는 패스키 설정, 네트워크 자가 점검 실행, Lucky에서 가져오기, 알림 설정을 차례로 안내합니다.

## 업그레이드

Gately에는 온라인 자체 업데이트 기능이 없으며 업그레이드는 배포 방식에 맡깁니다.

- Docker: 새 이미지를 `docker pull`한 뒤 컨테이너를 다시 만듭니다(설정 디렉터리 마운트만 유지하면 됩니다).
- 바이너리: 실행 파일을 교체하고 서비스를 재시작합니다.
- fnOS: 앱 센터에서 새 버전의 `.fpk`를 수동으로 설치합니다. 설정은 유지됩니다.

업그레이드 전에 설정 백업을 내보내거나 설정 디렉터리를 git으로 관리해 두는 것을 권장합니다. 내려받은 압축 파일과 이미지는 [릴리스 검증](/en/docs/guide/verify)에 따라 서명을 확인할 수 있습니다.
