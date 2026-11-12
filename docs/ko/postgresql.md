# PostgreSQL

[English](../postgresql.md) | **한국어**

[← README로 돌아가기](../../README.ko.md)

PostgreSQL은 `$N` 플레이스홀더와 큰따옴표 식별자를 사용하며, INSERT에서 `RETURNING`을 지원합니다. 조건은 `?`로 작성하면 `Build()`를 호출할 때 gqbd가 쿼리 순서대로 `$1, $2, ...` 번호를 매깁니다.

모든 SQL 데이터베이스에 공통인 API(IN, BETWEEN, 집계, 동적 조건)는 [SQL 빌더](./sql.md)를 참고하세요.

## SELECT 쿼리

```go
package main

import (
    "fmt"
    "github.com/donghquinn/gqbd"
)

func main() {
    // 조인과 조건을 사용한 기본 SELECT
    qb := gqbd.BuildSelect(gqbd.PostgreSQL, "example_table e", "e.id", "e.name", "u.username").
        LeftJoin("user_table u", "u.user_id = e.id").
        Where("e.status = ?", "active").
        OrderBy("e.created_at", "DESC", nil).
        Limit(10).
        Offset(0)

    query, args, err := qb.Build()
    if err != nil {
        panic(err)
    }

    fmt.Printf("Query: %s\n", query)
    fmt.Printf("Args: %v\n", args)
    
    // Output:
    // Query: SELECT e."id", e."name", u."username" FROM "example_table" e LEFT JOIN "user_table" u ON u.user_id = e.id WHERE e.status = $1 ORDER BY e."created_at" DESC LIMIT $2
    // Args: [active 10]   (OFFSET 0 is omitted)
}
```

## 동적 WHERE 조건

```go
func buildDynamicQuery(userName, title string, status string) (string, []interface{}, error) {
    qb := gqbd.BuildSelect(gqbd.PostgreSQL, "posts p", "p.id", "p.title", "u.username").
        LeftJoin("users u", "u.id = p.user_id")

    // 값이 있을 때만 조건 추가
    if userName != "" {
        qb = qb.Where("u.username ILIKE ?", "%"+userName+"%")
    }
    
    if title != "" {
        qb = qb.Where("p.title ILIKE ?", "%"+title+"%")
    }
    
    // status 필터는 항상 추가
    qb = qb.Where("p.status = ?", status)

    return qb.Build()
}
```

## RETURNING을 사용한 INSERT

```go
data := map[string]interface{}{
    "name":    "John Doe",
    "email":   "john@example.com",
    "active":  true,
}

qb := gqbd.BuildInsert(gqbd.PostgreSQL, "users").
    Values(data).
    Returning("id, created_at")

query, args, err := qb.Build()
// Query: INSERT INTO "users" ("active", "email", "name") VALUES ($1, $2, $3) RETURNING id, created_at
// Args: [true john@example.com John Doe]   (columns are sorted for deterministic output)
```

## UPDATE 쿼리

```go
data := map[string]interface{}{
    "name":       "Jane Doe",
    "updated_at": "NOW()",
}

qb := gqbd.BuildUpdate(gqbd.PostgreSQL, "users").
    Set(data).
    Where("id = ?", 123)

query, args, err := qb.Build()
// Query: UPDATE "users" SET "name" = $1, "updated_at" = $2 WHERE id = $3
// Args: [Jane Doe NOW() 123]
```

## 연결 문자열

```go
// PostgreSQL 연결 문자열
pgConfig := gqbd.DBConfig{
    Host:     "localhost",
    Port:     5432,
    User:     "postgres",
    Password: "postgres",
    DBName:   "myapp",
    SSLMode:  "disable",
}
pgDSN := gqbd.BuildConnectionString(gqbd.PostgreSQL, pgConfig)
// Result: "host=localhost port=5432 user=postgres password=postgres dbname=myapp sslmode=disable"
```

공백, 따옴표, 백슬래시가 포함된 값(예: 비밀번호)은 자동으로 인용됩니다.

## 드라이버

```go
import _ "github.com/lib/pq"              // 가장 일반적
// 또는
import _ "github.com/jackc/pgx/v5/stdlib" // 고성능
```

## 전체 예제

```go
package main

import (
    "database/sql"
    "fmt"
    "log"
    
    _ "github.com/lib/pq"           // PostgreSQL 드라이버
    _ "github.com/go-sql-driver/mysql" // MySQL 드라이버  
    _ "github.com/mattn/go-sqlite3"    // SQLite 드라이버
    "github.com/donghquinn/gqbd"
)

func main() {
    // 데이터베이스 연결 설정
    config := gqbd.DBConfig{
        Host:     "localhost",
        Port:     5432,
        User:     "postgres",
        Password: "password",
        DBName:   "myapp",
        SSLMode:  "disable",
    }
    
    // 연결 문자열 생성
    dsn := gqbd.BuildConnectionString(gqbd.PostgreSQL, config)
    
    // 데이터베이스 연결 열기
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // 쿼리 생성 및 실행
    qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id", "name", "email").
        Where("status = ?", "active").
        OrderBy("name", "ASC", nil).
        Limit(10)
    
    query, args, err := qb.Build()
    if err != nil {
        log.Fatal(err)
    }
    
    rows, err := db.Query(query, args...)
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()
    
    // 결과 처리...
    for rows.Next() {
        var id int
        var name, email string
        if err := rows.Scan(&id, &name, &email); err != nil {
            log.Fatal(err)
        }
        fmt.Printf("ID: %d, Name: %s, Email: %s\n", id, name, email)
    }
}
```
