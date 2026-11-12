# MongoDB

[English](../mongodb.md) | **한국어**

[← README로 돌아가기](../../README.ko.md)

MongoDB는 별도의 빌더를 사용합니다(`gqbd.MongoDB`에 대해 SQL 빌더는 에러를 반환합니다). gqbd는 MongoDB 드라이버를 import하지 **않습니다**. 필터, 업데이트, 파이프라인 스테이지는 일반 `map[string]interface{}` 값이며 드라이버가 직접 인코딩합니다. 정렬은 키 순서가 필요하므로 `gqbd.D`를 사용합니다. `gqbd.D`는 `bson.D`와 같은 구조이며 드라이버의 `bson.Marshaler` 인터페이스를 구현합니다. mongo-go-driver v1과 v2 모두에서 동작합니다.

## Find

```go
q, err := gqbd.MongoFind("users").
    Eq("status", "active").
    Gte("age", 18).Lt("age", 65).          // 같은 필드: {"age": {"$gte": 18, "$lt": 65}}
    In("role", []string{"admin", "editor"}).
    Or(gqbd.MongoFilter().Eq("country", "KR"), gqbd.MongoFilter().Exists("vip", true)).
    Sort("created_at", -1).Sort("name", 1). // 이 순서를 유지
    Project("name", "email").
    Limit(10).Skip(20).
    Build()

// 모든 필드는 그대로 전달할 수 있습니다. 빈 Filter는 모든 문서와 일치하고,
// 빈 Sort/Projection은 아무 효과가 없으며, Limit/Skip 0은 제한 없음을 뜻합니다.
cur, err := db.Collection(q.Collection).Find(ctx, q.Filter,
    options.Find().SetSort(q.Sort).SetProjection(q.Projection).SetLimit(q.Limit).SetSkip(q.Skip))
```

그 밖에 `Ne`, `Gt`, `Lte`, `Nin`, `Regex(field, pattern, options)`, 그리고 다른 모든 연산(예: `$text`, `$expr`)을 위한 `Raw(key, value)`도 사용할 수 있습니다.

## Update와 Delete

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

그 밖에 `Unset`, `AddToSet`, `Pull`도 사용할 수 있습니다.

## Aggregation

`Build()`는 SQL 빌더와 같은 형태인 `(collection, pipeline, err)`를 반환합니다.

```go
coll, pipeline, err := gqbd.MongoAggregate("orders").
    Match(gqbd.MongoFilter().Eq("status", "paid")).
    Group("$customer_id", map[string]interface{}{"total": map[string]interface{}{"$sum": "$amount"}}).
    Sort("total", -1).
    Limit(10).
    Build()
cur, err := db.Collection(coll).Aggregate(ctx, pipeline)
```

스테이지: `Match`, `Group`, `Sort`, `Project`, `Limit`, `Skip`, `Unwind`, `Lookup`, `Count`, 그리고 그 밖의 스테이지를 위한 `Stage(name, spec)`.

## 안전성

- `$`로 시작하는 필드 이름은 거부되므로, 사용자 입력에서 온 필드 이름으로 연산자를 인젝션할 수 없습니다.
- 문서 형태의 값(예: `{"$ne": null}` 같은 JSON 본문에서 디코딩한 map)을 `Eq`에 전달하면 `{"$eq": value}`로 생성되므로, 연산자로 해석되지 않고 값 그대로 비교됩니다.

## 연결 문자열

```go
// MongoDB URI (options.Client().ApplyURI에서 사용)
mongoConfig := gqbd.DBConfig{
    Host:     "localhost",
    Port:     27017,
    User:     "app",
    Password: "secret",
    DBName:   "shop",    // 기본 인증 데이터베이스
    SSLMode:  "require", // "disable" 이외의 값이면 tls=true가 추가됩니다
}
mongoURI := gqbd.BuildConnectionString(gqbd.MongoDB, mongoConfig)
// Result: "mongodb://app:secret@localhost:27017/shop?tls=true"
```

## 드라이버

```go
import "go.mongodb.org/mongo-driver/v2/mongo" // v1 (go.mongodb.org/mongo-driver/mongo)도 동작합니다

client, err := mongo.Connect(options.Client().ApplyURI(uri))
coll := client.Database("shop").Collection(q.Collection)
```
