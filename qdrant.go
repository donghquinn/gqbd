package gqbd

import (
	"encoding/json"
	"fmt"
)

// QdrantBuilder builds request bodies for Qdrant's REST API: the query API
// (vector search, or listing points when no vector is given), count and delete.
// Like QueryBuilder, it records the first error and returns it from Build.
//
// Match values must be strings, integers or bools, and other values are always
// JSON-encoded in value position, so input such as a decoded JSON object cannot
// change the structure of the filter.
//
// Example (net/http):
//
//	path, body, err := gqbd.QdrantQuery("docs").Nearest(vec).Match("lang", "en").Limit(5).WithPayload().Build()
//	req, _ := http.NewRequest("POST", addr+path, bytes.NewReader(body))
//	req.Header.Set("Content-Type", "application/json")
//	req.Header.Set("api-key", apiKey)
type QdrantBuilder struct {
	op         string
	collection string
	must       []interface{}
	mustNot    []interface{}
	ranges     map[string]map[string]interface{} // key -> gt/gte/lt/lte
	query      interface{}
	using      string
	limit      int
	offset     int
	payload    interface{} // true or a list of fields
	withVector bool
	threshold  *float64
	exact      bool
	err        error
}

// QdrantQuery creates a builder for POST /collections/{collection}/points/query.
// Without Nearest or NearestPoint it lists points that match the filter.
func QdrantQuery(collection string) *QdrantBuilder { return newQdrantBuilder("query", collection) }

// QdrantCount creates a builder for POST /collections/{collection}/points/count.
func QdrantCount(collection string) *QdrantBuilder { return newQdrantBuilder("count", collection) }

// QdrantDelete creates a builder for POST /collections/{collection}/points/delete,
// deleting the points that match the filter. Build returns an error if no
// condition was added.
func QdrantDelete(collection string) *QdrantBuilder { return newQdrantBuilder("delete", collection) }

// QdrantFilter creates a builder for a standalone filter, for use with Or and Not.
func QdrantFilter() *QdrantBuilder { return &QdrantBuilder{op: "filter"} }

func newQdrantBuilder(op, collection string) *QdrantBuilder {
	b := &QdrantBuilder{op: op}
	if collection == "" {
		b.err = fmt.Errorf("empty Qdrant collection name")
		return b
	}
	b.collection = collection
	return b
}

func (b *QdrantBuilder) addCondition(key string, cond map[string]interface{}) *QdrantBuilder {
	if b.err != nil {
		return b
	}
	if key == "" {
		b.err = fmt.Errorf("empty Qdrant payload key")
		return b
	}
	cond["key"] = key
	b.must = append(b.must, cond)
	return b
}

// isQdrantMatchValue reports whether v is a type Qdrant can match exactly:
// a string, an integer or a bool.
func isQdrantMatchValue(v interface{}) bool {
	switch v.(type) {
	case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	}
	return false
}

// Match adds {"key": key, "match": {"value": value}}. value must be a string,
// an integer or a bool.
func (b *QdrantBuilder) Match(key string, value interface{}) *QdrantBuilder {
	if b.err == nil && !isQdrantMatchValue(value) {
		b.err = fmt.Errorf("match on key %q: value must be a string, integer or bool, got %T", key, value)
		return b
	}
	return b.addCondition(key, map[string]interface{}{"match": map[string]interface{}{"value": value}})
}

// MatchAny matches points whose key equals any of values (Qdrant's IN).
func (b *QdrantBuilder) MatchAny(key string, values interface{}) *QdrantBuilder {
	return b.matchList("MatchAny", "any", key, values)
}

// MatchExcept matches points whose key equals none of values (Qdrant's NOT IN).
func (b *QdrantBuilder) MatchExcept(key string, values interface{}) *QdrantBuilder {
	return b.matchList("MatchExcept", "except", key, values)
}

func (b *QdrantBuilder) matchList(method, kind, key string, values interface{}) *QdrantBuilder {
	if b.err != nil {
		return b
	}
	list, err := toInterfaceSlice(values)
	if err != nil {
		b.err = fmt.Errorf("%s on key %q: %w", method, key, err)
		return b
	}
	if len(list) == 0 {
		b.err = fmt.Errorf("%s() requires at least one value", method)
		return b
	}
	for _, v := range list {
		if !isQdrantMatchValue(v) {
			b.err = fmt.Errorf("%s on key %q: values must be strings, integers or bools, got %T", method, key, v)
			return b
		}
	}
	return b.addCondition(key, map[string]interface{}{"match": map[string]interface{}{kind: list}})
}

// MatchText adds a full-text match, which needs a full-text index on key.
func (b *QdrantBuilder) MatchText(key, text string) *QdrantBuilder {
	return b.addCondition(key, map[string]interface{}{"match": map[string]interface{}{"text": text}})
}

// IsEmpty matches points where key is missing, null or an empty array.
func (b *QdrantBuilder) IsEmpty(key string) *QdrantBuilder { return b.keyCondition("is_empty", key) }

// IsNull matches points where key is null.
func (b *QdrantBuilder) IsNull(key string) *QdrantBuilder { return b.keyCondition("is_null", key) }

func (b *QdrantBuilder) keyCondition(kind, key string) *QdrantBuilder {
	if b.err != nil {
		return b
	}
	if key == "" {
		b.err = fmt.Errorf("empty Qdrant payload key")
		return b
	}
	b.must = append(b.must, map[string]interface{}{kind: map[string]interface{}{"key": key}})
	return b
}

// HasID matches the points with the given IDs (unsigned integers or UUID strings).
func (b *QdrantBuilder) HasID(ids ...interface{}) *QdrantBuilder {
	if b.err != nil {
		return b
	}
	if len(ids) == 0 {
		b.err = fmt.Errorf("HasID() requires at least one ID")
		return b
	}
	for _, id := range ids {
		if err := checkQdrantID(id); err != nil {
			b.err = err
			return b
		}
	}
	b.must = append(b.must, map[string]interface{}{"has_id": append([]interface{}(nil), ids...)})
	return b
}

func checkQdrantID(id interface{}) error {
	switch x := id.(type) {
	case string:
		if x != "" {
			return nil
		}
	case int:
		if x >= 0 {
			return nil
		}
	case int64:
		if x >= 0 {
			return nil
		}
	case int32:
		if x >= 0 {
			return nil
		}
	case uint, uint8, uint16, uint32, uint64:
		return nil
	}
	return fmt.Errorf("invalid Qdrant point ID %v (%T): use an unsigned integer or a UUID string", id, id)
}

// Gt adds key > value to the key's range condition.
func (b *QdrantBuilder) Gt(key string, value interface{}) *QdrantBuilder {
	return b.addRange(key, "gt", value)
}

// Gte adds key >= value to the key's range condition.
func (b *QdrantBuilder) Gte(key string, value interface{}) *QdrantBuilder {
	return b.addRange(key, "gte", value)
}

// Lt adds key < value to the key's range condition.
func (b *QdrantBuilder) Lt(key string, value interface{}) *QdrantBuilder {
	return b.addRange(key, "lt", value)
}

// Lte adds key <= value to the key's range condition.
func (b *QdrantBuilder) Lte(key string, value interface{}) *QdrantBuilder {
	return b.addRange(key, "lte", value)
}

// addRange merges bounds on the same key into one condition, like
// {"key": "price", "range": {"gte": 10, "lt": 100}}. Bounds may be numbers or
// RFC 3339 datetimes.
func (b *QdrantBuilder) addRange(key, op string, value interface{}) *QdrantBuilder {
	if b.err != nil {
		return b
	}
	if key == "" {
		b.err = fmt.Errorf("empty Qdrant payload key")
		return b
	}
	ops, ok := b.ranges[key]
	if !ok {
		if b.ranges == nil {
			b.ranges = make(map[string]map[string]interface{})
		}
		ops = make(map[string]interface{}, 2)
		b.ranges[key] = ops
		// A placeholder in must keeps conditions in the order they were added.
		b.must = append(b.must, qdrantRange(key))
	}
	if _, dup := ops[op]; dup {
		b.err = fmt.Errorf("duplicate %s on key %q", op, key)
		return b
	}
	ops[op] = value
	return b
}

type qdrantRange string

// Or requires at least one of the filters to match. Calling Or again ANDs the
// groups together.
func (b *QdrantBuilder) Or(filters ...*QdrantBuilder) *QdrantBuilder {
	if b.err != nil {
		return b
	}
	list, err := qdrantSubfilters("Or", filters)
	if err != nil {
		b.err = err
		return b
	}
	b.must = append(b.must, map[string]interface{}{"should": list})
	return b
}

// Not excludes points matching any of the filters.
func (b *QdrantBuilder) Not(filters ...*QdrantBuilder) *QdrantBuilder {
	if b.err != nil {
		return b
	}
	list, err := qdrantSubfilters("Not", filters)
	if err != nil {
		b.err = err
		return b
	}
	b.mustNot = append(b.mustNot, list...)
	return b
}

func qdrantSubfilters(method string, filters []*QdrantBuilder) ([]interface{}, error) {
	if len(filters) == 0 {
		return nil, fmt.Errorf("%s() requires at least one filter", method)
	}
	list := make([]interface{}, len(filters))
	for i, f := range filters {
		if f == nil {
			return nil, fmt.Errorf("%s(): nil filter", method)
		}
		if f.err != nil {
			return nil, f.err
		}
		if !f.hasFilter() {
			return nil, fmt.Errorf("%s(): empty filter", method)
		}
		list[i] = f.filter()
	}
	return list, nil
}

func (b *QdrantBuilder) requireQuery(method string) bool {
	if b.err != nil {
		return false
	}
	if b.op != "query" {
		b.err = fmt.Errorf("%s() can only be used with QdrantQuery", method)
		return false
	}
	return true
}

// Nearest searches for the points closest to vector.
func (b *QdrantBuilder) Nearest(vector []float32) *QdrantBuilder {
	if !b.requireQuery("Nearest") {
		return b
	}
	if len(vector) == 0 {
		b.err = fmt.Errorf("Nearest() requires a non-empty vector")
		return b
	}
	return b.setQuery("Nearest", append([]float32(nil), vector...))
}

// NearestPoint searches for the points closest to an existing point's vector.
func (b *QdrantBuilder) NearestPoint(id interface{}) *QdrantBuilder {
	if !b.requireQuery("NearestPoint") {
		return b
	}
	if err := checkQdrantID(id); err != nil {
		b.err = err
		return b
	}
	return b.setQuery("NearestPoint", id)
}

func (b *QdrantBuilder) setQuery(method string, q interface{}) *QdrantBuilder {
	if b.query != nil {
		b.err = fmt.Errorf("%s(): the query vector is already set", method)
		return b
	}
	b.query = q
	return b
}

// Using selects a named vector of the collection to search.
func (b *QdrantBuilder) Using(name string) *QdrantBuilder {
	if !b.requireQuery("Using") {
		return b
	}
	if name == "" {
		b.err = fmt.Errorf("Using(): empty vector name")
		return b
	}
	b.using = name
	return b
}

// Limit sets the maximum number of points to return (Qdrant's default is 10).
func (b *QdrantBuilder) Limit(n int) *QdrantBuilder {
	if !b.requireQuery("Limit") {
		return b
	}
	if n <= 0 {
		b.err = fmt.Errorf("limit must be positive, got %d", n)
		return b
	}
	b.limit = n
	return b
}

// Offset sets the number of points to skip.
func (b *QdrantBuilder) Offset(n int) *QdrantBuilder {
	if !b.requireQuery("Offset") {
		return b
	}
	if n < 0 {
		b.err = fmt.Errorf("offset must not be negative, got %d", n)
		return b
	}
	b.offset = n
	return b
}

// WithPayload returns the payload of each point: all of it, or only the given keys.
func (b *QdrantBuilder) WithPayload(keys ...string) *QdrantBuilder {
	if !b.requireQuery("WithPayload") {
		return b
	}
	if len(keys) == 0 {
		b.payload = true
	} else {
		b.payload = append([]string(nil), keys...)
	}
	return b
}

// WithVector returns the vector of each point.
func (b *QdrantBuilder) WithVector() *QdrantBuilder {
	if b.requireQuery("WithVector") {
		b.withVector = true
	}
	return b
}

// ScoreThreshold drops results scoring worse than score. Whether that means
// lower or higher depends on the collection's distance metric.
func (b *QdrantBuilder) ScoreThreshold(score float64) *QdrantBuilder {
	if b.requireQuery("ScoreThreshold") {
		b.threshold = &score
	}
	return b
}

// Exact searches without the HNSW index, checking every point.
func (b *QdrantBuilder) Exact() *QdrantBuilder {
	if b.requireQuery("Exact") {
		b.exact = true
	}
	return b
}

func (b *QdrantBuilder) hasFilter() bool { return len(b.must) > 0 || len(b.mustNot) > 0 }

// filter returns {"must": [...], "must_not": [...]} for the builder's conditions.
func (b *QdrantBuilder) filter() map[string]interface{} {
	f := make(map[string]interface{}, 2)
	if len(b.must) > 0 {
		must := make([]interface{}, len(b.must))
		for i, c := range b.must {
			if key, ok := c.(qdrantRange); ok {
				c = map[string]interface{}{"key": string(key), "range": copyMap(b.ranges[string(key)])}
			}
			must[i] = c
		}
		f["must"] = must
	}
	if len(b.mustNot) > 0 {
		f["must_not"] = append([]interface{}(nil), b.mustNot...)
	}
	return f
}

// Build returns the REST path (e.g. /collections/docs/points/query, with the
// collection name percent-encoded) and the JSON request body. It does not
// modify the builder, so it is safe to call more than once.
func (b *QdrantBuilder) Build() (string, []byte, error) {
	if b.err != nil {
		return "", nil, b.err
	}
	if b.op == "filter" {
		return "", nil, fmt.Errorf("QdrantFilter() is only for Or and Not; use QdrantQuery, QdrantCount or QdrantDelete")
	}
	if b.op == "delete" && !b.hasFilter() {
		return "", nil, fmt.Errorf("QdrantDelete requires a filter condition")
	}
	body := make(map[string]interface{}, 8)
	if b.hasFilter() {
		body["filter"] = b.filter()
	}
	if b.query != nil {
		body["query"] = b.query
	}
	if b.using != "" {
		body["using"] = b.using
	}
	if b.limit > 0 {
		body["limit"] = b.limit
	}
	if b.offset > 0 {
		body["offset"] = b.offset
	}
	if b.payload != nil {
		body["with_payload"] = b.payload
	}
	if b.withVector {
		body["with_vector"] = true
	}
	if b.threshold != nil {
		body["score_threshold"] = *b.threshold
	}
	if b.exact {
		body["params"] = map[string]interface{}{"exact": true}
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", nil, fmt.Errorf("encoding Qdrant request: %w", err)
	}
	return "/collections/" + percentEncode(b.collection) + "/points/" + b.op, data, nil
}
