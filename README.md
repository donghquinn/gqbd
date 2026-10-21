# GQBD - High-Performance Go Query Builder
[![Test](https://github.com/donghquinn/gqbd/actions/workflows/test.yml/badge.svg)](https://github.com/donghquinn/gqbd/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/donghquinn/gqbd/branch/main/graph/badge.svg)](https://codecov.io/gh/donghquinn/gqbd)

## Introduction
**GQBD** is a high-performance, low-allocation SQL query builder (plus Redis and MongoDB builders) for Go, inspired by gdct design principles.

✨ **Key Features:**
- 🚀 **Low allocations**: single-pass rendering into a pre-sized buffer
- 🔒 **SQL injection safe** with automatic parameter binding and identifier escaping  
- 🎯 **Multi-database support**: PostgreSQL, MySQL, MariaDB, SQLite, Redis, and MongoDB
- ⚡ **Performance optimized** - designed for high-throughput applications
- 📦 **Zero dependencies** - pure Go implementation
- 🔧 **Fluent API** - chainable method design for readable code

## Installation

```bash
go get github.com/donghquinn/gqbd
```

## Performance & Features

🏆 **Performance Benefits:**
- Low allocations in query building (see [Benchmarks](#benchmarks))
- Optimized string building and parameter handling
- Minimal overhead compared to other query builders
- Built for high-throughput, low-latency applications

📋 **Full Feature Set:**
- ✅ **SELECT**: Complex queries with joins, conditions, grouping, ordering
- ✅ **INSERT**: Bulk inserts with optional RETURNING clause (PostgreSQL)
- ✅ **UPDATE**: Conditional updates with proper parameter binding
- ✅ **DELETE**: Safe deletion with WHERE conditions
- ✅ **Advanced Clauses**: IN, BETWEEN, aggregate functions, DISTINCT
- ✅ **Security**: Automatic SQL injection prevention
- ✅ **Cross-Database**: PostgreSQL ($N), MySQL/MariaDB/SQLite (?) placeholder formats
- ✅ **Connection Strings**: Built-in database connection string generation (including `redis://` URLs)
- ✅ **Redis**: Command builders for strings, hashes, lists, sets, sorted sets, TTLs and SCAN
- ✅ **MongoDB**: Find/update/delete builders and aggregation pipelines, without importing the driver

## Quick Start

GQBD follows the same patterns as gdct but without database connections. Just build queries and use them with your preferred driver.

```go
// Simple example - matches gdct API pattern
query, args, err := gqbd.BuildSelect(gqbd.PostgreSQL, "users").
    Where("age > ?", 18).
    OrderBy("created_at", "DESC", nil).
    Limit(10).
    Build()

// Use with any database driver
rows, err := db.Query(query, args...)
```

## Detailed Examples

### PostgreSQL Examples

PostgreSQL uses `$N` parameter placeholders and double quotes for identifiers.

#### SELECT Query

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

#### Dynamic WHERE Conditions

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

#### INSERT with RETURNING

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

#### UPDATE Query

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

### MySQL/MariaDB Examples

MySQL/MariaDB uses `?` parameter placeholders and backticks for identifiers.

#### SELECT Query

```go
qb := gqbd.BuildSelect(gqbd.MariaDB, "products", "id", "name", "price").
    Where("category_id = ?", 10).
    Where("price BETWEEN ? AND ?", 100, 500).
    OrderBy("price", "ASC", nil).
    Limit(20)

query, args, err := qb.Build()
// Query: SELECT `id`, `name`, `price` FROM `products` WHERE category_id = ? AND price BETWEEN ? AND ? ORDER BY `price` ASC LIMIT ?
// Args: [10 100 500 20]
```

#### INSERT Query

```go
data := map[string]interface{}{
    "name":        "New Product",
    "price":       99.99,
    "category_id": 5,
}

qb := gqbd.BuildInsert(gqbd.MariaDB, "products").
    Values(data)

query, args, err := qb.Build()
// Query: INSERT INTO `products` (`category_id`, `name`, `price`) VALUES (?, ?, ?)
// Args: [5 New Product 99.99]
```

### SQLite Examples

SQLite uses `?` parameter placeholders and double quotes for identifiers, with support for RETURNING clause.

#### SELECT Query

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

#### INSERT with RETURNING

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

#### UPDATE Query

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

### Redis Examples

Redis is not a SQL database, so it has its own command builders instead of `BuildSelect`/`BuildInsert`/... (those return an error for `gqbd.Redis`). Like the SQL builders, they have no client dependency: `Build()` returns the command name and its arguments for any Redis client.

```go
// SET with TTL and NX
cmd, args, err := gqbd.RedisSet(gqbd.RedisKey("session", 42), token).
    TTL(30 * time.Minute).
    NX().
    Build()
// cmd:  "SET"
// args: [session:42 <token> EX 1800 NX]

conn.Do(cmd, args...) // redigo

// go-redis takes the name and args as one slice
all, err := gqbd.RedisHSet("user:1", map[string]interface{}{
    "name": "Kim",
    "age":  30,
}).BuildArgs()
// all: [HSET user:1 age 30 name Kim]   (fields are sorted for deterministic output)
rdb.Do(ctx, all...)
```

Available builders:

| Builder | Command |
|---------|---------|
| `RedisGet(key)` / `RedisSet(key, value)` | `GET` / `SET` (chain `.TTL(d)`, `.NX()`, `.XX()`) |
| `RedisDel(keys...)` / `RedisExists(keys...)` | `DEL` / `EXISTS` |
| `RedisExpire(key, d)` | `EXPIRE` (or `PEXPIRE` for sub-second durations) |
| `RedisIncr(key)` / `RedisIncrBy(key, n)` | `INCR` / `INCRBY` |
| `RedisHSet(key, map)` / `RedisHGet` / `RedisHMGet` / `RedisHGetAll` / `RedisHDel` | Hash commands |
| `RedisLPush` / `RedisRPush` / `RedisLRange` | List commands |
| `RedisSAdd` / `RedisSMembers` | Set commands |
| `RedisZAdd(key, map[string]float64)` / `RedisZRange` | Sorted set commands |
| `RedisScan(cursor)` | `SCAN` (chain `.Match(pattern)`, `.Count(n)`) |
| `NewRedisCommand(name, args...)` | Any other command (chain `.Arg(...)`) |

`RedisKey("user", 42, "profile")` joins key parts with `:` → `user:42:profile`.

### MongoDB Examples

MongoDB gets its own builders (the SQL builders return an error for `gqbd.MongoDB`). gqbd does **not** import the MongoDB driver: filters, updates and pipeline stages are plain `map[string]interface{}` values, which the driver encodes itself. Sorts need key order, so they use `gqbd.D`, which mirrors `bson.D` and implements the driver's `bson.Marshaler` interface. It works with mongo-go-driver v1 and v2.

#### Find

```go
q, err := gqbd.MongoFind("users").
    Eq("status", "active").
    Gte("age", 18).Lt("age", 65).          // same field: {"age": {"$gte": 18, "$lt": 65}}
    In("role", []string{"admin", "editor"}).
    Or(gqbd.MongoFilter().Eq("country", "KR"), gqbd.MongoFilter().Exists("vip", true)).
    Sort("created_at", -1).Sort("name", 1). // keeps this order
    Project("name", "email").
    Limit(10).Skip(20).
    Build()

// Every field can be passed as-is: an empty Filter matches all documents,
// an empty Sort/Projection is a no-op, and Limit/Skip 0 mean none.
cur, err := db.Collection(q.Collection).Find(ctx, q.Filter,
    options.Find().SetSort(q.Sort).SetProjection(q.Projection).SetLimit(q.Limit).SetSkip(q.Skip))
```

Also available: `Ne`, `Gt`, `Lte`, `Nin`, `Regex(field, pattern, options)`, and `Raw(key, value)` for anything else (e.g. `$text`, `$expr`).

#### Update and Delete

```go
q, err := gqbd.MongoUpdate("users").
    Eq("_id", id).
    Set(map[string]interface{}{"name": "Kim"}).
    Inc("logins", 1).
    Push("tags", "new").
    Upsert().
    Build()
// q.Update: {"$set": {"name": "Kim"}, "$inc": {"logins": 1}, "$push": {"tags": "new"}}
db.Collection(q.Collection).UpdateOne(ctx, q.Filter, q.Update, options.Update().SetUpsert(q.Upsert))

q, err = gqbd.MongoDelete("sessions").Lt("expires_at", time.Now()).Build()
db.Collection(q.Collection).DeleteMany(ctx, q.Filter)
```

Also available: `Unset`, `AddToSet`, `Pull`.

#### Aggregation

`Build()` returns `(collection, pipeline, err)`, the same shape as the SQL builders:

```go
coll, pipeline, err := gqbd.MongoAggregate("orders").
    Match(gqbd.MongoFilter().Eq("status", "paid")).
    Group("$customer_id", map[string]interface{}{"total": map[string]interface{}{"$sum": "$amount"}}).
    Sort("total", -1).
    Limit(10).
    Build()
cur, err := db.Collection(coll).Aggregate(ctx, pipeline)
```

Stages: `Match`, `Group`, `Sort`, `Project`, `Limit`, `Skip`, `Unwind`, `Lookup`, `Count`, and `Stage(name, spec)` for any other stage.

#### Safety

- Field names starting with `$` are rejected, so a field name taken from user input can't inject an operator.
- `Eq` with a document-like value (e.g. a map decoded from a JSON body such as `{"$ne": null}`) is emitted as `{"$eq": value}`, so it is compared literally instead of being read as operators.

## Database Connection Strings

GQBD includes built-in support for generating database connection strings for use with `sql.Open()`.

### Connection String Generation

```go
import "github.com/donghquinn/gqbd"

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

// MySQL connection string
mysqlConfig := gqbd.DBConfig{
    Host:     "localhost",
    Port:     3306,
    User:     "root",
    Password: "password",
    DBName:   "myapp",
    Charset:  "utf8mb4",
}
mysqlDSN := gqbd.BuildConnectionString(gqbd.Mysql, mysqlConfig)
// Result: "root:password@tcp(localhost:3306)/myapp?charset=utf8mb4&parseTime=True&loc=Local"

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

// Redis URL (for go-redis redis.ParseURL, etc.)
redisConfig := gqbd.DBConfig{
    Host:     "localhost",
    Port:     6379,
    Password: "secret",
    DBName:   "0",       // logical database number
    SSLMode:  "disable", // anything else selects rediss://
}
redisURL := gqbd.BuildConnectionString(gqbd.Redis, redisConfig)
// Result: "redis://:secret@localhost:6379/0"

// MongoDB URI (for options.Client().ApplyURI)
mongoConfig := gqbd.DBConfig{
    Host:     "localhost",
    Port:     27017,
    User:     "app",
    Password: "secret",
    DBName:   "shop",    // default auth database
    SSLMode:  "require", // anything other than "disable" adds tls=true
}
mongoURI := gqbd.BuildConnectionString(gqbd.MongoDB, mongoConfig)
// Result: "mongodb://app:secret@localhost:27017/shop?tls=true"
```

### Complete Database Setup Example

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

### Advanced Features

#### IN Clause

```go
qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id", "name").
    WhereIn("status", []interface{}{"active", "pending", "verified"})

query, args, err := qb.Build()
// Query: SELECT "id", "name" FROM "users" WHERE "status" IN ($1, $2, $3)
// Args: [active pending verified]
```

#### BETWEEN Clause

```go
qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders", "id", "total").
    WhereBetween("created_at", "2023-01-01", "2023-12-31")

query, args, err := qb.Build()
// Query: SELECT "id", "total" FROM "orders" WHERE "created_at" BETWEEN $1 AND $2
// Args: [2023-01-01 2023-12-31]
```

#### Aggregate Functions

```go
qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders", "customer_id").
    Aggregate("COUNT", "*").
    Aggregate("SUM", "total").
    GroupBy("customer_id").
    Having("COUNT(*) > ?", 5)

query, args, err := qb.Build()
// Query: SELECT "customer_id", COUNT(*), SUM("total") FROM "orders" GROUP BY "customer_id" HAVING COUNT(*) > $1
```

## Database Driver Integration

GQBD works with any Go database driver. Here are the recommended drivers for each database type:

### PostgreSQL
```go
import _ "github.com/lib/pq"              // Most common
// OR
import _ "github.com/jackc/pgx/v5/stdlib" // High performance
```

### MySQL
```go
import _ "github.com/go-sql-driver/mysql"
```

### SQLite
```go
import _ "github.com/mattn/go-sqlite3"
```

### Example Usage with Drivers

```go
// PostgreSQL example
func queryPostgreSQL() {
    config := gqbd.DBConfig{
        Host: "localhost", Port: 5432, User: "postgres",
        Password: "password", DBName: "myapp", SSLMode: "disable",
    }
    dsn := gqbd.BuildConnectionString(gqbd.PostgreSQL, config)
    db, _ := sql.Open("postgres", dsn)
    
    qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id", "name")
    query, args, _ := qb.Where("active = ?", true).Build()
    rows, _ := db.Query(query, args...)
    defer rows.Close()
}

// MySQL example  
func queryMySQL() {
    config := gqbd.DBConfig{
        Host: "localhost", Port: 3306, User: "root",
        Password: "password", DBName: "myapp", Charset: "utf8mb4",
    }
    dsn := gqbd.BuildConnectionString(gqbd.Mysql, config)
    db, _ := sql.Open("mysql", dsn)
    
    qb := gqbd.BuildSelect(gqbd.Mysql, "users", "id", "name")
    query, args, _ := qb.Where("active = ?", true).Build()
    rows, _ := db.Query(query, args...)
    defer rows.Close()
}

// SQLite example
func querySQLite() {
    config := gqbd.DBConfig{FilePath: "./app.db"}
    dsn := gqbd.BuildConnectionString(gqbd.SQLite, config)
    db, _ := sql.Open("sqlite3", dsn)
    
    qb := gqbd.BuildSelect(gqbd.SQLite, "users", "id", "name")
    query, args, _ := qb.Where("active = ?", true).Build()
    rows, _ := db.Query(query, args...)
    defer rows.Close()
}
```

## Supported Database Types

| Database | Constant | Placeholders | Identifiers | RETURNING Support |
|----------|----------|--------------|-------------|-------------------|
| PostgreSQL | `gqbd.PostgreSQL` | `$1, $2, $3...` | `"identifier"` | ✅ Yes |
| MySQL | `gqbd.Mysql` | `?, ?, ?...` | `` `identifier` `` | ❌ No |
| MariaDB | `gqbd.MariaDB` | `?, ?, ?...` | `` `identifier` `` | ❌ No |
| SQLite | `gqbd.SQLite` | `?, ?, ?...` | `"identifier"` | ✅ Yes (3.35.0+) |
| Redis | `gqbd.Redis` | — (command builders) | — | — |
| MongoDB | `gqbd.MongoDB` | — (filter/update/pipeline builders) | — | — |

## Benchmarks

Run with `go test -bench . -benchmem` (Apple M-series, Go 1.27). See `benchmark_test.go` for the queries.

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| SELECT, PostgreSQL (join, 3 WHERE, ORDER BY, LIMIT/OFFSET) | 703 | 1680 | 36 |
| SELECT, MySQL (same query) | 681 | 1680 | 36 |
| INSERT, PostgreSQL (4 columns, RETURNING) | 315 | 616 | 10 |
| UPDATE, PostgreSQL (3 columns, 2 WHERE) | 336 | 704 | 13 |
| DELETE, MySQL | 114 | 440 | 7 |

## Why Choose GQBD?

🎯 **Same design philosophy as gdct**:
- Low allocations in critical paths
- SQL injection prevention by design
- Cross-database compatibility
- Pure query building (no database connections)

🚀 **Performance focused**:
- Minimal overhead
- Optimized for high-throughput applications
- Memory efficient

🔒 **Security first**:
- Automatic parameter binding
- Identifier escaping
- Input validation

📋 **Complete solution**:
- Query building for all major databases
- Connection string generation
- Database-specific optimizations
- Modular, maintainable code structure

## Project Structure

- `gqbd.go` - Public API: builder constructors and chainable clause methods
- `build.go` - Renders SELECT/INSERT/UPDATE/DELETE in a single pass
- `dialect.go` - Per-database differences (identifier quoting, placeholders, RETURNING) and the SQL writer
- `connection.go` - Connection string generation
- `redis.go` - Redis command builders
- `mongodb.go` - MongoDB find/update/delete and aggregation builders
- `bson.go` - `gqbd.D`, an ordered document with a minimal BSON encoder (so the driver isn't a dependency)

Database differences live in one dialect table, so adding a SQL database means adding one `dialect` entry.

## Contributing

Feel free to open issues or submit pull requests. All major SQL databases are now supported!
