# Elasticsearch

[English](../elasticsearch.md) | **한국어**

[← README로 돌아가기](../../README.ko.md)

Elasticsearch는 별도의 빌더를 사용합니다(`gqbd.Elasticsearch`에 대해 SQL 빌더는 에러를 반환합니다). `Build()`는 SQL 빌더와 같은 형태인 `(index, body, err)`를 반환하며, `body`는 JSON 요청 본문입니다. gqbd는 Elasticsearch 클라이언트를 import하지 **않고** `encoding/json`만 사용하므로, 본문은 go-elasticsearch v7, v8, v9이나 일반 HTTP에서 모두 사용할 수 있습니다.

| 빌더 | API | 지원 |
|---------|-----|---------|
| `ESSearch(index)` | `_search` | 쿼리, `Sort`, `Source`, `Size`, `From`, `Agg` |
| `ESCount(index)` | `_count` | 쿼리 |
| `ESDeleteByQuery(index)` | `_delete_by_query` | 쿼리(최소 하나 필요) |
| `ESQuery()` | — | `Or`와 `Not`에 쓰는 독립 쿼리 |

`index`에는 여러 인덱스나 패턴(`"logs-*"`, `"a,b"`)을 지정할 수 있습니다. 요청 경로를 바꿀 수 있는 문자(`/`, `\`, `?`, `#`, 공백 등)는 거부됩니다.

## Search

```go
index, body, err := gqbd.ESSearch("products").
    Match("title", "wireless headphones").         // 전문 검색, 점수 계산
    Term("status", "active").                      // 정확히 일치, filter 컨텍스트
    Terms("brand", []string{"sony", "bose"}).
    Gte("price", 100).Lt("price", 500).            // 같은 필드: 하나의 range 쿼리
    Not(gqbd.ESQuery().Exists("discontinued")).
    Sort("price", "desc").
    Source("title", "price").
    Size(10).From(20).
    Build()

res, err := es.Search(es.Search.WithIndex(index), es.Search.WithBody(bytes.NewReader(body)))
```

본문:

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

전문 검색 쿼리(`Match`, `MatchPhrase`, `MultiMatch`)는 `must`에 들어가므로 점수에 영향을 줍니다. 정확히 일치하는 쿼리(`Term`, `Terms`, `Gt`/`Gte`/`Lt`/`Lte`, `Exists`, `Prefix`)는 `filter`에 들어가며, 점수를 계산하지 않고 캐시될 수 있습니다. 쿼리가 없는 검색에는 `query` 키가 없으므로 모든 문서와 일치합니다.

`Term`은 긴 형식 `{"term": {field: {"value": v}}}`으로 작성되므로, 객체 값(예: 요청에서 디코딩한 JSON)은 `boost` 같은 term 옵션이 되지 않고 값으로 남으며, Elasticsearch가 이를 거부합니다.

## OR와 NOT

```go
gqbd.ESSearch("products").
    Term("status", "active").
    Or(gqbd.ESQuery().Prefix("sku", "AC"), gqbd.ESQuery().MatchPhrase("title", "wireless speaker"))
```

`Or`는 쿼리 중 최소 하나가 일치해야 합니다. Elasticsearch에서는 `must`나 `filter` 절과 함께 쓴 단독 `should`가 선택 사항이 되므로, `"minimum_should_match": 1`을 지정한 중첩 `bool`로 작성됩니다. `Or`를 다시 호출하면 그룹들이 AND로 결합됩니다. `Not`은 각 쿼리를 `must_not`에 추가합니다.

## Aggregations

`Agg(name, body)`는 집계를 변경 없이 그대로 전달합니다. 집계와 전체 개수만 반환하려면 `Size(0)`을 사용합니다.

```go
index, body, err := gqbd.ESSearch("products").
    MultiMatch("bose", "title^2", "brand").
    Size(0).
    Agg("by_status", map[string]interface{}{"terms": map[string]interface{}{"field": "status"}}).
    Build()
```

## 기타 쿼리

`Raw(clause, query)`는 전용 메서드가 없는 쿼리를 `"must"`, `"filter"`, `"must_not"`에 추가합니다.

```go
gqbd.ESSearch("stores").Raw("filter", map[string]interface{}{
    "geo_distance": map[string]interface{}{"distance": "5km", "location": map[string]interface{}{"lat": 37.5, "lon": 127.0}},
})
```

`MatchAll()`은 모든 문서와 일치합니다.

## Count와 Delete by Query

```go
index, body, err := gqbd.ESCount("products").Term("brand", "sony").Build()
res, err := es.Count(es.Count.WithIndex(index), es.Count.WithBody(bytes.NewReader(body)))

index, body, err = gqbd.ESDeleteByQuery("products").Term("status", "archived").Build()
res, err = es.DeleteByQuery([]string{index}, bytes.NewReader(body))
```

`ESDeleteByQuery`는 쿼리가 하나도 추가되지 않으면 `Build()`에서 에러를 반환하므로, 조건이 빠져 인덱스 전체가 삭제되는 일이 없습니다. 의도적으로 모든 문서를 삭제하려면 `MatchAll()`을 사용합니다.

## 연결 문자열

```go
esConfig := gqbd.DBConfig{
    Host:     "localhost",
    Port:     9200,
    User:     "elastic",
    Password: "secret",
    SSLMode:  "disable", // 다른 값이면 https://를 사용합니다
}
addr := gqbd.BuildConnectionString(gqbd.Elasticsearch, esConfig)
// Result: "http://elastic:secret@localhost:9200"

es, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{addr}})
```

go-elasticsearch는 주소에 포함된 자격 증명을 basic auth에 사용합니다.
