# GQBD - High-Performance Go Query Builder

## Introduction
**GQBD** is a high-performance, zero-allocation SQL query builder for Go, inspired by gdct design principles.

✨ **Key Features:**
- 🚀 **Zero allocations** in critical query building paths
- 🔒 **SQL injection safe** with automatic parameter binding and identifier escaping  
- 🎯 **Multi-database support**: PostgreSQL, MySQL, MariaDB, SQLite
- ⚡ **Performance optimized** - designed for high-throughput applications
- 📦 **Zero dependencies** - pure Go implementation
- 🔧 **Fluent API** - chainable method design for readable code

## Installation

```bash
go get github.com/donghquinn/gqbd
```

## Performance & Features

🏆 **Performance Benefits:**
- Zero allocations in query building (similar to gdct)
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
- ✅ **Connection Strings**: Built-in database connection string generation

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
    // Query: SELECT "e"."id", "e"."name", "u"."username" FROM "example_table" e LEFT JOIN "user_table" u ON u.user_id = e.id WHERE e.status = $1 ORDER BY "e"."created_at" DESC LIMIT $2 OFFSET $3
    // Args: [active 10 0]
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
// Query: INSERT INTO "users" ("name", "email", "active") VALUES ($1, $2, $3) RETURNING id, created_at
// Args: [John Doe john@example.com true]
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
// Query: INSERT INTO `products` (`name`, `price`, `category_id`) VALUES (?, ?, ?)
// Args: [New Product 99.99 5]
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
// Query: INSERT INTO "users" ("name", "email", "age") VALUES (?, ?, ?) RETURNING id, name
// Args: [Alice alice@example.com 28]
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
// Query: SELECT "customer_id", COUNT("*"), SUM("total") FROM "orders" GROUP BY "customer_id" HAVING COUNT(*) > $1
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

## Performance Comparison

GQBD is designed with the same performance principles as gdct:

```go
// Benchmark example - zero allocations in hot path
func BenchmarkQueryBuilder(b *testing.B) {
    for i := 0; i < b.N; i++ {
        query, args, _ := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id", "name").
            Where("active = ?", true).
            Where("age > ?", 21).
            OrderBy("created_at", "DESC", nil).
            Limit(100).
            Build()
        _ = query
        _ = args
    }
}
```

## Why Choose GQBD?

🎯 **Same design philosophy as gdct**:
- Zero allocations in critical paths
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

The codebase is organized for maintainability and database-specific optimizations:

- `gqbd.go` - Main API, shared logic, and database selection
- `postgres.go` - PostgreSQL-specific query building methods
- `mariadb.go` - MySQL/MariaDB-specific query building methods  
- `sqlite.go` - SQLite-specific query building methods

This separation allows for:
- Database-specific optimizations
- Clean, maintainable code
- Easy addition of new database types
- Consistent API across all databases

## Contributing

Feel free to open issues or submit pull requests. All major SQL databases are now supported!
