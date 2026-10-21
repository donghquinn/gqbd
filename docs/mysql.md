# MySQL / MariaDB

[← Back to README](../README.md)

MySQL and MariaDB (`gqbd.Mysql`, `gqbd.MariaDB`) use `?` placeholders and backticks for identifiers. `Returning()` is ignored for these databases.

See [SQL builder](sql.md) for the API shared by all SQL databases (IN, BETWEEN, aggregates, dynamic conditions).

## SELECT Query

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

## INSERT Query

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

## UPDATE Query

```go
qb := gqbd.BuildUpdate(gqbd.Mysql, "products").
    Set(map[string]interface{}{"price": 89.99, "stock": 10}).
    Where("id = ?", 42)

query, args, err := qb.Build()
// Query: UPDATE `products` SET `price` = ?, `stock` = ? WHERE id = ?
// Args: [89.99 10 42]
```

## Connection String

```go
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
```

The same DSN works for MariaDB (`gqbd.MariaDB`).

## Driver

```go
import _ "github.com/go-sql-driver/mysql"

db, err := sql.Open("mysql", dsn)
```
