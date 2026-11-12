// Package gqbd builds queries without connecting to a database: parameterized
// SQL for PostgreSQL, MySQL, MariaDB, SQLite, ClickHouse and Oracle, plus
// commands and request bodies for Redis, MongoDB, Elasticsearch, Prometheus and
// Qdrant. Run the results with the driver you already use.
//
// SQL builders collect the first error and return it from Build, so a chain
// needs a single error check:
//
//	query, args, err := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id", "name").
//		Where("age > ?", 18).
//		OrderBy("created_at", "DESC", nil).
//		Limit(10).
//		Build()
//	// query: SELECT "id", "name" FROM "users" WHERE age > $1 ORDER BY "created_at" DESC LIMIT $2
//	// args:  [18 10]
//
// Identifiers are quoted for the target dialect, values are always bound as
// arguments, and ? placeholders are numbered at Build time ($1 for PostgreSQL,
// :1 for Oracle).
package gqbd
