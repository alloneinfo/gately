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

## 최초 로그인

관리 UI는 기본적으로 `16680` 포트에서 대기하며 사용자 이름은 `admin`입니다. 초기 비밀번호는 최초 실행 시 로그에 출력됩니다. 자동화 배포에서는 환경 변수 `GATELY_INITIAL_PASSWORD`로 초기 비밀번호를 지정할 수 있으며, 이때는 무작위 비밀번호를 생성하지 않습니다.

처음 로그인하면 시작 안내가 나타나 비밀번호 변경, 2단계 인증 또는 패스키 설정, 네트워크 자가 점검 실행, Lucky에서 가져오기, 알림 설정을 차례로 안내합니다.

## 업그레이드

Gately에는 온라인 자체 업데이트 기능이 없으며 업그레이드는 배포 방식에 맡깁니다.

- Docker: 새 이미지를 `docker pull`한 뒤 컨테이너를 다시 만듭니다(설정 디렉터리 마운트만 유지하면 됩니다).
- 바이너리: 실행 파일을 교체하고 서비스를 재시작합니다.

업그레이드 전에 설정 백업을 내보내거나 설정 디렉터리를 git으로 관리해 두는 것을 권장합니다. 내려받은 압축 파일과 이미지는 [릴리스 검증](/en/docs/guide/verify)에 따라 서명을 확인할 수 있습니다.
