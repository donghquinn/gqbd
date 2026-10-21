# GQBD - High-Performance Go Query Builder
[![Test](https://github.com/donghquinn/gqbd/actions/workflows/test.yml/badge.svg)](https://github.com/donghquinn/gqbd/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/donghquinn/gqbd/branch/main/graph/badge.svg)](https://codecov.io/gh/donghquinn/gqbd)

**GQBD** is a high-performance, low-allocation query builder for Go: SQL for PostgreSQL, MySQL, MariaDB, SQLite and ClickHouse, plus command builders for Redis and MongoDB. It only builds queries; run them with the driver you already use.

- 🚀 **Low allocations**: single-pass rendering into a pre-sized buffer
- 🔒 **Injection safe**: parameter binding, identifier escaping, and `$`-operator protection for MongoDB
- 📦 **Zero dependencies**: standard library only, no database drivers
- 🔧 **Fluent API**: chainable methods; errors are collected and returned from `Build()`
- 🔌 **Connection strings**: DSNs and URIs for every supported database

## Installation

```bash
go get github.com/donghquinn/gqbd
```

## Quick Start

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

## Supported Databases

| Database | Constant | Builder | Placeholders | Identifiers | RETURNING | Docs |
|----------|----------|---------|--------------|-------------|-----------|------|
| PostgreSQL | `gqbd.PostgreSQL` | SQL | `$1, $2, ...` | `"identifier"` | ✅ | [docs/postgresql.md](docs/postgresql.md) |
| MySQL | `gqbd.Mysql` | SQL | `?` | `` `identifier` `` | ❌ | [docs/mysql.md](docs/mysql.md) |
| MariaDB | `gqbd.MariaDB` | SQL | `?` | `` `identifier` `` | ❌ | [docs/mysql.md](docs/mysql.md) |
| SQLite | `gqbd.SQLite` | SQL | `?` | `"identifier"` | ✅ (3.35.0+) | [docs/sqlite.md](docs/sqlite.md) |
| ClickHouse | `gqbd.ClickHouse` | SQL (+ `FINAL`, `PREWHERE`, `SETTINGS`) | `?` | `` `identifier` `` | ❌ | [docs/clickhouse.md](docs/clickhouse.md) |
| Redis | `gqbd.Redis` | `Redis*` commands | — | — | — | [docs/redis.md](docs/redis.md) |
| MongoDB | `gqbd.MongoDB` | `Mongo*` filters, updates, pipelines | — | — | — | [docs/mongodb.md](docs/mongodb.md) |

The SQL databases share one API: see [docs/sql.md](docs/sql.md) for IN, BETWEEN, aggregates, dynamic conditions and identifier escaping.

## Benchmarks

Run with `go test -bench . -benchmem` (Apple M-series, Go 1.27). See `benchmark_test.go` for the queries.

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| SELECT, PostgreSQL (join, 3 WHERE, ORDER BY, LIMIT/OFFSET) | 709 | 1744 | 36 |
| SELECT, MySQL (same query) | 681 | 1744 | 36 |
| INSERT, PostgreSQL (4 columns, RETURNING) | 315 | 680 | 10 |
| UPDATE, PostgreSQL (3 columns, 2 WHERE) | 331 | 768 | 13 |
| DELETE, MySQL | 113 | 504 | 7 |

## Project Structure

- `gqbd.go` - Public API: builder constructors and chainable clause methods
- `build.go` - Renders SELECT/INSERT/UPDATE/DELETE in a single pass
- `dialect.go` - Per-database differences (identifier quoting, placeholders, RETURNING, ClickHouse mutations) and the SQL writer
- `connection.go` - Connection string generation
- `redis.go` - Redis command builders
- `mongodb.go` - MongoDB find/update/delete and aggregation builders
- `bson.go` - `gqbd.D`, an ordered document with a minimal BSON encoder (so the driver isn't a dependency)

Database differences live in one dialect table, so adding a SQL database means adding one `dialect` entry.

## Contributing

Feel free to open issues or submit pull requests.
