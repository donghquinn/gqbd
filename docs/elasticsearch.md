# Elasticsearch

[← Back to README](../README.md)

Elasticsearch gets its own builders (the SQL builders return an error for `gqbd.Elasticsearch`). `Build()` returns `(index, body, err)`, the same shape as the SQL builders, where `body` is the JSON request body. gqbd does **not** import the Elasticsearch client; it uses only `encoding/json`, so the body works with go-elasticsearch v7, v8 and v9 or plain HTTP.

| Builder | API | Accepts |
|---------|-----|---------|
| `ESSearch(index)` | `_search` | queries, `Sort`, `Source`, `Size`, `From`, `Agg` |
| `ESCount(index)` | `_count` | queries |
| `ESDeleteByQuery(index)` | `_delete_by_query` | queries (at least one is required) |
| `ESQuery()` | — | a standalone query for `Or` and `Not` |

`index` can name several indices or a pattern (`"logs-*"`, `"a,b"`). Characters that could change the request path (`/`, `\`, `?`, `#`, spaces, ...) are rejected.

## Search

```go
index, body, err := gqbd.ESSearch("products").
    Match("title", "wireless headphones").         // full text, scored
    Term("status", "active").                      // exact, filter context
    Terms("brand", []string{"sony", "bose"}).
    Gte("price", 100).Lt("price", 500).            // same field: one range query
    Not(gqbd.ESQuery().Exists("discontinued")).
    Sort("price", "desc").
    Source("title", "price").
    Size(10).From(20).
    Build()

res, err := es.Search(es.Search.WithIndex(index), es.Search.WithBody(bytes.NewReader(body)))
```

Body:

```json
{
  "_source": ["title", "price"],
  "from": 20,
  "query": {"bool": {
    "filter": [
      {"term": {"status": {"value": "active"}}},
      {"terms": {"brand": ["sony", "bose"]}},
      {"range": {"price": {"gte": 100, "lt": 500}}}
    ],
    "must": [{"match": {"title": "wireless headphones"}}],
    "must_not": [{"bool": {"filter": [{"exists": {"field": "discontinued"}}]}}]
  }},
  "size": 10,
  "sort": [{"price": {"order": "desc"}}]
}
```

Full-text queries (`Match`, `MatchPhrase`, `MultiMatch`) go into `must`, so they affect the score. Exact queries (`Term`, `Terms`, `Gt`/`Gte`/`Lt`/`Lte`, `Exists`, `Prefix`) go into `filter`, which doesn't score and can be cached. A search with no queries has no `query` key, so it matches every document.

`Term` writes the long form `{"term": {field: {"value": v}}}`, so an object value (for example decoded JSON from a request) stays a value, which Elasticsearch rejects, instead of becoming term options such as `boost`.

## OR and NOT

```go
gqbd.ESSearch("products").
    Term("status", "active").
    Or(gqbd.ESQuery().Prefix("sku", "AC"), gqbd.ESQuery().MatchPhrase("title", "wireless speaker"))
```

`Or` requires at least one of its queries to match. It is written as a nested `bool` with `"minimum_should_match": 1`, because a bare `should` next to `must` or `filter` clauses is optional in Elasticsearch. Calling `Or` again ANDs the groups together. `Not` adds each query to `must_not`.

## Aggregations

`Agg(name, body)` passes the aggregation through unchanged. Use `Size(0)` to return only aggregations and the total.

```go
index, body, err := gqbd.ESSearch("products").
    MultiMatch("bose", "title^2", "brand").
    Size(0).
    Agg("by_status", map[string]interface{}{"terms": map[string]interface{}{"field": "status"}}).
    Build()
```

## Other queries

`Raw(clause, query)` adds any query without a dedicated method to `"must"`, `"filter"` or `"must_not"`:

```go
gqbd.ESSearch("stores").Raw("filter", map[string]interface{}{
    "geo_distance": map[string]interface{}{"distance": "5km", "location": map[string]interface{}{"lat": 37.5, "lon": 127.0}},
})
```

`MatchAll()` matches every document.

## Count and Delete by Query

```go
index, body, err := gqbd.ESCount("products").Term("brand", "sony").Build()
res, err := es.Count(es.Count.WithIndex(index), es.Count.WithBody(bytes.NewReader(body)))

index, body, err = gqbd.ESDeleteByQuery("products").Term("status", "archived").Build()
res, err = es.DeleteByQuery([]string{index}, bytes.NewReader(body))
```

`ESDeleteByQuery` returns an error from `Build()` if no query was added, so a missing condition can't delete the whole index. Use `MatchAll()` to delete every document on purpose.

## Connection String

```go
esConfig := gqbd.DBConfig{
    Host:     "localhost",
    Port:     9200,
    User:     "elastic",
    Password: "secret",
    SSLMode:  "disable", // anything else uses https://
}
addr := gqbd.BuildConnectionString(gqbd.Elasticsearch, esConfig)
// Result: "http://elastic:secret@localhost:9200"

es, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{addr}})
```

go-elasticsearch uses the credentials in the address for basic auth.
