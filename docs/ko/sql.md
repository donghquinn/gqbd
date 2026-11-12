# SQL 빌더

[English](../sql.md) | **한국어**

[← README로 돌아가기](../../README.ko.md)

`BuildSelect`, `BuildInsert`, `BuildUpdate`, `BuildDelete`는 모든 SQL 데이터베이스에서 같은 방식으로 동작하며, 플레이스홀더와 식별자 인용 방식만 다릅니다([PostgreSQL](./postgresql.md), [MySQL / MariaDB](./mysql.md), [SQLite](./sqlite.md), [ClickHouse](./clickhouse.md), [Oracle](./oracle.md) 참고). 아래 예제는 PostgreSQL 출력을 보여 줍니다.

`Build()`는 `(query, args, err)`를 반환하며 빌더를 변경하지 않으므로 여러 번 호출할 수 있습니다. 인자는 항상 쿼리에서 플레이스홀더가 나타나는 순서대로 반환됩니다.

데이터베이스 타입은 반드시 `gqbd` 상수 중 하나여야 합니다. 그 밖의 값(예를 들어 설정 파일에서 읽은 `gqbd.DBType("postgresql")`)을 사용하면 gqbd가 해당 데이터베이스의 식별자를 어떻게 이스케이프해야 할지 알 수 없으므로 `Build()`가 에러를 반환합니다.

## 플레이스홀더

모든 파라미터에는 `?`를 씁니다. gqbd는 대상 데이터베이스에 맞게 `$1`, `:1` 또는 `?`로 바꿔 줍니다. 따옴표로 감싼 문자열이나 이름(`'...'`, `"..."`, `` `...` ``) 안의 `?`는 그대로 둡니다. PostgreSQL의 jsonb 연산자처럼 리터럴 `?`가 필요하면 `??`를 씁니다.

```go
gqbd.BuildSelect(gqbd.PostgreSQL, "docs").Where("data ?? ?", "key").Where("tags ??| ?", pq.Array(tags))
// Query: SELECT * FROM "docs" WHERE data ? $1 AND tags ?| $2
```

## 기본 쿼리 재사용

체인 메서드는 호출 대상 빌더를 변경합니다. 공유하는 기본 쿼리를 확장하려면 먼저 `Clone()`합니다.

```go
base := gqbd.BuildSelect(gqbd.PostgreSQL, "orders").Where("tenant_id = ?", tenantID)

open, args, err := base.Clone().Where("status = ?", "open").Build()
// base는 변경되지 않으므로 다시 Clone할 수 있습니다
```

빌더는 구성하는 동안 동시에 사용하기에 안전하지 않습니다. 고루틴(goroutine)마다 공유 기본 쿼리를 Clone하거나, 구성을 마친 뒤에 공유합니다. `Build()` 자체는 빌더를 변경하지 않습니다.

## IN 절

```go
qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id", "name").
    WhereIn("status", []interface{}{"active", "pending", "verified"})

query, args, err := qb.Build()
// Query: SELECT "id", "name" FROM "users" WHERE "status" IN ($1, $2, $3)
// Args: [active pending verified]
```

## BETWEEN 절

```go
qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders", "id", "total").
    WhereBetween("created_at", "2023-01-01", "2023-12-31")

query, args, err := qb.Build()
// Query: SELECT "id", "total" FROM "orders" WHERE "created_at" BETWEEN $1 AND $2
// Args: [2023-01-01 2023-12-31]
```

## 집계 함수

```go
qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders", "customer_id").
    Aggregate("COUNT", "*").
    Aggregate("SUM", "total").
    GroupBy("customer_id").
    Having("COUNT(*) > ?", 5)

query, args, err := qb.Build()
// Query: SELECT "customer_id", COUNT(*), SUM("total") FROM "orders" GROUP BY "customer_id" HAVING COUNT(*) > $1
```

## 동적 조건

`AddWhereIfNotEmpty`는 값이 `nil`, 빈 문자열, nil이거나 비어 있는 `*string`이면 조건을 건너뜁니다.

```go
qb := gqbd.BuildSelect(gqbd.PostgreSQL, "posts").
    AddWhereIfNotEmpty("author = ?", author). // author == ""이면 건너뜀
    AddWhereIfNotEmpty("title ILIKE ?", title)
```

## 안전한 ORDER BY

정렬 컬럼이 사용자 입력에서 오는 경우 허용 목록을 전달합니다. 목록에 없는 컬럼은 `id`로 대체됩니다. `ASC`/`DESC` 이외의 방향은 `DESC`로 대체됩니다.

```go
allowed := map[string]bool{"name": true, "created_at": true}
qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users").OrderBy(sortParam, dirParam, allowed)
```

## 식별자 이스케이프

테이블과 컬럼 이름은 대상 데이터베이스에 맞게 인용되며, 이름 안에 포함된 인용 문자는 두 번 써서 인용 밖으로 벗어날 수 없게 합니다. `"table alias"`, `"table AS alias"`, `"t.column"` 형식을 지원하며, 별칭과 접두사는 반드시 일반 식별자(문자, 숫자, `_`)여야 합니다. 괄호가 포함된 이름(예: `COUNT(id)`)은 이스케이프 없이 그대로 전달되므로 절대 사용자 입력으로 만들면 안 됩니다.
