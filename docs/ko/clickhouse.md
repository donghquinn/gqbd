# ClickHouse

[English](../clickhouse.md) | **한국어**

[← README로 돌아가기](../../README.ko.md)

ClickHouse(`gqbd.ClickHouse`)는 `?` 플레이스홀더와 백틱 식별자를 사용합니다. ClickHouse는 인용된 이름 안의 `\`를 이스케이프 문자로 취급하므로, 식별자의 백틱뿐 아니라 백슬래시도 이스케이프합니다. `RETURNING`은 지원하지 않으며 `Returning()`은 무시됩니다.

모든 SQL 데이터베이스에 공통인 API(IN, BETWEEN, 집계, 동적 조건)는 [SQL 빌더](./sql.md)를 참고하세요.

## FINAL, PREWHERE, SETTINGS를 사용한 SELECT

`Final()`, `Prewhere()`, `Setting()`은 ClickHouse 전용이며, 다른 데이터베이스에서는 `Build()`가 에러를 반환합니다.

```go
qb := gqbd.BuildSelect(gqbd.ClickHouse, "events e", "e.id", "e.name", "u.country").
    Final().                                // 쿼리 시점에 ReplacingMergeTree 행을 병합
    LeftJoin("users u", "u.id = e.id").
    Prewhere("e.status = ?", "active").     // 다른 컬럼을 읽기 전에 필터링
    Where("e.v >= ?", 10).
    OrderBy("e.id", "ASC", nil).
    Limit(10).
    Setting("max_threads", 2)

query, args, err := qb.Build()
// Query: SELECT e.`id`, e.`name`, u.`country` FROM `events` e FINAL LEFT JOIN `users` u ON u.id = e.id PREWHERE e.status = ? WHERE e.v >= ? ORDER BY e.`id` ASC LIMIT ? SETTINGS max_threads = 2
// Args: [active 10 10]
```

`Setting(name, value)`는 숫자, bool(`1`/`0`으로 작성), 문자열을 받습니다. ClickHouse는 `SETTINGS`에서 플레이스홀더를 허용하지 않으므로 값은 이스케이프된 리터럴로 작성되며, 이름은 반드시 일반 식별자여야 합니다.

## INSERT

```go
qb := gqbd.BuildInsert(gqbd.ClickHouse, "events").
    Values(map[string]interface{}{"id": 1, "name": "signup"}).
    Setting("async_insert", true)

query, args, err := qb.Build()
// Query: INSERT INTO `events` (`id`, `name`) SETTINGS async_insert = 1 VALUES (?, ?)
// Args: [1 signup]
```

## UPDATE와 DELETE

ClickHouse의 업데이트는 뮤테이션이므로 `BuildUpdate`는 `ALTER TABLE ... UPDATE`를 생성합니다. `BuildDelete`는 경량 `DELETE FROM`(ClickHouse 23.3 이상)을 생성합니다. ClickHouse는 둘 다 `WHERE` 절을 요구하므로 `WHERE` 절이 없으면 `Build()`가 에러를 반환합니다. 모든 행에 적용하려면 `Where("1 = 1")`을 사용합니다.

```go
query, args, err := gqbd.BuildUpdate(gqbd.ClickHouse, "events").
    Set(map[string]interface{}{"name": "renamed"}).
    Where("id = ?", 1).
    Setting("mutations_sync", 2). // 뮤테이션이 끝날 때까지 대기
    Build()
// Query: ALTER TABLE `events` UPDATE `name` = ? WHERE id = ? SETTINGS mutations_sync = 2
// Args: [renamed 1]

query, args, err = gqbd.BuildDelete(gqbd.ClickHouse, "events").
    Where("created_at < ?", cutoff).
    Build()
// Query: DELETE FROM `events` WHERE created_at < ?
```

## 연결 문자열

```go
// ClickHouse DSN (clickhouse-go v2, 네이티브 프로토콜)
chConfig := gqbd.DBConfig{
    Host:     "localhost",
    Port:     9000,
    User:     "default",
    Password: "secret",
    DBName:   "analytics",
    SSLMode:  "disable", // 다른 값이면 secure=true가 추가됩니다(TLS 포트, 보통 9440 사용)
}
chDSN := gqbd.BuildConnectionString(gqbd.ClickHouse, chConfig)
// Result: "clickhouse://default:secret@localhost:9000/analytics"
```

## 드라이버

```go
import _ "github.com/ClickHouse/clickhouse-go/v2"

db, err := sql.Open("clickhouse", dsn)
rows, err := db.Query(query, args...)
```

clickhouse-go는 `?` 인자를 클라이언트 측에서 바인딩하므로 `LIMIT`와 `OFFSET`에서도 동작합니다.
