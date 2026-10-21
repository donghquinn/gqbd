# MongoDB

[← Back to README](../README.md)

MongoDB gets its own builders (the SQL builders return an error for `gqbd.MongoDB`). gqbd does **not** import the MongoDB driver: filters, updates and pipeline stages are plain `map[string]interface{}` values, which the driver encodes itself. Sorts need key order, so they use `gqbd.D`, which mirrors `bson.D` and implements the driver's `bson.Marshaler` interface. It works with mongo-go-driver v1 and v2.

## Find

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

## Update and Delete

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

## Aggregation

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

## Safety

- Field names starting with `$` are rejected, so a field name taken from user input can't inject an operator.
- `Eq` with a document-like value (e.g. a map decoded from a JSON body such as `{"$ne": null}`) is emitted as `{"$eq": value}`, so it is compared literally instead of being read as operators.

## Connection String

```go
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

## Driver

```go
import "go.mongodb.org/mongo-driver/v2/mongo" // v1 (go.mongodb.org/mongo-driver/mongo) also works

client, err := mongo.Connect(options.Client().ApplyURI(uri))
coll := client.Database("shop").Collection(q.Collection)
```
