# Prometheus

[← Back to README](../README.md)

Prometheus gets its own builder (the SQL builders return an error for `gqbd.Prometheus`). `PromSelector(metric)` starts from a vector selector, and each function, aggregation or operator wraps the expression built so far. `Build()` returns `(query, err)`.

PromQL has no parameter binding, so gqbd keeps queries safe by construction: label values are always written as escaped string literals, and metric names, label names, function names and durations are validated. A label value such as `x"} or vector(1) or up{job="` stays a label value.

## Selectors

```go
q, err := gqbd.PromSelector("http_requests_total").
    Label("job", "api").                // job="api"
    LabelNe("code", "500").             // code!="500"
    LabelRegex("method", "GET|POST").   // method=~"GET|POST" (anchored by Prometheus)
    LabelNotRegex("handler", "/health"). // handler!~"/health"
    Build()
// http_requests_total{job="api", code!="500", method=~"GET|POST", handler!~"/health"}
```

`PromSelector("")` with label matchers gives `{__name__="up"}`-style selectors. `Range("5m")` makes a range vector (`metric[5m]`) and `Offset("1h")` shifts it back in time. Label matchers, `Range` and `Offset` must come before any function, aggregation or operator.

## Functions and aggregations

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

| Method | PromQL |
|--------|--------|
| `Rate(w)`, `Irate(w)`, `Increase(w)` | `rate(sel[w])`, ... |
| `RangeFunc(name, w)` | `name(sel[w])`, e.g. `avg_over_time` |
| `Func(name)` | `name(expr)`, e.g. `abs`, `ceil` |
| `Sum(by...)`, `Avg`, `Min`, `Max`, `Count` | `sum by (labels) (expr)`, or `sum(expr)` without labels |
| `Aggregate(op, by...)` | any aggregation, e.g. `stddev` |
| `AggregateWithout(op, labels...)` | `op without (labels) (expr)` |
| `TopK(k, by...)`, `BottomK(k, by...)` | `topk by (labels) (k, expr)` |
| `HistogramQuantile(phi)` | `histogram_quantile(phi, expr)` |

## Operators

`Add`, `Sub`, `Mul`, `Div` and the comparisons `Eq`, `Ne`, `Gt`, `Gte`, `Lt`, `Lte` take a number or another builder. Operands that are themselves operations are parenthesized, so the result reads the way the chain does:

```go
q, err := gqbd.PromSelector("http_requests_total").LabelRegex("code", "5..").Rate("5m").Sum().
    Div(gqbd.PromSelector("http_requests_total").Rate("5m").Sum()).
    Mul(100).
    Gt(1).
    Build()
// ((sum(rate(http_requests_total{code=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))) * 100) > 1
```

Vector matching modifiers (`on`, `ignoring`, `group_left`) and subqueries are not supported.

## Running queries

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
