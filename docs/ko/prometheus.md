# Prometheus

[English](../prometheus.md) | **한국어**

[← README로 돌아가기](../../README.ko.md)

Prometheus는 별도의 빌더를 사용합니다(`gqbd.Prometheus`에 대해 SQL 빌더는 에러를 반환합니다). `PromSelector(metric)`는 벡터 셀렉터에서 시작하며, 각 함수, 집계, 연산자는 지금까지 만든 표현식을 감쌉니다. `Build()`는 `(query, err)`를 반환합니다.

PromQL에는 파라미터 바인딩이 없으므로 gqbd는 구조적으로 쿼리를 안전하게 만듭니다. 레이블 값은 항상 이스케이프된 문자열 리터럴로 작성되고, 메트릭 이름, 레이블 이름, 함수 이름, 기간은 검증됩니다. `x"} or vector(1) or up{job="` 같은 레이블 값도 레이블 값으로 남습니다.

## 셀렉터

```go
q, err := gqbd.PromSelector("http_requests_total").
    Label("job", "api").                // job="api"
    LabelNe("code", "500").             // code!="500"
    LabelRegex("method", "GET|POST").   // method=~"GET|POST" (Prometheus가 앵커 처리)
    LabelNotRegex("handler", "/health"). // handler!~"/health"
    Build()
// http_requests_total{job="api", code!="500", method=~"GET|POST", handler!~"/health"}
```

레이블 매처와 함께 `PromSelector("")`를 쓰면 `{__name__="up"}` 형태의 셀렉터가 됩니다. `Range("5m")`는 범위 벡터(`metric[5m]`)를 만들고, `Offset("1h")`는 시간을 과거로 이동시킵니다. 레이블 매처, `Range`, `Offset`은 반드시 함수, 집계, 연산자보다 먼저 와야 합니다.

## 함수와 집계

```go
q, err := gqbd.PromSelector("http_requests_total").
    Label("job", "api").
    Rate("5m").
    Sum("status").
    Build()
// sum by (status) (rate(http_requests_total{job="api"}[5m]))

q, err = gqbd.PromSelector("http_request_duration_seconds_bucket").
    Rate("5m").
    Sum("le").
    HistogramQuantile(0.95).
    Build()
// histogram_quantile(0.95, sum by (le) (rate(http_request_duration_seconds_bucket[5m])))
```

| 메서드 | PromQL |
|--------|--------|
| `Rate(w)`, `Irate(w)`, `Increase(w)` | `rate(sel[w])`, ... |
| `RangeFunc(name, w)` | `name(sel[w])`, 예: `avg_over_time` |
| `Func(name)` | `name(expr)`, 예: `abs`, `ceil` |
| `Sum(by...)`, `Avg`, `Min`, `Max`, `Count` | `sum by (labels) (expr)`, 레이블이 없으면 `sum(expr)` |
| `Aggregate(op, by...)` | 모든 집계, 예: `stddev` |
| `AggregateWithout(op, labels...)` | `op without (labels) (expr)` |
| `TopK(k, by...)`, `BottomK(k, by...)` | `topk by (labels) (k, expr)` |
| `HistogramQuantile(phi)` | `histogram_quantile(phi, expr)` |

## 연산자

`Add`, `Sub`, `Mul`, `Div`와 비교 연산 `Eq`, `Ne`, `Gt`, `Gte`, `Lt`, `Lte`는 숫자나 다른 빌더를 받습니다. 피연산자가 그 자체로 연산이면 괄호로 감싸므로, 결과는 체인과 같은 순서로 읽힙니다.

```go
q, err := gqbd.PromSelector("http_requests_total").LabelRegex("code", "5..").Rate("5m").Sum().
    Div(gqbd.PromSelector("http_requests_total").Rate("5m").Sum()).
    Mul(100).
    Gt(1).
    Build()
// ((sum(rate(http_requests_total{code=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))) * 100) > 1
```

벡터 매칭 수정자(`on`, `ignoring`, `group_left`)와 서브쿼리는 지원하지 않습니다.

## 쿼리 실행

```go
addr := gqbd.BuildConnectionString(gqbd.Prometheus, gqbd.DBConfig{Host: "localhost", Port: 9090})
// Result: "http://localhost:9090" (https:// when SSLMode is not "disable";
// User and Password are added to the URL and sent as basic auth)

import (
    "github.com/prometheus/client_golang/api"
    v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

client, err := api.NewClient(api.Config{Address: addr})
result, warnings, err := v1.NewAPI(client).Query(ctx, q, time.Now())
```
