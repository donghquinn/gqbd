# Performance

**English** | [한국어](./ko/performance.md)

GQBD sits on the request path of the services that use it, so building a query should cost far less than running it. This page explains how the builders keep allocations low and how to use them efficiently.

## How the SQL builder renders a query

1. **Identifiers are parsed once, written once.** Chained methods such as `BuildSelect`, `LeftJoin`, `GroupBy` and `OrderBy` validate each table and column name and store it in parsed form (prefix, name, alias). Nothing is quoted or concatenated at that point. `Build` writes the quoted name straight into the output buffer, so escaping `u.id` or `users AS u` allocates nothing.
2. **One pre-sized buffer.** `Build` estimates the final length from the stored parts and grows a single `strings.Builder` once. Placeholders (`?`, `$N`, `:N`) are numbered while writing, so conditions never need a second pass.
3. **Slices sized for common queries.** The first `Where`, `WhereIn` or `WhereBetween` reserves room for four conditions and four arguments, so typical queries grow these slices once instead of at 1, 2 and 4 elements.
4. **Rare clauses stay out of the way.** ClickHouse-only clauses (`Final`, `Prewhere`, `Setting`) live behind a pointer allocated on first use, which keeps every other builder in a smaller allocation size class.
5. **Clone shares instead of copying.** Builders only ever append to their slices, so `Clone` shares the original's backing arrays, clipped to their current length. The first append on either side copies, so neither sees the other's additions.
6. **Build does not modify the builder.** It can be called again, and from several goroutines once building is finished.

Quoting at `Build` time has a cost: rendering a finished builder again does the quoting again, which makes repeated `Build` calls on one builder somewhat slower than in v1.5.0 (see the README benchmarks). Building a query once, the common case, allocates much less.

## How the other builders render

- **Elasticsearch and Qdrant** write the JSON body directly in sorted key order, instead of building nested maps and calling `json.Marshal`. Strings, numbers, booleans, `[]string`, `[]float32`, `[]interface{}` and `map[string]interface{}` are encoded in place with `encoding/json`'s escaping and float formatting. Any other value type, NaN or infinity, and strings with invalid UTF-8 are passed to `json.Marshal`, so the output and errors match it. A fuzz test (`FuzzAppendJSON`) keeps the two in step.
- **MongoDB** keeps filter conditions and update operators in slices and builds the returned maps once, in `Build`. `gqbd.D` encodes `map[string]interface{}` and `[]interface{}` without reflection.
- **Prometheus** stores label matchers as structs and quotes them while writing the selector into one buffer; numbers use `strconv`, not `fmt`.
- **Redis** builds each command's argument slice once, with room for options such as `TTL` and `NX`.
- **Connection strings** are written into one pre-sized buffer, with credentials percent-encoded in place: one allocation per call.

## Using the builders efficiently

- **Reuse a base query with `Clone`.** Build the shared part once and clone it per request rather than rebuilding it:

  ```go
  base := gqbd.BuildSelect(gqbd.PostgreSQL, "orders", "id", "total").Where("tenant_id = ?", tenantID)

  open := base.Clone().Where("status = ?", "open").Limit(20)
  query, args, err := open.Build()
  ```

- **Call `Build` again for a builder that does not change.** Rendering an already built builder costs two allocations: the query string and the argument slice.
- **Prefer `WhereIn` over hand-written `IN (?, ?, ...)`.** It writes the placeholders in one allocation and keeps arguments in query order.
- **Pass values, never formatted SQL.** Besides being injection safe, bound arguments skip `fmt` formatting entirely.

## Benchmarks

Run them with:

```bash
go test -run='^$' -bench=. -benchmem ./...
```

`benchmark_test.go` covers the SQL builders and `benchmark_nosql_test.go` covers Redis, MongoDB, Elasticsearch, Prometheus, Qdrant and connection strings. The results for the current release are in the [README](../README.md#benchmarks).

To check a change, compare runs with [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat):

```bash
go test -run='^$' -bench=. -benchmem -count=6 . > before.txt
# apply the change
go test -run='^$' -bench=. -benchmem -count=6 . > after.txt
benchstat before.txt after.txt
```

Use `-count=6` or more, and don't run anything else CPU-heavy at the same time: `benchstat` needs several quiet samples to tell a real change from noise.
