# PostgreSQL

[← Back to README](../README.md)

PostgreSQL uses `$N` placeholders and double quotes for identifiers, and supports `RETURNING` on INSERT. Write conditions with `?`; gqbd numbers them `$1, $2, ...` in query order when `Build()` is called.

See [SQL builder](sql.md) for the API shared by all SQL databases (IN, BETWEEN, aggregates, dynamic conditions).

## SELECT Query

```go
package main

import (
    "fmt"
    "github.com/donghquinn/gqbd"
)

func main() {
    // Basic SELECT with joins and conditions
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

## Dynamic WHERE Conditions

```go
func buildDynamicQuery(userName, title string, status string) (string, []interface{}, error) {
    qb := gqbd.BuildSelect(gqbd.PostgreSQL, "posts p", "p.id", "p.title", "u.username").
        LeftJoin("users u", "u.id = p.user_id")

    // Add conditions only if values are provided
    if userName != "" {
        qb = qb.Where("u.username ILIKE ?", "%"+userName+"%")
    }
    
    if title != "" {
        qb = qb.Where("p.title ILIKE ?", "%"+title+"%")
    }
    
    // Always add status filter
    qb = qb.Where("p.status = ?", status)

    return qb.Build()
}
```

## INSERT with RETURNING

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

## UPDATE Query

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

## Connection String

```go
// PostgreSQL connection string
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

Values containing spaces, quotes or backslashes (e.g. passwords) are quoted automatically.

## Driver

```go
import _ "github.com/lib/pq"              // Most common
// OR
import _ "github.com/jackc/pgx/v5/stdlib" // High performance
```

## Complete Example

```go
package main

import (
    "database/sql"
    "fmt"
    "log"
    
    _ "github.com/lib/pq"           // PostgreSQL driver
    _ "github.com/go-sql-driver/mysql" // MySQL driver  
    _ "github.com/mattn/go-sqlite3"    // SQLite driver
    "github.com/donghquinn/gqbd"
)

func main() {
    // Configure database connection
    config := gqbd.DBConfig{
        Host:     "localhost",
        Port:     5432,
        User:     "postgres",
        Password: "password",
        DBName:   "myapp",
        SSLMode:  "disable",
    }
    
    // Generate connection string
    dsn := gqbd.BuildConnectionString(gqbd.PostgreSQL, config)
    
    // Open database connection
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Build and execute query
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
    
    // Process results...
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
