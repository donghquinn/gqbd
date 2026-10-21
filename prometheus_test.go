package gqbd_test

import (
	"strings"
	"testing"

	"github.com/donghquinn/gqbd"
)

// Expected queries below were checked by running them on Prometheus 3.5 through the HTTP API.

func TestPromQueries(t *testing.T) {
	const m = "prometheus_http_requests_total"
	for _, tc := range []struct {
		name string
		b    *gqbd.PromBuilder
		want string
	}{
		{"selector", gqbd.PromSelector(m).Label("job", "prometheus"), `prometheus_http_requests_total{job="prometheus"}`},
		{"range", gqbd.PromSelector(m).Label("job", "prometheus").Range("1m"), `prometheus_http_requests_total{job="prometheus"}[1m]`},
		{"rate sum by",
			gqbd.PromSelector(m).LabelRegex("handler", "/api/v1/.*").LabelNe("code", "500").Rate("1m").Sum("handler", "code"),
			`sum by (handler, code) (rate(prometheus_http_requests_total{handler=~"/api/v1/.*", code!="500"}[1m]))`},
		{"offset without",
			gqbd.PromSelector(m).LabelNotRegex("handler", "/metrics").Offset("1s").Increase("1m").AggregateWithout("sum", "instance"),
			`sum without (instance) (increase(prometheus_http_requests_total{handler!~"/metrics"}[1m] offset 1s))`},
		{"histogram quantile",
			gqbd.PromSelector("prometheus_http_request_duration_seconds_bucket").Rate("1m").Sum("le").HistogramQuantile(0.95),
			`histogram_quantile(0.95, sum by (le) (rate(prometheus_http_request_duration_seconds_bucket[1m])))`},
		{"arithmetic",
			gqbd.PromSelector(m).Rate("1m").Sum().Div(gqbd.PromSelector("up").Count()).Mul(100),
			`(sum(rate(prometheus_http_requests_total[1m])) / count(up)) * 100`},
		{"nested binary rhs",
			gqbd.PromSelector(m).Rate("1m").Sub(gqbd.PromSelector(m).Irate("1m").Add(1)).Gte(-1000),
			`(rate(prometheus_http_requests_total[1m]) - (irate(prometheus_http_requests_total[1m]) + 1)) >= -1000`},
		{"topk", gqbd.PromSelector(m).TopK(3, "job"), `topk by (job) (3, prometheus_http_requests_total)`},
		{"funcs", gqbd.PromSelector("go_goroutines").RangeFunc("max_over_time", "1m").Func("abs").BottomK(1),
			`bottomk(1, abs(max_over_time(go_goroutines[1m])))`},
		{"no metric", gqbd.PromSelector("").Label("__name__", "up").Eq(1), `{__name__="up"} == 1`},
		{"other ops", gqbd.PromSelector("up").Avg().Min().Max().Ne(0).Gt(float32(0.5)).Lt(int64(2)).Lte(1.5),
			`(((max(min(avg(up))) != 0) > 0.5) < 2) <= 1.5`},
		{"colon metric and compound duration", gqbd.PromSelector("job:up:sum").Range("1h30m").Offset("500ms"), `job:up:sum[1h30m] offset 500ms`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, err := tc.b.Build()
			if err != nil {
				t.Fatal(err)
			}
			if q != tc.want {
				t.Errorf("got  %s\nwant %s", q, tc.want)
			}
		})
	}
}

func TestPromLabelValuesAreEscaped(t *testing.T) {
	q, err := gqbd.PromSelector("up").Label("job", `x"} or vector(1) or up{job="`).LabelRegex("instance", `prom\w+`).Build()
	if err != nil {
		t.Fatal(err)
	}
	if want := `up{job="x\"} or vector(1) or up{job=\"", instance=~"prom\\w+"}`; q != want {
		t.Errorf("got  %s\nwant %s", q, want)
	}
}

func TestPromBuildIsRepeatable(t *testing.T) {
	b := gqbd.PromSelector("up").Sum()
	first, _ := b.Build()
	second, _ := b.Build()
	if first != second || first != "sum(up)" {
		t.Errorf("got %q then %q", first, second)
	}
}

func TestPromErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		b    *gqbd.PromBuilder
		want string
	}{
		{"bad metric", gqbd.PromSelector("http-requests"), "invalid Prometheus metric name"},
		{"metric starts with digit", gqbd.PromSelector("1up"), "invalid Prometheus metric name"},
		{"empty selector", gqbd.PromSelector(""), "needs a metric name"},
		{"bad label", gqbd.PromSelector("up").Label("job name", "x"), "invalid Prometheus label name"},
		{"label with colon", gqbd.PromSelector("up").LabelNe("a:b", "x"), "invalid Prometheus label name"},
		{"label after wrap", gqbd.PromSelector("up").Sum().Label("job", "x"), "must be called before"},
		{"range after wrap", gqbd.PromSelector("up").Sum().Range("5m"), "must be called before"},
		{"bad duration", gqbd.PromSelector("up").Rate("5 minutes"), "invalid Prometheus duration"},
		{"empty duration", gqbd.PromSelector("up").Range(""), "invalid Prometheus duration"},
		{"duration without number", gqbd.PromSelector("up").Offset("m"), "invalid Prometheus duration"},
		{"range twice", gqbd.PromSelector("up").Range("5m").Rate("1m"), "already set"},
		{"bad func", gqbd.PromSelector("up").Func("abs) or vector(1"), "invalid Prometheus function name"},
		{"bad range func", gqbd.PromSelector("up").RangeFunc("", "5m"), "invalid Prometheus function name"},
		{"bad aggregation", gqbd.PromSelector("up").Aggregate("sum(", "job"), "invalid Prometheus aggregation"},
		{"bad by label", gqbd.PromSelector("up").Sum("job) or (vector(1)"), "invalid Prometheus label name"},
		{"without no labels", gqbd.PromSelector("up").AggregateWithout("sum"), "at least one label"},
		{"topk zero", gqbd.PromSelector("up").TopK(0), "must be positive"},
		{"phi range", gqbd.PromSelector("up").HistogramQuantile(1.5), "between 0 and 1"},
		{"string operand", gqbd.PromSelector("up").Mul("2) or vector(1"), "must be a number"},
		{"nil operand", gqbd.PromSelector("up").Div((*gqbd.PromBuilder)(nil)), "nil operand"},
		{"operand error", gqbd.PromSelector("up").Div(gqbd.PromSelector("")), "needs a metric name"},
		{"error kept", gqbd.PromSelector("-").Label("a", "b").Range("1m").Rate("1m").Sum().TopK(1).HistogramQuantile(0.5).AggregateWithout("sum", "a").Add(1), "invalid Prometheus metric name"},
		{"sql builder", nil, "not a SQL database"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.b == nil {
				_, _, err = gqbd.BuildSelect(gqbd.Prometheus, "up").Build()
			} else {
				_, err = tc.b.Build()
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestPromConnectionString(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  gqbd.DBConfig
		want string
	}{
		{"default", gqbd.DBConfig{Host: "localhost", Port: 9090}, "http://localhost:9090"},
		{"auth tls", gqbd.DBConfig{Host: "prom", Port: 443, User: "u", Password: "p@ss", SSLMode: "require"}, "https://u:p%40ss@prom:443"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := gqbd.BuildConnectionString(gqbd.Prometheus, tc.cfg); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
