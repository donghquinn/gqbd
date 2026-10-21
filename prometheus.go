package gqbd

import (
	"fmt"
	"strconv"
	"strings"
)

// PromBuilder builds PromQL expressions. PromQL has no parameter binding, so
// label values are always written as escaped string literals, and metric
// names, label names, function names and durations are validated. Like
// QueryBuilder, it records the first error and returns it from Build.
//
// Label matchers, Range and Offset apply to the selector and must come before
// functions, aggregations and operators, which each wrap the expression built
// so far.
//
// Example (client_golang):
//
//	q, err := gqbd.PromSelector("http_requests_total").Label("job", "api").Rate("5m").Sum("status").Build()
//	// sum by (status) (rate(http_requests_total{job="api"}[5m]))
//	result, warnings, err := v1.NewAPI(client).Query(ctx, q, time.Now())
type PromBuilder struct {
	metric   string
	matchers []string
	window   string // range selector duration, e.g. "5m"
	offset   string
	expr     string // set once the selector is wrapped; matchers can no longer be added
	binary   bool   // expr is a binary operation, so it needs parentheses as an operand
	err      error
}

// PromSelector creates a builder starting from a vector selector. metric may be
// empty if at least one label matcher is added.
func PromSelector(metric string) *PromBuilder {
	b := &PromBuilder{}
	if metric != "" && !isPromName(metric, true) {
		b.err = fmt.Errorf("invalid Prometheus metric name %q", metric)
		return b
	}
	b.metric = metric
	return b
}

// isPromName reports whether s is a valid metric name ([a-zA-Z_:][a-zA-Z0-9_:]*)
// or, with colon false, a valid label or function name.
func isPromName(s string, colon bool) bool {
	if s == "" || s[0] >= '0' && s[0] <= '9' {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] == ':' && colon {
			continue
		}
		if !isPlainIdent(s[i : i+1]) {
			return false
		}
	}
	return true
}

// isPromDuration reports whether s is a PromQL duration such as "5m" or "1h30m".
func isPromDuration(s string) bool {
	if s == "" {
		return false
	}
	for s != "" {
		i := 0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == 0 {
			return false
		}
		s = s[i:]
		switch {
		case strings.HasPrefix(s, "ms"):
			s = s[2:]
		case s != "" && strings.IndexByte("smhdwy", s[0]) >= 0:
			s = s[1:]
		default:
			return false
		}
	}
	return true
}

func (b *PromBuilder) addMatcher(method, name, op, value string) *PromBuilder {
	if b.err != nil {
		return b
	}
	if b.expr != "" {
		b.err = fmt.Errorf("%s() must be called before functions, aggregations and operators", method)
		return b
	}
	if !isPromName(name, false) {
		b.err = fmt.Errorf("invalid Prometheus label name %q", name)
		return b
	}
	b.matchers = append(b.matchers, name+op+strconv.Quote(value))
	return b
}

// Label adds the matcher name="value".
func (b *PromBuilder) Label(name, value string) *PromBuilder {
	return b.addMatcher("Label", name, "=", value)
}

// LabelNe adds the matcher name!="value".
func (b *PromBuilder) LabelNe(name, value string) *PromBuilder {
	return b.addMatcher("LabelNe", name, "!=", value)
}

// LabelRegex adds the matcher name=~"pattern". The pattern is anchored by
// Prometheus, so "GET|POST" matches only those two values.
func (b *PromBuilder) LabelRegex(name, pattern string) *PromBuilder {
	return b.addMatcher("LabelRegex", name, "=~", pattern)
}

// LabelNotRegex adds the matcher name!~"pattern".
func (b *PromBuilder) LabelNotRegex(name, pattern string) *PromBuilder {
	return b.addMatcher("LabelNotRegex", name, "!~", pattern)
}

func (b *PromBuilder) setSelector(method string, dst *string, d string) *PromBuilder {
	if b.err != nil {
		return b
	}
	if b.expr != "" {
		b.err = fmt.Errorf("%s() must be called before functions, aggregations and operators", method)
		return b
	}
	if !isPromDuration(d) {
		b.err = fmt.Errorf("invalid Prometheus duration %q", d)
		return b
	}
	if *dst != "" {
		b.err = fmt.Errorf("%s() is already set", method)
		return b
	}
	*dst = d
	return b
}

// Range makes the selector a range vector, e.g. metric[5m]. Use it for range
// functions without a dedicated method; Rate, Irate, Increase and RangeFunc
// set the range themselves.
func (b *PromBuilder) Range(window string) *PromBuilder {
	return b.setSelector("Range", &b.window, window)
}

// Offset shifts the selector back in time, e.g. metric offset 1h.
func (b *PromBuilder) Offset(d string) *PromBuilder {
	return b.setSelector("Offset", &b.offset, d)
}

// selector renders the vector selector, e.g. metric{job="api"}[5m] offset 1h.
func (b *PromBuilder) selector() string {
	var s strings.Builder
	s.WriteString(b.metric)
	if len(b.matchers) > 0 || b.metric == "" {
		s.WriteByte('{')
		s.WriteString(strings.Join(b.matchers, ", "))
		s.WriteByte('}')
	}
	if b.window != "" {
		s.WriteByte('[')
		s.WriteString(b.window)
		s.WriteByte(']')
	}
	if b.offset != "" {
		s.WriteString(" offset ")
		s.WriteString(b.offset)
	}
	return s.String()
}

func (b *PromBuilder) current() string {
	if b.expr != "" {
		return b.expr
	}
	return b.selector()
}

// Rate wraps the selector in rate(selector[window]).
func (b *PromBuilder) Rate(window string) *PromBuilder { return b.RangeFunc("rate", window) }

// Irate wraps the selector in irate(selector[window]).
func (b *PromBuilder) Irate(window string) *PromBuilder { return b.RangeFunc("irate", window) }

// Increase wraps the selector in increase(selector[window]).
func (b *PromBuilder) Increase(window string) *PromBuilder { return b.RangeFunc("increase", window) }

// RangeFunc wraps the selector in a range-vector function such as
// avg_over_time or max_over_time: name(selector[window]).
func (b *PromBuilder) RangeFunc(name, window string) *PromBuilder {
	if b.Range(window); b.err != nil {
		return b
	}
	return b.Func(name)
}

// Func wraps the expression in a single-argument function, e.g. abs or
// ceil: name(expr).
func (b *PromBuilder) Func(name string) *PromBuilder {
	if b.err != nil {
		return b
	}
	if !isPromName(name, false) {
		b.err = fmt.Errorf("invalid Prometheus function name %q", name)
		return b
	}
	b.expr, b.binary = name+"("+b.current()+")", false
	return b
}

// Sum aggregates with sum, grouped by the given labels if any.
func (b *PromBuilder) Sum(by ...string) *PromBuilder { return b.Aggregate("sum", by...) }

// Avg aggregates with avg, grouped by the given labels if any.
func (b *PromBuilder) Avg(by ...string) *PromBuilder { return b.Aggregate("avg", by...) }

// Min aggregates with min, grouped by the given labels if any.
func (b *PromBuilder) Min(by ...string) *PromBuilder { return b.Aggregate("min", by...) }

// Max aggregates with max, grouped by the given labels if any.
func (b *PromBuilder) Max(by ...string) *PromBuilder { return b.Aggregate("max", by...) }

// Count aggregates with count, grouped by the given labels if any.
func (b *PromBuilder) Count(by ...string) *PromBuilder { return b.Aggregate("count", by...) }

// Aggregate wraps the expression in an aggregation: op by (labels) (expr), or
// op(expr) without labels.
func (b *PromBuilder) Aggregate(op string, by ...string) *PromBuilder {
	return b.aggregate(op, "by", by, "")
}

// AggregateWithout wraps the expression in op without (labels) (expr).
func (b *PromBuilder) AggregateWithout(op string, without ...string) *PromBuilder {
	if b.err == nil && len(without) == 0 {
		b.err = fmt.Errorf("AggregateWithout() requires at least one label")
		return b
	}
	return b.aggregate(op, "without", without, "")
}

// TopK keeps the k largest series, per group of the given labels if any.
func (b *PromBuilder) TopK(k int, by ...string) *PromBuilder { return b.kAggregate("topk", k, by) }

// BottomK keeps the k smallest series, per group of the given labels if any.
func (b *PromBuilder) BottomK(k int, by ...string) *PromBuilder {
	return b.kAggregate("bottomk", k, by)
}

func (b *PromBuilder) kAggregate(op string, k int, by []string) *PromBuilder {
	if b.err == nil && k <= 0 {
		b.err = fmt.Errorf("%s k must be positive, got %d", op, k)
		return b
	}
	return b.aggregate(op, "by", by, strconv.Itoa(k)+", ")
}

func (b *PromBuilder) aggregate(op, clause string, labels []string, param string) *PromBuilder {
	if b.err != nil {
		return b
	}
	if !isPromName(op, false) {
		b.err = fmt.Errorf("invalid Prometheus aggregation %q", op)
		return b
	}
	var s strings.Builder
	s.WriteString(op)
	if len(labels) > 0 {
		for _, l := range labels {
			if !isPromName(l, false) {
				b.err = fmt.Errorf("invalid Prometheus label name %q", l)
				return b
			}
		}
		s.WriteByte(' ')
		s.WriteString(clause)
		s.WriteString(" (")
		s.WriteString(strings.Join(labels, ", "))
		s.WriteString(") ")
	}
	s.WriteByte('(')
	s.WriteString(param)
	s.WriteString(b.current())
	s.WriteByte(')')
	b.expr, b.binary = s.String(), false
	return b
}

// HistogramQuantile wraps the expression in histogram_quantile(phi, expr). The
// expression should keep the le label, e.g. Rate("5m").Sum("le").
func (b *PromBuilder) HistogramQuantile(phi float64) *PromBuilder {
	if b.err != nil {
		return b
	}
	if phi < 0 || phi > 1 {
		b.err = fmt.Errorf("histogram_quantile phi must be between 0 and 1, got %v", phi)
		return b
	}
	b.expr, b.binary = "histogram_quantile("+strconv.FormatFloat(phi, 'g', -1, 64)+", "+b.current()+")", false
	return b
}

// Add writes expr + rhs. rhs is a number or another *PromBuilder.
func (b *PromBuilder) Add(rhs interface{}) *PromBuilder { return b.binaryOp("+", rhs) }

// Sub writes expr - rhs. rhs is a number or another *PromBuilder.
func (b *PromBuilder) Sub(rhs interface{}) *PromBuilder { return b.binaryOp("-", rhs) }

// Mul writes expr * rhs. rhs is a number or another *PromBuilder.
func (b *PromBuilder) Mul(rhs interface{}) *PromBuilder { return b.binaryOp("*", rhs) }

// Div writes expr / rhs. rhs is a number or another *PromBuilder.
func (b *PromBuilder) Div(rhs interface{}) *PromBuilder { return b.binaryOp("/", rhs) }

// Eq keeps series where expr == rhs. rhs is a number or another *PromBuilder.
func (b *PromBuilder) Eq(rhs interface{}) *PromBuilder { return b.binaryOp("==", rhs) }

// Ne keeps series where expr != rhs. rhs is a number or another *PromBuilder.
func (b *PromBuilder) Ne(rhs interface{}) *PromBuilder { return b.binaryOp("!=", rhs) }

// Gt keeps series where expr > rhs. rhs is a number or another *PromBuilder.
func (b *PromBuilder) Gt(rhs interface{}) *PromBuilder { return b.binaryOp(">", rhs) }

// Gte keeps series where expr >= rhs. rhs is a number or another *PromBuilder.
func (b *PromBuilder) Gte(rhs interface{}) *PromBuilder { return b.binaryOp(">=", rhs) }

// Lt keeps series where expr < rhs. rhs is a number or another *PromBuilder.
func (b *PromBuilder) Lt(rhs interface{}) *PromBuilder { return b.binaryOp("<", rhs) }

// Lte keeps series where expr <= rhs. rhs is a number or another *PromBuilder.
func (b *PromBuilder) Lte(rhs interface{}) *PromBuilder { return b.binaryOp("<=", rhs) }

// binaryOp writes lhs op rhs, parenthesizing operands that are themselves
// binary operations so the result does not depend on operator precedence.
func (b *PromBuilder) binaryOp(op string, rhs interface{}) *PromBuilder {
	if b.err != nil {
		return b
	}
	var right string
	switch x := rhs.(type) {
	case *PromBuilder:
		if x == nil {
			b.err = fmt.Errorf("nil operand for %s", op)
			return b
		}
		r, err := x.Build()
		if err != nil {
			b.err = err
			return b
		}
		if x.binary {
			r = "(" + r + ")"
		}
		right = r
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		right = fmt.Sprint(x)
	case float32:
		right = strconv.FormatFloat(float64(x), 'g', -1, 32)
	case float64:
		right = strconv.FormatFloat(x, 'g', -1, 64)
	default:
		b.err = fmt.Errorf("operand for %s must be a number or *PromBuilder, got %T", op, rhs)
		return b
	}
	left := b.current()
	if b.binary {
		left = "(" + left + ")"
	}
	b.expr, b.binary = left+" "+op+" "+right, true
	return b
}

// Build returns the PromQL expression. It does not modify the builder, so it
// is safe to call more than once.
func (b *PromBuilder) Build() (string, error) {
	if b.err != nil {
		return "", b.err
	}
	if b.metric == "" && len(b.matchers) == 0 {
		return "", fmt.Errorf("a Prometheus selector needs a metric name or at least one label matcher")
	}
	return b.current(), nil
}
