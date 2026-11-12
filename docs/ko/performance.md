# 성능

[English](../performance.md) | **한국어**

GQBD는 이를 사용하는 서비스의 요청 경로에 놓이므로, 쿼리를 만드는 비용은 쿼리를 실행하는 비용보다 훨씬 작아야 합니다. 이 문서는 빌더가 할당(allocation)을 적게 유지하는 방법과 빌더를 효율적으로 사용하는 방법을 설명합니다.

## SQL 빌더가 쿼리를 만드는 방식

1. **식별자는 한 번 파싱하고 한 번 씁니다.** `BuildSelect`, `LeftJoin`, `GroupBy`, `OrderBy` 같은 체이닝 메서드는 테이블과 컬럼 이름을 검증한 뒤 파싱된 형태(접두사, 이름, 별칭)로 저장합니다. 이 시점에는 인용(quoting)이나 문자열 연결이 일어나지 않습니다. `Build`가 인용된 이름을 출력 버퍼에 바로 쓰므로 `u.id`나 `users AS u`를 이스케이프해도 할당이 발생하지 않습니다.
2. **미리 크기를 정한 버퍼 하나.** `Build`는 저장된 부분들로 최종 길이를 추정해 `strings.Builder` 하나를 한 번만 키웁니다. 플레이스홀더(`?`, `$N`, `:N`)는 쓰는 동안 번호가 매겨지므로 조건을 두 번 훑을 필요가 없습니다.
3. **흔한 쿼리에 맞춘 슬라이스 크기.** 처음 호출된 `Where`, `WhereIn`, `WhereBetween`은 조건 네 개와 인자 네 개를 담을 공간을 확보합니다. 그래서 일반적인 쿼리는 이 슬라이스를 1, 2, 4개 단위로 여러 번 키우지 않고 한 번만 키웁니다.
4. **드문 절은 따로 둡니다.** ClickHouse 전용 절(`Final`, `Prewhere`, `Setting`)은 처음 사용할 때 할당되는 포인터 뒤에 있어서, 다른 데이터베이스의 빌더는 더 작은 할당 크기 클래스에 머뭅니다.
5. **Clone은 복사하지 않고 공유합니다.** 빌더는 슬라이스에 항상 추가(append)만 하므로, `Clone`은 원본의 기반 배열을 현재 길이로 잘라(clip) 공유합니다. 어느 쪽이든 처음 추가할 때 복사가 일어나므로 서로가 추가한 내용을 보지 않습니다.
6. **Build는 빌더를 변경하지 않습니다.** 여러 번 호출할 수 있고, 빌드가 끝난 뒤에는 여러 고루틴(goroutine)에서 호출할 수 있습니다.

`Build` 시점에 인용하는 데에는 비용이 있습니다. 완성된 빌더를 다시 렌더링하면 인용도 다시 하므로, 같은 빌더에 `Build`를 반복 호출하는 경우는 v1.5.0보다 다소 느립니다(README의 벤치마크 참고). 대신 흔한 경우인 쿼리를 한 번 만드는 경우에는 할당이 크게 줄어듭니다.

## 다른 빌더가 결과를 만드는 방식

- **Elasticsearch와 Qdrant**는 중첩된 맵을 만들어 `json.Marshal`을 호출하는 대신, JSON 본문을 정렬된 키 순서로 직접 씁니다. 문자열, 숫자, 불리언, `[]string`, `[]float32`, `[]interface{}`, `map[string]interface{}`는 `encoding/json`과 같은 이스케이프와 실수 형식으로 바로 인코딩합니다. 그 밖의 값 타입, NaN이나 무한대, 잘못된 UTF-8이 포함된 문자열은 `json.Marshal`에 넘기므로 출력과 에러가 그것과 같습니다. 퍼즈 테스트(`FuzzAppendJSON`)가 두 결과가 일치하는지 확인합니다.
- **MongoDB**는 필터 조건과 업데이트 연산자를 슬라이스에 보관하고, 반환할 맵은 `Build`에서 한 번만 만듭니다. `gqbd.D`는 `map[string]interface{}`와 `[]interface{}`를 리플렉션 없이 인코딩합니다.
- **Prometheus**는 레이블 매처를 구조체로 저장하고, 셀렉터를 버퍼 하나에 쓰면서 인용합니다. 숫자는 `fmt`가 아닌 `strconv`로 형식화합니다.
- **Redis**는 명령마다 인자 슬라이스를 한 번만 만들며, `TTL`과 `NX` 같은 옵션을 위한 공간을 미리 둡니다.
- **연결 문자열**은 미리 크기를 정한 버퍼 하나에 쓰고, 자격 증명은 그 자리에서 퍼센트 인코딩합니다. 호출당 할당은 한 번입니다.

## 빌더를 효율적으로 사용하기

- **`Clone`으로 기본 쿼리를 재사용합니다.** 공통 부분은 한 번만 만들고, 요청마다 다시 만들지 말고 복제합니다.

  ```go
  base := gqbd.BuildSelect(gqbd.PostgreSQL, "orders", "id", "total").Where("tenant_id = ?", tenantID)

  open := base.Clone().Where("status = ?", "open").Limit(20)
  query, args, err := open.Build()
  ```

- **바뀌지 않는 빌더는 `Build`를 다시 호출합니다.** 이미 만든 빌더를 렌더링하는 비용은 할당 두 번(쿼리 문자열과 인자 슬라이스)입니다.
- **`IN (?, ?, ...)`을 직접 쓰기보다 `WhereIn`을 사용합니다.** 플레이스홀더를 할당 한 번으로 쓰고, 인자를 쿼리 순서대로 유지합니다.
- **SQL을 문자열로 조립하지 말고 값을 넘깁니다.** 인젝션에 안전할 뿐 아니라, 바인딩된 인자는 `fmt` 형식화를 전혀 거치지 않습니다.

## 벤치마크

다음 명령으로 실행합니다.

```bash
go test -run='^$' -bench=. -benchmem ./...
```

`benchmark_test.go`는 SQL 빌더를, `benchmark_nosql_test.go`는 Redis, MongoDB, Elasticsearch, Prometheus, Qdrant, 연결 문자열을 다룹니다. 현재 릴리스의 결과는 [README](../../README.ko.md#벤치마크)에 있습니다.

변경 사항을 확인할 때는 [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)으로 실행 결과를 비교합니다.

```bash
go test -run='^$' -bench=. -benchmem -count=6 . > before.txt
# apply the change
go test -run='^$' -bench=. -benchmem -count=6 . > after.txt
benchstat before.txt after.txt
```

`-count=6` 이상을 사용하고, 그동안 CPU를 많이 쓰는 다른 작업은 실행하지 마세요. `benchstat`이 실제 변화와 잡음을 구분하려면 조용한 환경에서 얻은 샘플이 여러 개 필요합니다.
