# Qdrant

[English](../qdrant.md) | **한국어**

[← README로 돌아가기](../../README.ko.md)

Qdrant는 별도의 빌더를 사용합니다(`gqbd.Qdrant`에 대해 SQL 빌더는 에러를 반환합니다). 이 빌더는 Qdrant **REST API**(포트 6333)용 요청을 만듭니다. `Build()`는 `(path, body, err)`를 반환하며, `path`는 컬렉션 이름이 퍼센트 인코딩된 엔드포인트이고 `body`는 JSON 요청 본문입니다. `net/http`로 전송하며, gqbd는 Qdrant 클라이언트를 import하지 않습니다.

공식 Go 클라이언트(`github.com/qdrant/go-client`)는 생성된 타입과 gRPC를 사용하므로 이 본문을 받을 수 없습니다.

| 빌더 | 엔드포인트 | 지원 |
|---------|----------|---------|
| `QdrantQuery(collection)` | `POST /collections/{c}/points/query` | 필터, `Nearest`, `NearestPoint`, `Using`, `Limit`, `Offset`, `WithPayload`, `WithVector`, `ScoreThreshold`, `Exact` |
| `QdrantCount(collection)` | `POST /collections/{c}/points/count` | 필터 |
| `QdrantDelete(collection)` | `POST /collections/{c}/points/delete` | 필터(최소 하나 필요) |
| `QdrantFilter()` | — | `Or`와 `Not`에 쓰는 독립 필터 |

## 벡터 검색

```go
path, body, err := gqbd.QdrantQuery("docs").
    Nearest(embedding).                 // []float32
    Match("lang", "en").
    Gte("price", 10).Lt("price", 50).   // 같은 키: 하나의 range 조건
    Limit(5).
    WithPayload("title", "price").      // 전체 payload는 WithPayload()
    Build()
// path: /collections/docs/points/query
// body: {"filter":{"must":[{"key":"lang","match":{"value":"en"}},{"key":"price","range":{"gte":10,"lt":50}}]},
//        "limit":5,"query":[...],"with_payload":["title","price"]}

req, _ := http.NewRequest("POST", addr+path, bytes.NewReader(body))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("api-key", apiKey) // 서버가 요구하는 경우
res, err := http.DefaultClient.Do(req)
```

- `NearestPoint(id)`는 대신 기존 포인트의 벡터로 검색합니다. Qdrant는 해당 포인트를 결과에서 제외합니다.
- `Using(name)`은 컬렉션에 벡터가 여러 개일 때 이름 있는 벡터를 선택합니다.
- `ScoreThreshold(s)`는 더 나쁜 결과를 제외합니다. "나쁜"이 더 낮은 값인지 높은 값인지는 거리 메트릭에 따라 다릅니다.
- `Exact()`는 HNSW 인덱스를 건너뛰고 모든 포인트를 검사합니다.
- `Nearest`나 `NearestPoint`가 없으면 쿼리는 필터와 일치하는 포인트를 ID 순서로 나열합니다.

## 필터

| 메서드 | 조건 |
|--------|-----------|
| `Match(key, value)` | 같음. 값은 반드시 문자열, 정수, bool이어야 함 |
| `MatchAny(key, values)` | 하나라도 같음 (IN) |
| `MatchExcept(key, values)` | 모두 같지 않음 (NOT IN) |
| `MatchText(key, text)` | 전문 일치 (`key`에 전문 인덱스 필요) |
| `Gt`, `Gte`, `Lt`, `Lte` | 범위. 숫자 또는 RFC 3339 날짜시간 |
| `IsEmpty(key)` | 없음, null, 또는 `[]` |
| `IsNull(key)` | null |
| `HasID(ids...)` | 포인트 ID (부호 없는 정수 또는 UUID 문자열) |
| `Or(filters...)` | 필터 중 최소 하나가 일치. 다시 호출하면 그룹들을 AND로 결합 |
| `Not(filters...)` | 어떤 필터도 일치하지 않음 |

```go
gqbd.QdrantQuery("docs").
    Match("lang", "en").
    Or(gqbd.QdrantFilter().Match("vip", true), gqbd.QdrantFilter().HasID(2)).
    Not(gqbd.QdrantFilter().IsEmpty("tags"))
```

`Match`는 Qdrant가 정확히 일치시킬 수 있는 타입인 문자열, 정수, bool만 받습니다. 따라서 사용자 입력에서 온 map(예: 디코딩한 JSON `{"any": [...]}`)은 다른 종류의 일치로 해석되지 않고 에러가 됩니다.

## Count와 Delete

```go
path, body, err := gqbd.QdrantCount("docs").Gt("price", 20).Build()

path, body, err = gqbd.QdrantDelete("docs").Match("lang", "ja").Build()
// 삭제가 적용될 때까지 기다리려면 path에 "?wait=true"를 추가합니다.
```

`QdrantDelete`는 필터 조건이 없으면 `Build()`에서 에러를 반환하므로, 조건이 빠져 컬렉션이 비워지는 일이 없습니다.

## 연결 문자열

```go
addr := gqbd.BuildConnectionString(gqbd.Qdrant, gqbd.DBConfig{Host: "localhost", Port: 6333})
// Result: "http://localhost:6333" (https:// when SSLMode is not "disable")
```

Qdrant는 `api-key` 헤더로 인증하므로 `User`와 `Password`는 주소에 절대 포함되지 않습니다.
