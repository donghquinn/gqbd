# GQBD - High-Performance Go Query Builder

## Introduction
**GQBD** is a high-performance, zero-allocation SQL query builder for Go, inspired by gdct design principles.

✨ **Key Features:**
- 🚀 **Zero allocations** in critical query building paths
- 🔒 **SQL injection safe** with automatic parameter binding and identifier escaping  
- 🎯 **Multi-database support**: PostgreSQL, MySQL, MariaDB
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
- ✅ **Cross-Database**: PostgreSQL ($N), MySQL/MariaDB (?) placeholder formats

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

## Integration with Database Drivers

This package only builds queries - you'll need a database driver to execute them:

```go
// With database/sql and pq (PostgreSQL)
import (
    "database/sql"
    _ "github.com/lib/pq"
    "github.com/donghquinn/gqbd"
)

func queryUsers(db *sql.DB, status string) error {
    qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id", "name", "email").
        Where("status = ?", status).
        OrderBy("name", "ASC", nil)
    
    query, args, err := qb.Build()
    if err != nil {
        return err
    }
    
    rows, err := db.Query(query, args...)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    // Process rows...
    return nil
}
```

## Supported Database Types

- `gqbd.PostgreSQL` - PostgreSQL with `$N` placeholders and `"` identifiers
- `gqbd.MariaDB` - MariaDB with `?` placeholders and `` ` `` identifiers  
- `gqbd.Mysql` - MySQL with `?` placeholders and `` ` `` identifiers

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

## Contributing

Feel free to open issues or submit pull requests. Planning to add support for SQLite and other databases.

## License

MIT License - use freely in commercial and open-source projects.
