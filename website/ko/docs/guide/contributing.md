# 기여 가이드

Gately는 커뮤니티가 이끄는 오픈소스 프로젝트이며 Issue와 Pull Request를 환영합니다. 저장소: <https://github.com/wangwanjie/gately>

## 로컬 개발

필요한 도구: Go 1.26+, Node.js 24+, pnpm.

```bash
make web      # 프런트엔드 빌드 (web/dist에 출력되며 go:embed로 포함됨)
make build    # bin/gately 컴파일
make test     # go test -race ./...
make lint     # gofmt, go vet, 프런트엔드 타입 검사
```

프런트엔드를 개발할 때는 `cd web && pnpm dev`를 실행하세요. `/api`가 로컬 `127.0.0.1:16680`으로 프록시됩니다.

## 기여 규칙

- Lucky와의 호환성은 공개적으로 관찰 가능한 동작, 인터페이스, 설정 포맷을 기준으로 합니다. Lucky 바이너리나 프런트엔드 산출물에서 복사한 코드는 제출하지 마세요.
- 새 모듈에는 설정 구조와 검증, 런타임 시작/중지, REST 엔드포인트, 단위 테스트가 필요합니다. Lucky에서 마이그레이션할 수 있는 모듈은 `luckyimport.Importer`를 구현하고, 개요 페이지에 상태를 표시하려는 모듈은 `module.Summarizer`를 구현합니다.
- 코드 주석은 중국어로 작성하며 제어문에는 반드시 중괄호를 사용합니다.

## 추가 테스트

일부 테스트는 기본적으로 건너뛰며 추가 조건이 필요합니다.

| 테스트 | 활성화 방법 |
| --- | --- |
| ACME 엔드 투 엔드 (Pebble) | pebble과 pebble-challtestsrv를 설치하고 `GATELY_PEBBLE_BIN`, `GATELY_PEBBLE_SRC`를 설정 |
| 실제 공개 STUN 서버 | `GATELY_NET_TEST=1` |
| 실제 Lucky 설정으로 가져오기 검증 | `GATELY_LUCKY_SAMPLE` 등, 각 모듈의 `TestImportRealSample` 참고 |

## 라이선스

[Apache License 2.0](https://github.com/wangwanjie/gately/blob/main/LICENSE)
