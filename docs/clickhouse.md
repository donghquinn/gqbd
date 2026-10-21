# ClickHouse

[← Back to README](../README.md)

ClickHouse (`gqbd.ClickHouse`) uses `?` placeholders and backticks for identifiers. Backslashes in identifiers are escaped as well as backticks, because ClickHouse treats `\` as an escape character inside quoted names. `RETURNING` is not supported and `Returning()` is ignored.

See [SQL builder](sql.md) for the API shared by all SQL databases (IN, BETWEEN, aggregates, dynamic conditions).

## SELECT with FINAL, PREWHERE and SETTINGS

`Final()`, `Prewhere()` and `Setting()` are ClickHouse-only; on other databases they return an error from `Build()`.

```go
qb := gqbd.BuildSelect(gqbd.ClickHouse, "events e", "e.id", "e.name", "u.country").
    Final().                                // merge ReplacingMergeTree rows at query time
    LeftJoin("users u", "u.id = e.id").
    Prewhere("e.status = ?", "active").     // filtered before reading other columns
    Where("e.v >= ?", 10).
    OrderBy("e.id", "ASC", nil).
    Limit(10).
    Setting("max_threads", 2)

query, args, err := qb.Build()
// Query: SELECT e.`id`, e.`name`, u.`country` FROM `events` e FINAL LEFT JOIN `users` u ON u.id = e.id PREWHERE e.status = ? WHERE e.v >= ? ORDER BY e.`id` ASC LIMIT ? SETTINGS max_threads = 2
// Args: [active 10 10]
```

`Setting(name, value)` accepts numbers, bools (written as `1`/`0`) and strings. ClickHouse doesn't accept placeholders in `SETTINGS`, so the value is written as an escaped literal and the name must be a plain identifier.

## INSERT

```go
qb := gqbd.BuildInsert(gqbd.ClickHouse, "events").
    Values(map[string]interface{}{"id": 1, "name": "signup"}).
    Setting("async_insert", true)

query, args, err := qb.Build()
// Query: INSERT INTO `events` (`id`, `name`) SETTINGS async_insert = 1 VALUES (?, ?)
// Args: [1 signup]
```

## UPDATE and DELETE

ClickHouse updates are mutations, so `BuildUpdate` produces `ALTER TABLE ... UPDATE`. `BuildDelete` produces a lightweight `DELETE FROM` (ClickHouse 23.3+). ClickHouse requires a `WHERE` clause for both, so `Build()` returns an error without one; use `Where("1 = 1")` to affect every row.

```go
query, args, err := gqbd.BuildUpdate(gqbd.ClickHouse, "events").
    Set(map[string]interface{}{"name": "renamed"}).
    Where("id = ?", 1).
    Setting("mutations_sync", 2). // wait for the mutation to finish
    Build()
// Query: ALTER TABLE `events` UPDATE `name` = ? WHERE id = ? SETTINGS mutations_sync = 2
// Args: [renamed 1]

query, args, err = gqbd.BuildDelete(gqbd.ClickHouse, "events").
    Where("created_at < ?", cutoff).
    Build()
// Query: DELETE FROM `events` WHERE created_at < ?
```

## Connection String

```go
// ClickHouse DSN (clickhouse-go v2, native protocol)
chConfig := gqbd.DBConfig{
    Host:     "localhost",
    Port:     9000,
    User:     "default",
    Password: "secret",
    DBName:   "analytics",
    SSLMode:  "disable", // anything else adds secure=true (use the TLS port, usually 9440)
}
chDSN := gqbd.BuildConnectionString(gqbd.ClickHouse, chConfig)
// Result: "clickhouse://default:secret@localhost:9000/analytics"
```

## Driver

```go
import _ "github.com/ClickHouse/clickhouse-go/v2"

db, err := sql.Open("clickhouse", dsn)
rows, err := db.Query(query, args...)
```

clickhouse-go binds `?` arguments on the client side, so they also work in `LIMIT` and `OFFSET`.
