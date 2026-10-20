---
title: "다운로드"
description: "Gately 최신 바이너리와 fnOS 패키지를 내려받거나 Docker 이미지로 배포하세요."
---

# Gately 다운로드

## Docker 이미지 (권장)

이미지는 amd64, arm64, arm/v7을 모두 제공합니다.

```bash
docker pull ghcr.io/wangwanjie/gately:latest
```

전체 배포 옵션은 [설치 및 배포](/ko/docs/guide/install)를 참고하세요.

## 바이너리

사용하는 플랫폼용 압축 파일을 내려받아 압축을 풀고 `./gately -c /etc/gately`를 실행합니다. Linux(amd64 / arm64 / arm / mips / mipsle / riscv64), macOS, Windows, FreeBSD를 지원합니다.

<DownloadList />

::: tip 다운로드가 느린가요?
일부 네트워크 환경에서는 GitHub 접속이 느릴 수 있습니다. NAS와 소프트 라우터 사용자는 프록시를 통해 먼저 받아 오거나 Docker 이미지를 사용하세요.
:::

## fnOS 네이티브 앱 {#fnos}

fnOS 사용자는 네이티브 앱을 바로 설치할 수 있습니다. 위 목록의 fnOS 그룹에서 `.fpk`를 내려받은 뒤(Intel / AMD는 `x86`, ARM 장치는 `arm`) fnOS의 「앱 센터 → 수동 설치」에서 선택하세요.

- fnOS 통합 게이트웨이 연동: 바탕 화면 아이콘으로 열 수 있고 fnOS 원격 도메인으로도 접속할 수 있으며 포트를 따로 열 필요가 없습니다.
- fnOS 로그인 상태로 관리자 자동 로그인: 켜 두면 fnOS에 이미 로그인한 관리자는 Gately 비밀번호와 2단계 인증 코드를 건너뜁니다.

자세한 내용은 [설치 및 배포 · fnOS 네이티브 앱](/ko/docs/guide/install#fnos)을 참고하세요.

## 소스에서 빌드

Go 1.26+, Node.js 24+, pnpm이 필요합니다.

```bash
git clone https://github.com/wangwanjie/gately.git
cd gately
make web      # 프런트엔드 빌드
make build    # bin/gately 출력
```

## 검증

모든 릴리스에는 체크섬 파일이 포함됩니다. 내려받은 뒤 실행하기 전에 SHA-256을 확인하세요.

v0.5.0부터 체크섬 파일과 Docker 이미지에 cosign 키리스 서명이 붙고, 압축 파일에는 빌드 출처 증명과 SBOM이 제공됩니다. 검증 방법은 [릴리스 검증](/en/docs/guide/verify)(영문)을 참고하세요.
