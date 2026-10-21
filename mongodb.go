package gqbd

import (
	"fmt"
	"reflect"
	"strings"
)

// MongoQuery is the result of building a MongoDB find, update or delete. Every
// field can be passed to the driver as-is: an empty Filter matches all
// documents, an empty Sort or Projection is a no-op, and a zero Limit or Skip
// means none.
//
// Example (mongo-go-driver):
//
//	q, err := gqbd.MongoFind("users").Eq("status", "active").Sort("created_at", -1).Limit(10).Build()
//	cur, err := db.Collection(q.Collection).Find(ctx, q.Filter,
//	    options.Find().SetSort(q.Sort).SetProjection(q.Projection).SetLimit(q.Limit).SetSkip(q.Skip))
type MongoQuery struct {
	Collection string
	Filter     map[string]interface{}
	Update     map[string]interface{} // update operators, for MongoUpdate
	Projection map[string]interface{}
	Sort       D
	Limit      int64
	Skip       int64
	Upsert     bool
}

// MongoBuilder builds MongoDB filters, finds, updates and deletes. Like
// QueryBuilder, it records the first error and returns it from Build.
type MongoBuilder struct {
	op         string
	collection string
	fields     map[string]map[string]interface{} // field -> operator -> value
	raw        map[string]interface{}
	ors        [][]interface{}
	sort       D
	projection map[string]interface{}
	limit      int64
	skip       int64
	update     map[string]map[string]interface{} // operator -> field -> value
	updated    map[string]string                 // field -> operator, to detect conflicts
	upsert     bool
	err        error
}

// MongoFind creates a builder for find queries (Find, FindOne, CountDocuments).
func MongoFind(collection string) *MongoBuilder { return newMongoBuilder("find", collection) }

// MongoUpdate creates a builder for updates (UpdateOne, UpdateMany, FindOneAndUpdate).
func MongoUpdate(collection string) *MongoBuilder { return newMongoBuilder("update", collection) }

// MongoDelete creates a builder for deletes (DeleteOne, DeleteMany).
func MongoDelete(collection string) *MongoBuilder { return newMongoBuilder("delete", collection) }

// MongoFilter creates a builder for a standalone filter, for use with Or and
// MongoAggregate().Match.
func MongoFilter() *MongoBuilder { return &MongoBuilder{op: "filter"} }

func newMongoBuilder(op, collection string) *MongoBuilder {
	b := &MongoBuilder{op: op}
	if collection == "" || strings.ContainsAny(collection, "$\x00") {
		b.err = fmt.Errorf("invalid MongoDB collection name %q", collection)
		return b
	}
	b.collection = collection
	return b
}

// validMongoField rejects empty names and names starting with "$", so a field
// name taken from user input cannot inject a query operator.
func validMongoField(field string) error {
	if field == "" || field[0] == '$' || strings.IndexByte(field, 0) >= 0 {
		return fmt.Errorf("invalid MongoDB field name %q", field)
	}
	return nil
}

func (b *MongoBuilder) addOp(field, op string, value interface{}) *MongoBuilder {
	if b.err != nil {
		return b
	}
	if err := validMongoField(field); err != nil {
		b.err = err
		return b
	}
	if b.fields == nil {
		b.fields = make(map[string]map[string]interface{})
	}
	ops := b.fields[field]
	if ops == nil {
		ops = make(map[string]interface{}, 1)
		b.fields[field] = ops
	}
	if _, dup := ops[op]; dup {
		b.err = fmt.Errorf("duplicate %s on field %q", op, field)
		return b
	}
	ops[op] = value
	return b
}

// Eq matches documents where field equals value. A document-like value (such
// as a map from user input) is compared literally via $eq, never interpreted
// as operators.
func (b *MongoBuilder) Eq(field string, value interface{}) *MongoBuilder {
	return b.addOp(field, "$eq", value)
}

// Ne adds {field: {$ne: value}}.
func (b *MongoBuilder) Ne(field string, value interface{}) *MongoBuilder {
	return b.addOp(field, "$ne", value)
}

// Gt adds {field: {$gt: value}}.
func (b *MongoBuilder) Gt(field string, value interface{}) *MongoBuilder {
	return b.addOp(field, "$gt", value)
}

// Gte adds {field: {$gte: value}}.
func (b *MongoBuilder) Gte(field string, value interface{}) *MongoBuilder {
	return b.addOp(field, "$gte", value)
}

// Lt adds {field: {$lt: value}}.
func (b *MongoBuilder) Lt(field string, value interface{}) *MongoBuilder {
	return b.addOp(field, "$lt", value)
}

// Lte adds {field: {$lte: value}}.
func (b *MongoBuilder) Lte(field string, value interface{}) *MongoBuilder {
	return b.addOp(field, "$lte", value)
}

// In adds {field: {$in: values}}. values must be a slice or array of any type.
func (b *MongoBuilder) In(field string, values interface{}) *MongoBuilder {
	return b.addListOp(field, "$in", values)
}

// Nin adds {field: {$nin: values}}. values must be a slice or array of any type.
func (b *MongoBuilder) Nin(field string, values interface{}) *MongoBuilder {
	return b.addListOp(field, "$nin", values)
}

func (b *MongoBuilder) addListOp(field, op string, values interface{}) *MongoBuilder {
	if b.err != nil {
		return b
	}
	list, err := toInterfaceSlice(values)
	if err != nil {
		b.err = fmt.Errorf("%s on field %q: %w", op, field, err)
		return b
	}
	return b.addOp(field, op, list)
}

// Exists adds {field: {$exists: exists}}.
func (b *MongoBuilder) Exists(field string, exists bool) *MongoBuilder {
	return b.addOp(field, "$exists", exists)
}

// Regex adds {field: {$regex: pattern, $options: options}}; options may be empty.
func (b *MongoBuilder) Regex(field, pattern, options string) *MongoBuilder {
	b.addOp(field, "$regex", pattern)
	if options != "" {
		b.addOp(field, "$options", options)
	}
	return b
}

// Or adds {$or: [filter, ...]}. Calling Or again ANDs the groups together.
func (b *MongoBuilder) Or(filters ...*MongoBuilder) *MongoBuilder {
	if b.err != nil {
		return b
	}
	if len(filters) == 0 {
		b.err = fmt.Errorf("Or() requires at least one filter")
		return b
	}
	group := make([]interface{}, len(filters))
	for i, f := range filters {
		if f == nil {
			b.err = fmt.Errorf("Or(): nil filter")
			return b
		}
		if f.err != nil {
			b.err = f.err
			return b
		}
		group[i] = f.buildFilter()
	}
	b.ors = append(b.ors, group)
	return b
}

// Raw sets a top-level filter key directly, for operators without a dedicated
// method (e.g. "$text", "$expr"). The value is passed through unchanged.
func (b *MongoBuilder) Raw(key string, value interface{}) *MongoBuilder {
	if b.err != nil {
		return b
	}
	if key == "" || key == "$or" || key == "$and" {
		b.err = fmt.Errorf("invalid Raw() key %q (use Or for $or)", key)
		return b
	}
	if b.raw == nil {
		b.raw = make(map[string]interface{})
	}
	b.raw[key] = value
	return b
}

var mongoBuilderNames = map[string]string{"find": "MongoFind", "update": "MongoUpdate"}

func (b *MongoBuilder) requireOp(method, op string) bool {
	if b.err != nil {
		return false
	}
	if b.op != op {
		b.err = fmt.Errorf("%s() can only be used with %s", method, mongoBuilderNames[op])
		return false
	}
	return true
}

// Sort adds a sort key; direction is 1 (ascending) or -1 (descending). Keys
// keep the order in which they are added.
func (b *MongoBuilder) Sort(field string, direction int) *MongoBuilder {
	if !b.requireOp("Sort", "find") {
		return b
	}
	if err := validMongoField(field); err != nil {
		b.err = err
		return b
	}
	if direction != 1 && direction != -1 {
		b.err = fmt.Errorf("sort direction for %q must be 1 or -1, got %d", field, direction)
		return b
	}
	for _, e := range b.sort {
		if e.Key == field {
			b.err = fmt.Errorf("duplicate sort field %q", field)
			return b
		}
	}
	b.sort = append(b.sort, E{Key: field, Value: direction})
	return b
}

// Project includes only the given fields in results.
func (b *MongoBuilder) Project(fields ...string) *MongoBuilder {
	return b.setProjection("Project", fields, 1)
}

// Exclude removes the given fields from results.
func (b *MongoBuilder) Exclude(fields ...string) *MongoBuilder {
	return b.setProjection("Exclude", fields, 0)
}

func (b *MongoBuilder) setProjection(method string, fields []string, v int) *MongoBuilder {
	if !b.requireOp(method, "find") {
		return b
	}
	if b.projection == nil {
		b.projection = make(map[string]interface{}, len(fields))
	}
	for _, f := range fields {
		if err := validMongoField(f); err != nil {
			b.err = err
			return b
		}
		b.projection[f] = v
	}
	return b
}

// Limit sets the maximum number of documents to return.
func (b *MongoBuilder) Limit(n int64) *MongoBuilder {
	if !b.requireOp("Limit", "find") {
		return b
	}
	if n < 0 {
		b.err = fmt.Errorf("limit must not be negative, got %d", n)
		return b
	}
	b.limit = n
	return b
}

// Skip sets the number of documents to skip.
func (b *MongoBuilder) Skip(n int64) *MongoBuilder {
	if !b.requireOp("Skip", "find") {
		return b
	}
	if n < 0 {
		b.err = fmt.Errorf("skip must not be negative, got %d", n)
		return b
	}
	b.skip = n
	return b
}

func (b *MongoBuilder) addUpdate(method, op, field string, value interface{}) *MongoBuilder {
	if !b.requireOp(method, "update") {
		return b
	}
	if err := validMongoField(field); err != nil {
		b.err = err
		return b
	}
	if prev, ok := b.updated[field]; ok {
		b.err = fmt.Errorf("field %q is already updated by %s", field, prev)
		return b
	}
	if b.update == nil {
		b.update = make(map[string]map[string]interface{})
		b.updated = make(map[string]string)
	}
	if b.update[op] == nil {
		b.update[op] = make(map[string]interface{})
	}
	b.update[op][field] = value
	b.updated[field] = op
	return b
}

// Set adds {$set: data}.
func (b *MongoBuilder) Set(data map[string]interface{}) *MongoBuilder {
	for _, k := range sortedKeys(data) {
		b.addUpdate("Set", "$set", k, data[k])
	}
	return b
}

// Unset adds {$unset: {field: ""}} for each field.
func (b *MongoBuilder) Unset(fields ...string) *MongoBuilder {
	for _, f := range fields {
		b.addUpdate("Unset", "$unset", f, "")
	}
	return b
}

// Inc adds {$inc: {field: amount}}.
func (b *MongoBuilder) Inc(field string, amount interface{}) *MongoBuilder {
	return b.addUpdate("Inc", "$inc", field, amount)
}

// Push adds {$push: {field: value}}.
func (b *MongoBuilder) Push(field string, value interface{}) *MongoBuilder {
	return b.addUpdate("Push", "$push", field, value)
}

// AddToSet adds {$addToSet: {field: value}}.
func (b *MongoBuilder) AddToSet(field string, value interface{}) *MongoBuilder {
	return b.addUpdate("AddToSet", "$addToSet", field, value)
}

// Pull adds {$pull: {field: value}}.
func (b *MongoBuilder) Pull(field string, value interface{}) *MongoBuilder {
	return b.addUpdate("Pull", "$pull", field, value)
}

// Upsert marks the update as an upsert (pass q.Upsert to options.Update().SetUpsert).
func (b *MongoBuilder) Upsert() *MongoBuilder {
	if b.requireOp("Upsert", "update") {
		b.upsert = true
	}
	return b
}

// Build returns the query. It does not modify the builder, so it is safe to
// call more than once.
func (b *MongoBuilder) Build() (*MongoQuery, error) {
	if b.err != nil {
		return nil, b.err
	}
	for key := range b.raw {
		if _, clash := b.fields[key]; clash {
			return nil, fmt.Errorf("Raw() key %q is also used as a field", key)
		}
	}
	q := &MongoQuery{
		Collection: b.collection,
		Filter:     b.buildFilter(),
		Limit:      b.limit,
		Skip:       b.skip,
		Upsert:     b.upsert,
	}
	if b.op == "update" {
		if len(b.update) == 0 {
			return nil, fmt.Errorf("no update operators provided for MongoUpdate")
		}
		q.Update = make(map[string]interface{}, len(b.update))
		for op, fields := range b.update {
			q.Update[op] = copyMap(fields)
		}
	}
	if b.sort != nil {
		q.Sort = append(D(nil), b.sort...)
	}
	if b.projection != nil {
		q.Projection = copyMap(b.projection)
	}
	return q, nil
}

func (b *MongoBuilder) buildFilter() map[string]interface{} {
	f := make(map[string]interface{}, len(b.fields)+len(b.raw)+1)
	for field, ops := range b.fields {
		if v, ok := ops["$eq"]; ok && len(ops) == 1 && !isMongoDocument(v) {
			f[field] = v
		} else {
			f[field] = copyMap(ops)
		}
	}
	for k, v := range b.raw {
		f[k] = v
	}
	switch len(b.ors) {
	case 0:
	case 1:
		f["$or"] = b.ors[0]
	default:
		and := make([]interface{}, len(b.ors))
		for i, group := range b.ors {
			and[i] = map[string]interface{}{"$or": group}
		}
		f["$and"] = and
	}
	return f
}

// isMongoDocument reports whether v would be encoded as a document, and so must
// stay wrapped in $eq to avoid being read as query operators.
func isMongoDocument(v interface{}) bool {
	switch v.(type) {
	case nil:
		return false
	case D, bsonMarshaler:
		return true
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr && !rv.IsNil() {
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Map, reflect.Struct:
		return true
	case reflect.Slice:
		return isKeyValueSlice(rv.Type())
	}
	return false
}

func toInterfaceSlice(values interface{}) ([]interface{}, error) {
	if list, ok := values.([]interface{}); ok {
		return append([]interface{}{}, list...), nil
	}
	rv := reflect.ValueOf(values)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("values must be a slice or array, got %T", values)
	}
	list := make([]interface{}, rv.Len())
	for i := range list {
		list[i] = rv.Index(i).Interface()
	}
	return list, nil
}

func copyMap(m map[string]interface{}) map[string]interface{} {
	c := make(map[string]interface{}, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

// MongoAggregateBuilder builds an aggregation pipeline.
type MongoAggregateBuilder struct {
	collection string
	stages     []mongoStage
	err        error
}

type mongoStage struct {
	name string
	spec interface{}
}

// MongoAggregate creates a builder for an aggregation pipeline. Build returns
// the collection name and the pipeline, in the same shape as the SQL builders'
// (query, args, err):
//
//	coll, pipeline, err := gqbd.MongoAggregate("orders").
//	    Match(gqbd.MongoFilter().Eq("status", "paid")).
//	    Group("$customer_id", map[string]interface{}{"total": map[string]interface{}{"$sum": "$amount"}}).
//	    Sort("total", -1).
//	    Limit(10).
//	    Build()
//	cur, err := db.Collection(coll).Aggregate(ctx, pipeline)
func MongoAggregate(collection string) *MongoAggregateBuilder {
	a := &MongoAggregateBuilder{}
	if collection == "" || strings.ContainsAny(collection, "$\x00") {
		a.err = fmt.Errorf("invalid MongoDB collection name %q", collection)
		return a
	}
	a.collection = collection
	return a
}

// Stage appends a raw stage, e.g. Stage("$sample", map[string]interface{}{"size": 5}).
func (a *MongoAggregateBuilder) Stage(name string, spec interface{}) *MongoAggregateBuilder {
	if a.err != nil {
		return a
	}
	if len(name) < 2 || name[0] != '$' {
		a.err = fmt.Errorf("invalid pipeline stage name %q", name)
		return a
	}
	a.stages = append(a.stages, mongoStage{name, spec})
	return a
}

// Match appends a $match stage built from a filter.
func (a *MongoAggregateBuilder) Match(filter *MongoBuilder) *MongoAggregateBuilder {
	if a.err != nil {
		return a
	}
	if filter == nil {
		a.err = fmt.Errorf("Match(): nil filter")
		return a
	}
	if filter.err != nil {
		a.err = filter.err
		return a
	}
	return a.Stage("$match", filter.buildFilter())
}

// Group appends {$group: {_id: id, field: accumulator, ...}}.
func (a *MongoAggregateBuilder) Group(id interface{}, fields map[string]interface{}) *MongoAggregateBuilder {
	if a.err != nil {
		return a
	}
	spec := make(map[string]interface{}, len(fields)+1)
	for k, v := range fields {
		if k == "_id" {
			a.err = fmt.Errorf("Group(): pass _id as the id argument")
			return a
		}
		if err := validMongoField(k); err != nil {
			a.err = err
			return a
		}
		spec[k] = v
	}
	spec["_id"] = id
	return a.Stage("$group", spec)
}

// Sort appends a sort key; consecutive Sort calls share one $sort stage, in order.
func (a *MongoAggregateBuilder) Sort(field string, direction int) *MongoAggregateBuilder {
	if a.err != nil {
		return a
	}
	if err := validMongoField(field); err != nil {
		a.err = err
		return a
	}
	if direction != 1 && direction != -1 {
		a.err = fmt.Errorf("sort direction for %q must be 1 or -1, got %d", field, direction)
		return a
	}
	if n := len(a.stages); n > 0 && a.stages[n-1].name == "$sort" {
		d := a.stages[n-1].spec.(D)
		a.stages[n-1].spec = append(d[:len(d):len(d)], E{Key: field, Value: direction})
		return a
	}
	return a.Stage("$sort", D{{Key: field, Value: direction}})
}

// Project appends {$project: spec}.
func (a *MongoAggregateBuilder) Project(spec map[string]interface{}) *MongoAggregateBuilder {
	return a.Stage("$project", copyMap(spec))
}

// Limit appends {$limit: n}; n must be positive.
func (a *MongoAggregateBuilder) Limit(n int64) *MongoAggregateBuilder {
	if a.err == nil && n <= 0 {
		a.err = fmt.Errorf("$limit must be positive, got %d", n)
	}
	return a.Stage("$limit", n)
}

// Skip appends {$skip: n}.
func (a *MongoAggregateBuilder) Skip(n int64) *MongoAggregateBuilder {
	if a.err == nil && n < 0 {
		a.err = fmt.Errorf("$skip must not be negative, got %d", n)
	}
	return a.Stage("$skip", n)
}

// Unwind appends {$unwind: "$path"}; the leading "$" is optional.
func (a *MongoAggregateBuilder) Unwind(path string) *MongoAggregateBuilder {
	if a.err != nil {
		return a
	}
	if err := validMongoField(strings.TrimPrefix(path, "$")); err != nil {
		a.err = err
		return a
	}
	if path[0] != '$' {
		path = "$" + path
	}
	return a.Stage("$unwind", path)
}

// Lookup appends a $lookup stage joining another collection.
func (a *MongoAggregateBuilder) Lookup(from, localField, foreignField, as string) *MongoAggregateBuilder {
	if a.err != nil {
		return a
	}
	for _, f := range []string{localField, foreignField, as} {
		if err := validMongoField(f); err != nil {
			a.err = err
			return a
		}
	}
	if from == "" || strings.ContainsAny(from, "$\x00") {
		a.err = fmt.Errorf("invalid MongoDB collection name %q", from)
		return a
	}
	return a.Stage("$lookup", map[string]interface{}{
		"from": from, "localField": localField, "foreignField": foreignField, "as": as,
	})
}

// Count appends {$count: field}.
func (a *MongoAggregateBuilder) Count(field string) *MongoAggregateBuilder {
	if a.err != nil {
		return a
	}
	if err := validMongoField(field); err != nil {
		a.err = err
		return a
	}
	return a.Stage("$count", field)
}

// Build returns the collection name and the pipeline. It does not modify the
// builder, so it is safe to call more than once.
func (a *MongoAggregateBuilder) Build() (string, []interface{}, error) {
	if a.err != nil {
		return "", nil, a.err
	}
	pipeline := make([]interface{}, len(a.stages))
	for i, s := range a.stages {
		spec := s.spec
		if d, ok := spec.(D); ok {
			spec = append(D(nil), d...)
		}
		pipeline[i] = map[string]interface{}{s.name: spec}
	}
	return a.collection, pipeline, nil
}
