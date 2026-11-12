# SQLite

[English](../sqlite.md) | **한국어**

[← README로 돌아가기](../../README.ko.md)

SQLite는 `?` 플레이스홀더와 큰따옴표 식별자를 사용하며, INSERT에서 `RETURNING`을 지원합니다(SQLite 3.35.0 이상).

모든 SQL 데이터베이스에 공통인 API(IN, BETWEEN, 집계, 동적 조건)는 [SQL 빌더](./sql.md)를 참고하세요.

## SELECT 쿼리

```go
qb := gqbd.BuildSelect(gqbd.SQLite, "users", "id", "name", "email").
    Where("age > ?", 18).
    Where("status = ?", "active").
    OrderBy("created_at", "DESC", nil).
    Limit(10).
    Offset(5)

query, args, err := qb.Build()
// Query: SELECT "id", "name", "email" FROM "users" WHERE age > ? AND status = ? ORDER BY "created_at" DESC LIMIT ? OFFSET ?
// Args: [18 active 10 5]
```

## RETURNING을 사용한 INSERT

```go
data := map[string]interface{}{
    "name":  "Alice",
    "email": "alice@example.com",
    "age":   28,
}

qb := gqbd.BuildInsert(gqbd.SQLite, "users").
    Values(data).
    Returning("id, name")

query, args, err := qb.Build()
// Query: INSERT INTO "users" ("age", "email", "name") VALUES (?, ?, ?) RETURNING id, name
// Args: [28 alice@example.com Alice]
```

## UPDATE 쿼리

```go
qb := gqbd.BuildUpdate(gqbd.SQLite, "users").
    Set(map[string]interface{}{
        "name":   "Updated Name",
        "status": "inactive",
    }).
    Where("id = ?", 1)

query, args, err := qb.Build()
// Query: UPDATE "users" SET "name" = ?, "status" = ? WHERE id = ?
// Args: [Updated Name inactive 1]
```

## 연결 문자열

```go
// SQLite 연결 문자열
sqliteConfig := gqbd.DBConfig{
    FilePath: "/path/to/database.db",
}
sqliteDSN := gqbd.BuildConnectionString(gqbd.SQLite, sqliteConfig)
// Result: "/path/to/database.db"

// SQLite 인메모리
sqliteMemConfig := gqbd.DBConfig{} // 빈 설정
sqliteMemDSN := gqbd.BuildConnectionString(gqbd.SQLite, sqliteMemConfig)
// Result: ":memory:"
```

## 드라이버

```go
import _ "github.com/mattn/go-sqlite3"

db, err := sql.Open("sqlite3", dsn)
```
