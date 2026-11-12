# MySQL / MariaDB

[English](../mysql.md) | **한국어**

[← README로 돌아가기](../../README.ko.md)

MySQL과 MariaDB(`gqbd.Mysql`, `gqbd.MariaDB`)는 `?` 플레이스홀더와 백틱 식별자를 사용합니다. 이 데이터베이스에서는 `Returning()`이 무시됩니다.

모든 SQL 데이터베이스에 공통인 API(IN, BETWEEN, 집계, 동적 조건)는 [SQL 빌더](./sql.md)를 참고하세요.

## SELECT 쿼리

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

## INSERT 쿼리

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

## UPDATE 쿼리

```go
qb := gqbd.BuildUpdate(gqbd.Mysql, "products").
    Set(map[string]interface{}{"price": 89.99, "stock": 10}).
    Where("id = ?", 42)

query, args, err := qb.Build()
// Query: UPDATE `products` SET `price` = ?, `stock` = ? WHERE id = ?
// Args: [89.99 10 42]
```

## 연결 문자열

```go
// MySQL 연결 문자열
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

같은 DSN을 MariaDB(`gqbd.MariaDB`)에도 사용할 수 있습니다.

## 드라이버

```go
import _ "github.com/go-sql-driver/mysql"

db, err := sql.Open("mysql", dsn)
```
