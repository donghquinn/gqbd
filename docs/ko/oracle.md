# Oracle

[English](../oracle.md) | **한국어**

[← README로 돌아가기](../../README.ko.md)

Oracle(`gqbd.Oracle`)은 `:N` 플레이스홀더와 큰따옴표 식별자를 사용합니다. 조건은 `?`로 작성하면 `Build()`를 호출할 때 gqbd가 쿼리 순서대로 `:1, :2, ...` 번호를 매깁니다. `LIMIT`/`OFFSET`은 `OFFSET :n ROWS FETCH NEXT :m ROWS ONLY`로 생성되며, Oracle 12c 이상이 필요합니다. `RETURNING`은 지원하지 않으며(Oracle의 `RETURNING ... INTO`는 출력 바인딩이 필요합니다) `Returning()`은 무시됩니다.

모든 SQL 데이터베이스에 공통인 API(IN, BETWEEN, 집계, 동적 조건)는 [SQL 빌더](./sql.md)를 참고하세요.

## 식별자는 대소문자를 구분합니다

식별자는 작성한 그대로 인용되며, Oracle에서 인용된 이름은 대소문자를 구분합니다. Oracle은 인용되지 않은 이름을 대문자로 저장하므로 `CREATE TABLE users`로 만든 테이블은 `USERS`입니다. 따라서 빌더에는 `"users"`와 `"id"`가 아니라 `"USERS"`와 `"ID"`로 씁니다. 따옴표로 만든 이름(`CREATE TABLE "MyTable"`)은 만들 때와 똑같이 씁니다. 인용은 `LEVEL`이나 `DATE` 같은 예약어도 보호합니다.

Oracle은 테이블 별칭 앞에 `AS`를 허용하지 않으므로 `"USERS AS u"`는 `"USERS" u`로 생성됩니다. 조건(`Where`, `Having`, 조인 `ON` 절)에 전달한 텍스트는 변경하지 않으므로, 그 안의 인용되지 않은 이름은 Oracle이 직접 대문자로 변환합니다.

## SELECT 쿼리

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

Oracle에서는 `OFFSET`이 `FETCH`보다 앞에 오므로, `args`에서도 offset 값이 limit 값보다 앞에 옵니다.

## INSERT, UPDATE, DELETE

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

## 연결 문자열

```go
// go-ora URL: DBName은 서비스 이름입니다
oraConfig := gqbd.DBConfig{
    Host:     "localhost",
    Port:     1521,
    User:     "app",
    Password: "secret",
    DBName:   "FREEPDB1",
    SSLMode:  "disable", // 다른 값이면 ssl=true가 추가됩니다(TCPS 포트, 보통 2484 사용)
}
oraDSN := gqbd.BuildConnectionString(gqbd.Oracle, oraConfig)
// Result: "oracle://app:secret@localhost:1521/FREEPDB1"
```

## 드라이버

```go
import _ "github.com/sijms/go-ora/v2"

db, err := sql.Open("oracle", dsn)
rows, err := db.Query(query, args...)
```

go-ora는 `:1, :2, ...`를 위치 기준으로 바인딩하므로 `args`를 반환된 그대로 전달합니다. godror도 위치 기준으로 바인딩합니다.
