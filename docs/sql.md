# SQL Builder

[← Back to README](../README.md)

`BuildSelect`, `BuildInsert`, `BuildUpdate` and `BuildDelete` work the same way for every SQL database; only the placeholder and identifier quoting style differ (see [PostgreSQL](postgresql.md), [MySQL / MariaDB](mysql.md), [SQLite](sqlite.md), [ClickHouse](clickhouse.md), [Oracle](oracle.md)). The examples below show PostgreSQL output.

`Build()` returns `(query, args, err)` and does not modify the builder, so it can be called more than once. Arguments are always returned in the order their placeholders appear in the query.

The database type must be one of the `gqbd` constants. Any other value (for example `gqbd.DBType("postgresql")` read from a config file) makes `Build()` return an error, because gqbd wouldn't know how to escape identifiers for it.

## Placeholders

Write `?` for every parameter; gqbd renders it as `$1`, `:1` or `?` for the target database. A `?` inside a quoted string or name (`'...'`, `"..."`, `` `...` ``) is left alone. Write `??` for a literal `?`, such as PostgreSQL's jsonb operators:

```go
gqbd.BuildSelect(gqbd.PostgreSQL, "docs").Where("data ?? ?", "key").Where("tags ??| ?", pq.Array(tags))
// Query: SELECT * FROM "docs" WHERE data ? $1 AND tags ?| $2
```

## Reusing a Base Query

Chained methods modify the builder they are called on. To extend a shared base query, `Clone()` it first:

```go
base := gqbd.BuildSelect(gqbd.PostgreSQL, "orders").Where("tenant_id = ?", tenantID)

open, args, err := base.Clone().Where("status = ?", "open").Build()
// base is unchanged and can be cloned again
```

A builder is not safe for concurrent use while it is being built. Clone a shared base in each goroutine, or finish building before sharing it: `Build()` itself doesn't modify the builder.

## IN Clause

```go
qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id", "name").
    WhereIn("status", []interface{}{"active", "pending", "verified"})

query, args, err := qb.Build()
// Query: SELECT "id", "name" FROM "users" WHERE "status" IN ($1, $2, $3)
// Args: [active pending verified]
```

## BETWEEN Clause

```go
qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders", "id", "total").
    WhereBetween("created_at", "2023-01-01", "2023-12-31")

query, args, err := qb.Build()
// Query: SELECT "id", "total" FROM "orders" WHERE "created_at" BETWEEN $1 AND $2
// Args: [2023-01-01 2023-12-31]
```

## Aggregate Functions

```go
qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders", "customer_id").
    Aggregate("COUNT", "*").
    Aggregate("SUM", "total").
    GroupBy("customer_id").
    Having("COUNT(*) > ?", 5)

query, args, err := qb.Build()
// Query: SELECT "customer_id", COUNT(*), SUM("total") FROM "orders" GROUP BY "customer_id" HAVING COUNT(*) > $1
```

## Dynamic Conditions

`AddWhereIfNotEmpty` skips the condition when the value is `nil`, an empty string, or a nil/empty `*string`:

```go
qb := gqbd.BuildSelect(gqbd.PostgreSQL, "posts").
    AddWhereIfNotEmpty("author = ?", author). // skipped when author == ""
    AddWhereIfNotEmpty("title ILIKE ?", title)
```

## Safe ORDER BY

Pass an allowlist when the sort column comes from user input; columns not in the list fall back to `id`. Directions other than `ASC`/`DESC` fall back to `DESC`.

```go
allowed := map[string]bool{"name": true, "created_at": true}
qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users").OrderBy(sortParam, dirParam, allowed)
```

## Identifier Escaping

Table and column names are quoted for the target database, and embedded quote characters are doubled so they can't break out of the quoting. The `"table alias"`, `"table AS alias"` and `"t.column"` forms are supported; aliases and prefixes must be plain identifiers (letters, digits, `_`). Names containing parentheses (e.g. `COUNT(id)`) are passed through unescaped, so never build them from user input.
