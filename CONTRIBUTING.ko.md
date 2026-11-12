# GQBD 기여 가이드

[English](./CONTRIBUTING.md) | **한국어**

기여해 주셔서 감사합니다. GQBD는 의존성이 없는 단일 Go 패키지이므로 1분이면 개발을 시작할 수 있습니다.

## 준비 사항

- Go 1.26.10 이상 (`go.mod`의 최소 버전)
- 선택: [golangci-lint v2](https://golangci-lint.run/), [govulncheck](https://go.dev/doc/security/vuln/), [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)

## 풀 리퀘스트를 열기 전에

CI와 같은 검사를 실행합니다.

```bash
gofmt -l .                       # must print nothing
go vet ./...
go test -race -shuffle=on ./...
golangci-lint run
govulncheck ./...
```

데이터베이스가 필요한 테스트는 없습니다. 모든 빌더는 쿼리 문자열, 인자, 요청 본문을 반환하며 테스트는 이 값을 비교합니다.

## 핫 패스 변경

`gqbd.go`, `build.go`, `dialect.go`와 다른 빌더의 `Build` 메서드는 할당(allocation)에 민감합니다. 이 파일을 수정한다면 변경 전후의 벤치마크를 비교합니다.

```bash
git stash && go test -run='^$' -bench=. -benchmem -count=6 . > /tmp/before.txt
git stash pop && go test -run='^$' -bench=. -benchmem -count=6 . > /tmp/after.txt
benchstat /tmp/before.txt /tmp/after.txt
```

`benchstat` 출력을 풀 리퀘스트에 붙여 주세요. 빌더가 사용하는 기법은 [docs/ko/performance.md](docs/ko/performance.md)를 참고합니다.

## SQL 데이터베이스 추가

데이터베이스별 차이는 `dialect.go`의 테이블 하나에 모여 있습니다. 새 SQL 데이터베이스는 보통 `dialect` 항목 하나, `DBType` 상수, `dialectFor`와 `NewQueryBuilder`의 case, `connection.go`의 연결 문자열 빌더, 테스트, 그리고 `docs/` 아래 문서(한국어 번역은 `docs/ko/` 아래)로 구성됩니다.

## 문서

영어 문서가 원본입니다. `README.md`, `CONTRIBUTING.md` 또는 `docs/`의 문서를 수정하면 같은 풀 리퀘스트에서 한국어 파일(`README.ko.md`, `CONTRIBUTING.ko.md`, `docs/ko/...`)도 수정합니다. 그럴 수 없다면 메인테이너가 처리할 수 있도록 풀 리퀘스트에 적어 주세요.

## 지속적 통합(CI)

| 워크플로 | 실행 시점 | 내용 |
|----------|-----------|------|
| `test.yml` | `main`/`dev` 푸시, 풀 리퀘스트 | `go mod tidy` 검사, gofmt, vet, 최소·최신 Go 버전에서 race 테스트, 벤치마크 스모크 실행, 퍼징, 커버리지, golangci-lint |
| `security.yml` | 푸시, 풀 리퀘스트, 매주 | govulncheck, gosec, CodeQL(security-extended), 풀 리퀘스트 의존성 검토 |
| `release.yml` | `v*.*.*` 태그 | 테스트와 govulncheck를 다시 실행하고, 자동 생성된 릴리스 노트로 GitHub Release를 만들고, Go 모듈 프록시에 버전을 게시 |

## 릴리스 (메인테이너)

GQBD는 [유의적 버전](https://semver.org/lang/ko/)을 따릅니다. `main`의 커밋에 태그를 달고 푸시합니다.

```bash
git tag v1.6.0
git push origin v1.6.0
```

나머지는 릴리스 워크플로가 처리합니다. 접미사가 붙은 태그(`v1.6.0-rc.1`)는 프리릴리스로 표시됩니다.
