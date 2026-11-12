# GQBD — Go Query Builder

**English** | [한국어](./README.ko.md)

[![Test](https://github.com/donghquinn/gqbd/actions/workflows/test.yml/badge.svg)](https://github.com/donghquinn/gqbd/actions/workflows/test.yml)
[![Security](https://github.com/donghquinn/gqbd/actions/workflows/security.yml/badge.svg)](https://github.com/donghquinn/gqbd/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/donghquinn/gqbd/branch/main/graph/badge.svg)](https://codecov.io/gh/donghquinn/gqbd)
[![Go Reference](https://pkg.go.dev/badge/github.com/donghquinn/gqbd.svg)](https://pkg.go.dev/github.com/donghquinn/gqbd)
[![Go Version](https://img.shields.io/github/go-mod/go-version/donghquinn/gqbd)](https://go.dev/)
[![License](https://img.shields.io/github/license/donghquinn/gqbd)](./LICENSE)

GQBD builds queries for Go without connecting to a database: parameterized SQL for PostgreSQL, MySQL, MariaDB, SQLite, ClickHouse and Oracle, plus commands and request bodies for Redis, MongoDB, Elasticsearch, Prometheus and Qdrant. Run the results with the driver you already use.

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

## Features

- **Low allocations**: identifiers are parsed once and written straight into one pre-sized buffer; a typical SELECT costs 9 allocations. See [Performance](docs/performance.md).
- **Injection safe**: values are always bound as arguments, identifiers are quoted per dialect, MongoDB `$` operators are rejected in field names, Elasticsearch and Qdrant bodies are structure-safe JSON, and PromQL label values are escaped.
- **No dependencies**: standard library only, no database drivers.
- **Fluent API**: chainable methods; the first error is kept and returned from `Build()`, so a chain needs one error check.
- **Reusable**: `Build()` doesn't modify the builder, and `Clone()` extends a shared base query without changing it.
- **Connection strings**: DSNs and URIs for every supported database with `BuildConnectionString`.

## Installation

```bash
go get github.com/donghquinn/gqbd
```

Requires Go 1.26.10 or newer.

## Supported databases

| Database | Constant | Builder | Placeholders | Identifiers | RETURNING | Docs |
|----------|----------|---------|--------------|-------------|-----------|------|
| PostgreSQL | `gqbd.PostgreSQL` | SQL | `$1, $2, ...` | `"identifier"` | ✅ | [docs/postgresql.md](docs/postgresql.md) |
| MySQL | `gqbd.Mysql` | SQL | `?` | `` `identifier` `` | ❌ | [docs/mysql.md](docs/mysql.md) |
| MariaDB | `gqbd.MariaDB` | SQL | `?` | `` `identifier` `` | ❌ | [docs/mysql.md](docs/mysql.md) |
| SQLite | `gqbd.SQLite` | SQL | `?` | `"identifier"` | ✅ (3.35.0+) | [docs/sqlite.md](docs/sqlite.md) |
| ClickHouse | `gqbd.ClickHouse` | SQL (+ `FINAL`, `PREWHERE`, `SETTINGS`) | `?` | `` `identifier` `` | ❌ | [docs/clickhouse.md](docs/clickhouse.md) |
| Oracle | `gqbd.Oracle` | SQL (`OFFSET ... FETCH NEXT`) | `:1, :2, ...` | `"identifier"` (case-sensitive) | ❌ | [docs/oracle.md](docs/oracle.md) |
| Redis | `gqbd.Redis` | `Redis*` commands | — | — | — | [docs/redis.md](docs/redis.md) |
| MongoDB | `gqbd.MongoDB` | `Mongo*` filters, updates, pipelines | — | — | — | [docs/mongodb.md](docs/mongodb.md) |
| Elasticsearch | `gqbd.Elasticsearch` | `ES*` search, count, delete-by-query bodies | — | — | — | [docs/elasticsearch.md](docs/elasticsearch.md) |
| Prometheus | `gqbd.Prometheus` | `PromSelector` PromQL expressions | — | — | — | [docs/prometheus.md](docs/prometheus.md) |
| Qdrant | `gqbd.Qdrant` | `Qdrant*` REST query, count, delete bodies | — | — | — | [docs/qdrant.md](docs/qdrant.md) |

The SQL databases share one API: see [docs/sql.md](docs/sql.md) for placeholders, `Clone`, IN, BETWEEN, joins, aggregates, dynamic conditions and identifier escaping.

## A taste of each builder

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

Each database page under [docs/](docs/) has complete examples.

## Benchmarks

Apple M5, Go 1.27, `go test -bench . -benchmem -count=6`; medians. "Before" is v1.5.0, measured with the same benchmark code.

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| SELECT, PostgreSQL (join, 3 WHERE, IN, ORDER BY, LIMIT/OFFSET) | 768 → **542** | 1744 → **1280** | 36 → **9** |
| SELECT, MySQL (same query) | 737 → **505** | 1744 → **1280** | 36 → **9** |
| SELECT, Oracle (OFFSET … FETCH NEXT) | 404 → **296** | 960 → **1008** | 21 → **7** |
| INSERT, PostgreSQL (4 columns, RETURNING) | 321 → **252** | 680 → **624** | 10 → **4** |
| UPDATE, PostgreSQL (3 columns, 2 WHERE) | 347 → **274** | 768 → **752** | 13 → **6** |
| DELETE, MySQL | 119 → **105** | 504 → **576** | 7 → **5** |
| `Build()` of a finished builder | 127 → **156** | 544 → **256** | 3 → **2** |
| `Clone()` + WHERE + `Build()` | 146 → **162** | 288 → **640** | 7 → **5** |
| Redis SET with TTL and NX | 95 → **62** | 392 → **248** | 7 → **5** |
| MongoDB find (filter, sort, projection) | 1255 → **703** | 5448 → **3376** | 45 → **31** |
| MongoDB update | 970 → **484** | 4128 → **2400** | 26 → **15** |
| MongoDB aggregate | 898 → **724** | 4584 → **3952** | 41 → **34** |
| `gqbd.D` MarshalBSON | 285 → **189** | 600 → **432** | 10 → **2** |
| Elasticsearch search body | 3655 → **1199** | 9346 → **4440** | 93 → **32** |
| Prometheus selector | 269 → **161** | 528 → **496** | 14 → **4** |
| Qdrant query body | 2941 → **1153** | 5774 → **3040** | 77 → **47** |
| Connection string, PostgreSQL | 241 → **106** | 416 → **112** | 11 → **1** |

Rendering a finished builder is slower than before because identifiers are now quoted at `Build()` time rather than when each method is called. That trade cuts allocations for the common build-once case. See [docs/performance.md](docs/performance.md) for how the builders work and how to benchmark your own changes.

## Project structure

| File | Contents |
|------|----------|
| `gqbd.go` | Public SQL API: builder constructors and chainable clause methods |
| `build.go` | Renders SELECT/INSERT/UPDATE/DELETE in a single pass |
| `dialect.go` | Per-database differences (quoting, placeholders, RETURNING, ClickHouse mutations, Oracle pagination), identifier parsing and the SQL writer |
| `connection.go` | Connection string generation |
| `redis.go` | Redis command builders |
| `mongodb.go` | MongoDB find/update/delete and aggregation builders |
| `bson.go` | `gqbd.D`, an ordered document with a minimal BSON encoder (so the driver isn't a dependency) |
| `elasticsearch.go` | Elasticsearch query DSL builders and the JSON body encoder |
| `prometheus.go` | PromQL builder: selectors, functions, aggregations and operators |
| `qdrant.go` | Qdrant REST request builders for vector search, count and delete |

## Contributing

Issues and pull requests are welcome. [CONTRIBUTING.md](CONTRIBUTING.md) covers the local checks, benchmarking hot-path changes, the CI workflows and the release process.

## License

[MIT](./LICENSE)
