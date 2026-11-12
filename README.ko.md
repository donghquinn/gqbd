# GQBD — Go 쿼리 빌더

[English](./README.md) | **한국어**

[![Test](https://github.com/donghquinn/gqbd/actions/workflows/test.yml/badge.svg)](https://github.com/donghquinn/gqbd/actions/workflows/test.yml)
[![Security](https://github.com/donghquinn/gqbd/actions/workflows/security.yml/badge.svg)](https://github.com/donghquinn/gqbd/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/donghquinn/gqbd/branch/main/graph/badge.svg)](https://codecov.io/gh/donghquinn/gqbd)
[![Go Reference](https://pkg.go.dev/badge/github.com/donghquinn/gqbd.svg)](https://pkg.go.dev/github.com/donghquinn/gqbd)
[![Go Version](https://img.shields.io/github/go-mod/go-version/donghquinn/gqbd)](https://go.dev/)
[![License](https://img.shields.io/github/license/donghquinn/gqbd)](./LICENSE)

GQBD는 데이터베이스에 연결하지 않고 Go에서 쿼리를 만듭니다. PostgreSQL, MySQL, MariaDB, SQLite, ClickHouse, Oracle용 파라미터 바인딩 SQL과, Redis, MongoDB, Elasticsearch, Prometheus, Qdrant용 명령 및 요청 본문을 만들어 줍니다. 결과는 이미 사용 중인 드라이버로 실행합니다.

```go
query, args, err := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id", "name").
    Where("age > ?", 18).
    OrderBy("created_at", "DESC", nil).
    Limit(10).
    Build()
// Query: SELECT "id", "name" FROM "users" WHERE age > $1 ORDER BY "created_at" DESC LIMIT $2
// Args: [18 10]

rows, err := db.Query(query, args...)
```

## 특징

- **적은 할당**: 식별자는 한 번 파싱되어 미리 크기를 정한 버퍼 하나에 바로 쓰입니다. 일반적인 SELECT는 할당 9번이면 됩니다. [성능](docs/ko/performance.md) 문서를 참고하세요.
- **인젝션 안전**: 값은 항상 인자로 바인딩되고, 식별자는 데이터베이스별 방식으로 인용됩니다. MongoDB 필드 이름의 `$` 연산자는 거부되고, Elasticsearch와 Qdrant 본문은 구조가 깨지지 않는 JSON으로 만들어지며, PromQL 레이블 값은 이스케이프됩니다.
- **의존성 없음**: 표준 라이브러리만 사용하며 데이터베이스 드라이버도 필요 없습니다.
- **플루언트 API**: 메서드를 체이닝할 수 있습니다. 첫 번째 에러가 보관되어 `Build()`에서 반환되므로 체인마다 에러 검사는 한 번이면 됩니다.
- **재사용 가능**: `Build()`는 빌더를 변경하지 않으며, `Clone()`으로 공통 기본 쿼리를 바꾸지 않고 확장할 수 있습니다.
- **연결 문자열**: `BuildConnectionString`으로 지원하는 모든 데이터베이스의 DSN과 URI를 만들 수 있습니다.

## 설치

```bash
go get github.com/donghquinn/gqbd
```

Go 1.26.10 이상이 필요합니다.

## 지원 데이터베이스

| 데이터베이스 | 상수 | 빌더 | 플레이스홀더 | 식별자 | RETURNING | 문서 |
|--------------|------|------|--------------|--------|-----------|------|
| PostgreSQL | `gqbd.PostgreSQL` | SQL | `$1, $2, ...` | `"identifier"` | ✅ | [docs/ko/postgresql.md](docs/ko/postgresql.md) |
| MySQL | `gqbd.Mysql` | SQL | `?` | `` `identifier` `` | ❌ | [docs/ko/mysql.md](docs/ko/mysql.md) |
| MariaDB | `gqbd.MariaDB` | SQL | `?` | `` `identifier` `` | ❌ | [docs/ko/mysql.md](docs/ko/mysql.md) |
| SQLite | `gqbd.SQLite` | SQL | `?` | `"identifier"` | ✅ (3.35.0+) | [docs/ko/sqlite.md](docs/ko/sqlite.md) |
| ClickHouse | `gqbd.ClickHouse` | SQL (+ `FINAL`, `PREWHERE`, `SETTINGS`) | `?` | `` `identifier` `` | ❌ | [docs/ko/clickhouse.md](docs/ko/clickhouse.md) |
| Oracle | `gqbd.Oracle` | SQL (`OFFSET ... FETCH NEXT`) | `:1, :2, ...` | `"identifier"` (대소문자 구분) | ❌ | [docs/ko/oracle.md](docs/ko/oracle.md) |
| Redis | `gqbd.Redis` | `Redis*` 명령 | — | — | — | [docs/ko/redis.md](docs/ko/redis.md) |
| MongoDB | `gqbd.MongoDB` | `Mongo*` 필터, 업데이트, 파이프라인 | — | — | — | [docs/ko/mongodb.md](docs/ko/mongodb.md) |
| Elasticsearch | `gqbd.Elasticsearch` | `ES*` search, count, delete-by-query 본문 | — | — | — | [docs/ko/elasticsearch.md](docs/ko/elasticsearch.md) |
| Prometheus | `gqbd.Prometheus` | `PromSelector` PromQL 표현식 | — | — | — | [docs/ko/prometheus.md](docs/ko/prometheus.md) |
| Qdrant | `gqbd.Qdrant` | `Qdrant*` REST query, count, delete 본문 | — | — | — | [docs/ko/qdrant.md](docs/ko/qdrant.md) |

SQL 데이터베이스는 하나의 API를 공유합니다. 플레이스홀더, `Clone`, IN, BETWEEN, 조인, 집계, 동적 조건, 식별자 이스케이프는 [docs/ko/sql.md](docs/ko/sql.md)를 참고하세요.

## 빌더 맛보기

```go
// SQL: INSERT with RETURNING
q, args, err := gqbd.BuildInsert(gqbd.PostgreSQL, "users").
    Values(map[string]interface{}{"name": "Ann", "email": "ann@example.com"}).
    Returning("id").
    Build()
// Query: INSERT INTO "users" ("email", "name") VALUES ($1, $2) RETURNING id

// Redis
cmd, args, err := gqbd.RedisSet("session:42", token).TTL(30 * time.Minute).NX().Build()
// cmd: "SET", args: [session:42 <token> EX 1800 NX]

// MongoDB
mq, err := gqbd.MongoFind("users").Eq("status", "active").Sort("created_at", -1).Limit(10).Build()

// Connection strings
dsn := gqbd.BuildConnectionString(gqbd.PostgreSQL, gqbd.DBConfig{
    Host: "localhost", Port: 5432, User: "app", Password: "secret", DBName: "app",
})
```

[docs/ko/](docs/ko/)의 데이터베이스별 문서에 전체 예제가 있습니다.

## 벤치마크

Apple M5, Go 1.27, `go test -bench . -benchmem -count=6`의 중앙값입니다. "이전"은 v1.5.0이며, 같은 벤치마크 코드로 측정했습니다.

| 벤치마크 | ns/op | B/op | allocs/op |
|----------|------:|-----:|----------:|
| SELECT, PostgreSQL (조인, WHERE 3개, IN, ORDER BY, LIMIT/OFFSET) | 768 → **542** | 1744 → **1280** | 36 → **9** |
| SELECT, MySQL (같은 쿼리) | 737 → **505** | 1744 → **1280** | 36 → **9** |
| SELECT, Oracle (OFFSET … FETCH NEXT) | 404 → **296** | 960 → **1008** | 21 → **7** |
| INSERT, PostgreSQL (컬럼 4개, RETURNING) | 321 → **252** | 680 → **624** | 10 → **4** |
| UPDATE, PostgreSQL (컬럼 3개, WHERE 2개) | 347 → **274** | 768 → **752** | 13 → **6** |
| DELETE, MySQL | 119 → **105** | 504 → **576** | 7 → **5** |
| 완성된 빌더의 `Build()` | 127 → **156** | 544 → **256** | 3 → **2** |
| `Clone()` + WHERE + `Build()` | 146 → **162** | 288 → **640** | 7 → **5** |
| Redis SET (TTL, NX 포함) | 95 → **62** | 392 → **248** | 7 → **5** |
| MongoDB find (필터, 정렬, 프로젝션) | 1255 → **703** | 5448 → **3376** | 45 → **31** |
| MongoDB update | 970 → **484** | 4128 → **2400** | 26 → **15** |
| MongoDB aggregate | 898 → **724** | 4584 → **3952** | 41 → **34** |
| `gqbd.D` MarshalBSON | 285 → **189** | 600 → **432** | 10 → **2** |
| Elasticsearch search 본문 | 3655 → **1199** | 9346 → **4440** | 93 → **32** |
| Prometheus 셀렉터 | 269 → **161** | 528 → **496** | 14 → **4** |
| Qdrant query 본문 | 2941 → **1153** | 5774 → **3040** | 77 → **47** |
| 연결 문자열, PostgreSQL | 241 → **106** | 416 → **112** | 11 → **1** |

이제 식별자를 각 메서드 호출 시점이 아니라 `Build()` 시점에 인용하므로, 완성된 빌더를 다시 렌더링하는 경우는 이전보다 느립니다. 이 맞교환으로 한 번 만들고 끝나는 일반적인 경우의 할당이 줄었습니다. 빌더의 동작 방식과 변경 사항을 벤치마크하는 방법은 [docs/ko/performance.md](docs/ko/performance.md)를 참고하세요.

## 프로젝트 구조

| 파일 | 내용 |
|------|------|
| `gqbd.go` | 공개 SQL API: 빌더 생성자와 체이닝 절 메서드 |
| `build.go` | SELECT/INSERT/UPDATE/DELETE를 한 번에 렌더링 |
| `dialect.go` | 데이터베이스별 차이(인용, 플레이스홀더, RETURNING, ClickHouse 뮤테이션, Oracle 페이지네이션), 식별자 파싱, SQL 라이터 |
| `connection.go` | 연결 문자열 생성 |
| `redis.go` | Redis 명령 빌더 |
| `mongodb.go` | MongoDB find/update/delete 및 집계 빌더 |
| `bson.go` | `gqbd.D`: 순서가 있는 문서와 최소한의 BSON 인코더(드라이버를 의존성으로 두지 않기 위함) |
| `elasticsearch.go` | Elasticsearch 쿼리 DSL 빌더와 JSON 본문 인코더 |
| `prometheus.go` | PromQL 빌더: 셀렉터, 함수, 집계, 연산자 |
| `qdrant.go` | 벡터 검색, count, delete용 Qdrant REST 요청 빌더 |

## 기여

이슈와 풀 리퀘스트를 환영합니다. 로컬 검사, 핫 패스 변경 시 벤치마크, CI 워크플로, 릴리스 절차는 [CONTRIBUTING.ko.md](CONTRIBUTING.ko.md)에 정리되어 있습니다.

## 라이선스

[MIT](./LICENSE)
