package gqbd

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ESBuilder builds Elasticsearch query DSL request bodies for the search, count
// and delete-by-query APIs. Like QueryBuilder, it records the first error and
// returns it from Build.
//
// Full-text queries (Match, MatchPhrase, MultiMatch) go into the bool query's
// must clause so they score results; exact queries (Term, Terms, ranges,
// Exists, Prefix) go into filter, which does not score and can be cached.
// Values are always JSON-encoded in value position, so input such as a decoded
// JSON object cannot change the structure of the query.
//
// Example (go-elasticsearch):
//
//	index, body, err := gqbd.ESSearch("products").Match("title", "headphones").Term("status", "active").Size(10).Build()
//	res, err := es.Search(es.Search.WithIndex(index), es.Search.WithBody(bytes.NewReader(body)))
type ESBuilder struct {
	op      string
	index   string
	clauses []esClause
	ranges  map[string]map[string]interface{} // field -> gt/gte/lt/lte
	sort    []interface{}
	source  []string
	size    int // -1 when unset, because size 0 (aggregations only) is meaningful
	from    int
	aggs    map[string]interface{}
	err     error
}

type esClause struct {
	occur      string // must, filter or must_not
	query      map[string]interface{}
	rangeField string // set instead of query for range clauses, which are built from ranges
}

// ESSearch creates a builder for the search API. index may name several
// indices or a pattern, e.g. "logs-*" or "a,b".
func ESSearch(index string) *ESBuilder { return newESBuilder("search", index) }

// ESCount creates a builder for the count API, which accepts only a query.
func ESCount(index string) *ESBuilder { return newESBuilder("count", index) }

// ESDeleteByQuery creates a builder for the delete-by-query API. Build returns
// an error if no query was added; use MatchAll to delete every document.
func ESDeleteByQuery(index string) *ESBuilder { return newESBuilder("delete_by_query", index) }

// ESQuery creates a builder for a standalone query, for use with Or and Not.
func ESQuery() *ESBuilder { return &ESBuilder{op: "query", size: -1} }

func newESBuilder(op, index string) *ESBuilder {
	b := &ESBuilder{op: op, size: -1}
	// The index goes into the request path, so reject characters that could
	// change the path or are never valid in an index name.
	if index == "" || strings.ContainsAny(index, " \"\\/<>|?#\x00") {
		b.err = fmt.Errorf("invalid Elasticsearch index name %q", index)
		return b
	}
	b.index = index
	return b
}

func (b *ESBuilder) add(occur, field string, query map[string]interface{}) *ESBuilder {
	if b.err != nil {
		return b
	}
	if field == "" {
		b.err = fmt.Errorf("empty Elasticsearch field name")
		return b
	}
	b.clauses = append(b.clauses, esClause{occur: occur, query: query})
	return b
}

// Match adds a full-text {"match": {field: text}} query.
func (b *ESBuilder) Match(field, text string) *ESBuilder {
	return b.add("must", field, map[string]interface{}{"match": map[string]interface{}{field: text}})
}

// MatchPhrase adds a {"match_phrase": {field: text}} query.
func (b *ESBuilder) MatchPhrase(field, text string) *ESBuilder {
	return b.add("must", field, map[string]interface{}{"match_phrase": map[string]interface{}{field: text}})
}

// MultiMatch adds a {"multi_match": {"query": text, "fields": fields}} query.
// Fields may carry a boost, e.g. "title^2".
func (b *ESBuilder) MultiMatch(text string, fields ...string) *ESBuilder {
	if b.err != nil {
		return b
	}
	if len(fields) == 0 {
		b.err = fmt.Errorf("MultiMatch() requires at least one field")
		return b
	}
	for _, f := range fields {
		if f == "" {
			b.err = fmt.Errorf("empty Elasticsearch field name")
			return b
		}
	}
	return b.add("must", fields[0], map[string]interface{}{
		"multi_match": map[string]interface{}{"query": text, "fields": append([]string(nil), fields...)},
	})
}

// Term adds an exact {"term": {field: {"value": value}}} filter. The long form
// keeps an object value from being read as term options.
func (b *ESBuilder) Term(field string, value interface{}) *ESBuilder {
	return b.add("filter", field, map[string]interface{}{
		"term": map[string]interface{}{field: map[string]interface{}{"value": value}},
	})
}

// Terms adds a {"terms": {field: values}} filter. values must be a non-empty
// slice or array of any type.
func (b *ESBuilder) Terms(field string, values interface{}) *ESBuilder {
	if b.err != nil {
		return b
	}
	list, err := toInterfaceSlice(values)
	if err != nil {
		b.err = fmt.Errorf("terms on field %q: %w", field, err)
		return b
	}
	if len(list) == 0 {
		b.err = fmt.Errorf("Terms() requires at least one value")
		return b
	}
	return b.add("filter", field, map[string]interface{}{"terms": map[string]interface{}{field: list}})
}

// Exists adds an {"exists": {"field": field}} filter.
func (b *ESBuilder) Exists(field string) *ESBuilder {
	return b.add("filter", field, map[string]interface{}{"exists": map[string]interface{}{"field": field}})
}

// Prefix adds a {"prefix": {field: {"value": prefix}}} filter.
func (b *ESBuilder) Prefix(field, prefix string) *ESBuilder {
	return b.add("filter", field, map[string]interface{}{
		"prefix": map[string]interface{}{field: map[string]interface{}{"value": prefix}},
	})
}

// Gt adds field > value to the field's range filter.
func (b *ESBuilder) Gt(field string, value interface{}) *ESBuilder {
	return b.addRange(field, "gt", value)
}

// Gte adds field >= value to the field's range filter.
func (b *ESBuilder) Gte(field string, value interface{}) *ESBuilder {
	return b.addRange(field, "gte", value)
}

// Lt adds field < value to the field's range filter.
func (b *ESBuilder) Lt(field string, value interface{}) *ESBuilder {
	return b.addRange(field, "lt", value)
}

// Lte adds field <= value to the field's range filter.
func (b *ESBuilder) Lte(field string, value interface{}) *ESBuilder {
	return b.addRange(field, "lte", value)
}

// addRange merges bounds on the same field into one range query, like
// {"range": {"price": {"gte": 10, "lt": 100}}}.
func (b *ESBuilder) addRange(field, op string, value interface{}) *ESBuilder {
	if b.err != nil {
		return b
	}
	if field == "" {
		b.err = fmt.Errorf("empty Elasticsearch field name")
		return b
	}
	ops, ok := b.ranges[field]
	if !ok {
		if b.ranges == nil {
			b.ranges = make(map[string]map[string]interface{})
		}
		ops = make(map[string]interface{}, 2)
		b.ranges[field] = ops
		b.clauses = append(b.clauses, esClause{occur: "filter", rangeField: field})
	}
	if _, dup := ops[op]; dup {
		b.err = fmt.Errorf("duplicate %s on field %q", op, field)
		return b
	}
	ops[op] = value
	return b
}

// MatchAll adds a {"match_all": {}} query, which matches every document.
func (b *ESBuilder) MatchAll() *ESBuilder {
	if b.err != nil {
		return b
	}
	b.clauses = append(b.clauses, esClause{occur: "must", query: map[string]interface{}{"match_all": map[string]interface{}{}}})
	return b
}

// Or requires at least one of the queries to match. Calling Or again ANDs the
// groups together. It is written as a nested bool query with
// "minimum_should_match": 1, because a should clause next to must or filter
// clauses would otherwise be optional.
func (b *ESBuilder) Or(queries ...*ESBuilder) *ESBuilder {
	if b.err != nil {
		return b
	}
	if len(queries) == 0 {
		b.err = fmt.Errorf("Or() requires at least one query")
		return b
	}
	should, err := esSubqueries("Or", queries)
	if err != nil {
		b.err = err
		return b
	}
	b.clauses = append(b.clauses, esClause{occur: "must", query: map[string]interface{}{
		"bool": map[string]interface{}{"should": should, "minimum_should_match": 1},
	}})
	return b
}

// Not excludes documents matching any of the queries (bool must_not).
func (b *ESBuilder) Not(queries ...*ESBuilder) *ESBuilder {
	if b.err != nil {
		return b
	}
	if len(queries) == 0 {
		b.err = fmt.Errorf("Not() requires at least one query")
		return b
	}
	list, err := esSubqueries("Not", queries)
	if err != nil {
		b.err = err
		return b
	}
	for _, q := range list {
		b.clauses = append(b.clauses, esClause{occur: "must_not", query: q.(map[string]interface{})})
	}
	return b
}

func esSubqueries(method string, queries []*ESBuilder) ([]interface{}, error) {
	list := make([]interface{}, len(queries))
	for i, q := range queries {
		if q == nil {
			return nil, fmt.Errorf("%s(): nil query", method)
		}
		if q.err != nil {
			return nil, q.err
		}
		list[i] = q.queryDSL()
	}
	return list, nil
}

// Raw adds a query without a dedicated method (e.g. geo_distance, nested,
// wildcard) to a bool clause: "must", "filter" or "must_not". The query is
// passed through unchanged.
func (b *ESBuilder) Raw(occur string, query map[string]interface{}) *ESBuilder {
	if b.err != nil {
		return b
	}
	switch occur {
	case "must", "filter", "must_not":
	default:
		b.err = fmt.Errorf("invalid Raw() clause %q (use must, filter or must_not; use Or for should)", occur)
		return b
	}
	if len(query) == 0 {
		b.err = fmt.Errorf("Raw(): empty query")
		return b
	}
	b.clauses = append(b.clauses, esClause{occur: occur, query: copyMap(query)})
	return b
}

func (b *ESBuilder) requireSearch(method string) bool {
	if b.err != nil {
		return false
	}
	if b.op != "search" {
		b.err = fmt.Errorf("%s() can only be used with ESSearch", method)
		return false
	}
	return true
}

// Sort adds a sort key; order is "asc" or "desc". Keys keep the order in which
// they are added.
func (b *ESBuilder) Sort(field, order string) *ESBuilder {
	if !b.requireSearch("Sort") {
		return b
	}
	if field == "" {
		b.err = fmt.Errorf("empty Elasticsearch field name")
		return b
	}
	order = strings.ToLower(order)
	if order != "asc" && order != "desc" {
		b.err = fmt.Errorf("sort order for %q must be asc or desc, got %q", field, order)
		return b
	}
	b.sort = append(b.sort, map[string]interface{}{field: map[string]interface{}{"order": order}})
	return b
}

// Source returns only the given fields of each document (_source filtering).
func (b *ESBuilder) Source(fields ...string) *ESBuilder {
	if b.requireSearch("Source") {
		b.source = append(b.source, fields...)
	}
	return b
}

// Size sets the maximum number of hits to return. Size(0) returns only
// aggregations and the total.
func (b *ESBuilder) Size(n int) *ESBuilder {
	if !b.requireSearch("Size") {
		return b
	}
	if n < 0 {
		b.err = fmt.Errorf("size must not be negative, got %d", n)
		return b
	}
	b.size = n
	return b
}

// From sets the number of hits to skip.
func (b *ESBuilder) From(n int) *ESBuilder {
	if !b.requireSearch("From") {
		return b
	}
	if n < 0 {
		b.err = fmt.Errorf("from must not be negative, got %d", n)
		return b
	}
	b.from = n
	return b
}

// Agg adds a named aggregation, passed through unchanged, e.g.
// Agg("by_brand", map[string]interface{}{"terms": map[string]interface{}{"field": "brand"}}).
func (b *ESBuilder) Agg(name string, agg map[string]interface{}) *ESBuilder {
	if !b.requireSearch("Agg") {
		return b
	}
	if name == "" {
		b.err = fmt.Errorf("empty aggregation name")
		return b
	}
	if _, dup := b.aggs[name]; dup {
		b.err = fmt.Errorf("duplicate aggregation %q", name)
		return b
	}
	if b.aggs == nil {
		b.aggs = make(map[string]interface{})
	}
	b.aggs[name] = copyMap(agg)
	return b
}

// Build returns the index and the JSON request body. It does not modify the
// builder, so it is safe to call more than once. A search or count without
// queries has no "query" key, which matches every document.
func (b *ESBuilder) Build() (string, []byte, error) {
	if b.err != nil {
		return "", nil, b.err
	}
	if b.op == "delete_by_query" && len(b.clauses) == 0 {
		return "", nil, fmt.Errorf("ESDeleteByQuery requires a query; use MatchAll() to delete every document")
	}
	body := make(map[string]interface{}, 6)
	if len(b.clauses) > 0 {
		body["query"] = b.queryDSL()
	}
	if len(b.sort) > 0 {
		body["sort"] = b.sort
	}
	if b.source != nil {
		body["_source"] = b.source
	}
	if b.size >= 0 {
		body["size"] = b.size
	}
	if b.from > 0 {
		body["from"] = b.from
	}
	if len(b.aggs) > 0 {
		body["aggs"] = b.aggs
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", nil, fmt.Errorf("encoding Elasticsearch query: %w", err)
	}
	return b.index, data, nil
}

// queryDSL returns the bool query for the builder's clauses, or match_all when
// there are none.
func (b *ESBuilder) queryDSL() map[string]interface{} {
	if len(b.clauses) == 0 {
		return map[string]interface{}{"match_all": map[string]interface{}{}}
	}
	boolQuery := make(map[string]interface{}, 3)
	for _, c := range b.clauses {
		q := c.query
		if c.rangeField != "" {
			q = map[string]interface{}{"range": map[string]interface{}{c.rangeField: copyMap(b.ranges[c.rangeField])}}
		}
		list, _ := boolQuery[c.occur].([]interface{})
		boolQuery[c.occur] = append(list, q)
	}
	return map[string]interface{}{"bool": boolQuery}
}
