# Qdrant

[← Back to README](../README.md)

Qdrant gets its own builders (the SQL builders return an error for `gqbd.Qdrant`). They build requests for Qdrant's **REST API** (port 6333): `Build()` returns `(path, body, err)`, where `path` is the endpoint with the collection name percent-encoded and `body` is the JSON request body. Send it with `net/http`; gqbd imports no Qdrant client.

The official Go client (`github.com/qdrant/go-client`) uses gRPC with generated types, so it can't take these bodies.

| Builder | Endpoint | Accepts |
|---------|----------|---------|
| `QdrantQuery(collection)` | `POST /collections/{c}/points/query` | filters, `Nearest`, `NearestPoint`, `Using`, `Limit`, `Offset`, `WithPayload`, `WithVector`, `ScoreThreshold`, `Exact` |
| `QdrantCount(collection)` | `POST /collections/{c}/points/count` | filters |
| `QdrantDelete(collection)` | `POST /collections/{c}/points/delete` | filters (at least one is required) |
| `QdrantFilter()` | — | a standalone filter for `Or` and `Not` |

## Vector search

```go
path, body, err := gqbd.QdrantQuery("docs").
    Nearest(embedding).                 // []float32
    Match("lang", "en").
    Gte("price", 10).Lt("price", 50).   // same key: one range condition
    Limit(5).
    WithPayload("title", "price").      // or WithPayload() for the whole payload
    Build()
// path: /collections/docs/points/query
// body: {"filter":{"must":[{"key":"lang","match":{"value":"en"}},{"key":"price","range":{"gte":10,"lt":50}}]},
//        "limit":5,"query":[...],"with_payload":["title","price"]}

req, _ := http.NewRequest("POST", addr+path, bytes.NewReader(body))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("api-key", apiKey) // if the server requires one
res, err := http.DefaultClient.Do(req)
```

- `NearestPoint(id)` searches by an existing point's vector instead; Qdrant leaves that point out of the results.
- `Using(name)` picks a named vector when the collection has more than one.
- `ScoreThreshold(s)` drops worse matches. Whether "worse" means lower or higher depends on the distance metric.
- `Exact()` skips the HNSW index and checks every point.
- Without `Nearest` or `NearestPoint`, the query lists the points that match the filter, in ID order.

## Filters

| Method | Condition |
|--------|-----------|
| `Match(key, value)` | equals; value must be a string, integer or bool |
| `MatchAny(key, values)` | equals any (IN) |
| `MatchExcept(key, values)` | equals none (NOT IN) |
| `MatchText(key, text)` | full-text match (needs a full-text index on `key`) |
| `Gt`, `Gte`, `Lt`, `Lte` | range; numbers or RFC 3339 datetimes |
| `IsEmpty(key)` | missing, null or `[]` |
| `IsNull(key)` | null |
| `HasID(ids...)` | point IDs (unsigned integers or UUID strings) |
| `Or(filters...)` | at least one filter matches; calling it again ANDs the groups |
| `Not(filters...)` | none of the filters match |

```go
gqbd.QdrantQuery("docs").
    Match("lang", "en").
    Or(gqbd.QdrantFilter().Match("vip", true), gqbd.QdrantFilter().HasID(2)).
    Not(gqbd.QdrantFilter().IsEmpty("tags"))
```

`Match` only accepts strings, integers and bools, which are the types Qdrant can match exactly. So a map from user input (such as decoded JSON `{"any": [...]}`) is an error, not a different kind of match.

## Count and Delete

```go
path, body, err := gqbd.QdrantCount("docs").Gt("price", 20).Build()

path, body, err = gqbd.QdrantDelete("docs").Match("lang", "ja").Build()
// Add "?wait=true" to the path to wait until the delete is applied.
```

`QdrantDelete` returns an error from `Build()` without a filter condition, so a missing condition can't empty the collection.

## Connection String

```go
addr := gqbd.BuildConnectionString(gqbd.Qdrant, gqbd.DBConfig{Host: "localhost", Port: 6333})
// Result: "http://localhost:6333" (https:// when SSLMode is not "disable")
```

Qdrant authenticates with an `api-key` header, so `User` and `Password` are never put in the address.
