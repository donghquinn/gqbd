# Oracle

[← Back to README](../README.md)

Oracle (`gqbd.Oracle`) uses `:N` placeholders and double quotes for identifiers. Write conditions with `?`; gqbd numbers them `:1, :2, ...` in query order when `Build()` is called. `LIMIT`/`OFFSET` are rendered as `OFFSET :n ROWS FETCH NEXT :m ROWS ONLY`, which needs Oracle 12c or later. `RETURNING` is not supported (Oracle's `RETURNING ... INTO` needs output binds) and `Returning()` is ignored.

See [SQL builder](sql.md) for the API shared by all SQL databases (IN, BETWEEN, aggregates, dynamic conditions).

## Identifiers are case-sensitive

Identifiers are quoted exactly as written, and quoted names are case-sensitive in Oracle. Oracle stores unquoted names in upper case, so a table created as `CREATE TABLE users` is `USERS`: write `"USERS"` and `"ID"` in the builder, not `"users"` and `"id"`. Names created with quotes (`CREATE TABLE "MyTable"`) are written exactly as they were created. Quoting also protects reserved words such as `LEVEL` or `DATE`.

Oracle doesn't allow `AS` before a table alias, so `"USERS AS u"` is written as `"USERS" u`. Text you pass in conditions (`Where`, `Having`, join `ON` clauses) is not changed, so unquoted names there are folded by Oracle itself.

## SELECT Query

```go
qb := gqbd.BuildSelect(gqbd.Oracle, "USERS u", "u.ID", "u.NAME", "o.AMOUNT").
    InnerJoin("ORDERS o", "o.user_id = u.id").
    Where("u.status = ?", "active").
    OrderBy("u.ID", "ASC", nil).
    Limit(10).
    Offset(20)

query, args, err := qb.Build()
// Query: SELECT u."ID", u."NAME", o."AMOUNT" FROM "USERS" u INNER JOIN "ORDERS" o ON o.user_id = u.id WHERE u.status = :1 ORDER BY u."ID" ASC OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY
// Args: [active 20 10]
```

`OFFSET` comes before `FETCH` in Oracle, so the offset value comes before the limit in `args`.

## INSERT, UPDATE and DELETE

```go
query, args, err := gqbd.BuildInsert(gqbd.Oracle, "USERS").
    Values(map[string]interface{}{"ID": 1, "NAME": "Alice"}).
    Build()
// Query: INSERT INTO "USERS" ("ID", "NAME") VALUES (:1, :2)
// Args: [1 Alice]

query, args, err = gqbd.BuildUpdate(gqbd.Oracle, "USERS").
    Set(map[string]interface{}{"STATUS": "inactive"}).
    Where("id = ?", 1).
    Build()
// Query: UPDATE "USERS" SET "STATUS" = :1 WHERE id = :2
// Args: [inactive 1]

query, args, err = gqbd.BuildDelete(gqbd.Oracle, "USERS").
    Where("id = ?", 1).
    Build()
// Query: DELETE FROM "USERS" WHERE id = :1
```

## Connection String

```go
// go-ora URL: DBName is the service name
oraConfig := gqbd.DBConfig{
    Host:     "localhost",
    Port:     1521,
    User:     "app",
    Password: "secret",
    DBName:   "FREEPDB1",
    SSLMode:  "disable", // anything else adds ssl=true (use the TCPS port, usually 2484)
}
oraDSN := gqbd.BuildConnectionString(gqbd.Oracle, oraConfig)
// Result: "oracle://app:secret@localhost:1521/FREEPDB1"
```

## Driver

```go
import _ "github.com/sijms/go-ora/v2"

db, err := sql.Open("oracle", dsn)
rows, err := db.Query(query, args...)
```

go-ora binds `:1, :2, ...` by position, so pass `args` as returned. godror binds them by position too.
