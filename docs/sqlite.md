# SQLite

[← Back to README](../README.md)

SQLite uses `?` placeholders and double quotes for identifiers, and supports `RETURNING` on INSERT (SQLite 3.35.0+).

See [SQL builder](sql.md) for the API shared by all SQL databases (IN, BETWEEN, aggregates, dynamic conditions).

## SELECT Query

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

## INSERT with RETURNING

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

## UPDATE Query

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

## Connection String

```go
// SQLite connection strings
sqliteConfig := gqbd.DBConfig{
    FilePath: "/path/to/database.db",
}
sqliteDSN := gqbd.BuildConnectionString(gqbd.SQLite, sqliteConfig)
// Result: "/path/to/database.db"

// SQLite in-memory
sqliteMemConfig := gqbd.DBConfig{} // Empty config
sqliteMemDSN := gqbd.BuildConnectionString(gqbd.SQLite, sqliteMemConfig)
// Result: ":memory:"
```

## Driver

```go
import _ "github.com/mattn/go-sqlite3"

db, err := sql.Open("sqlite3", dsn)
```
