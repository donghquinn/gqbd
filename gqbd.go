package gqbd

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// DBType represents the type of database.
type DBType string

const (
	PostgreSQL DBType = "postgres"
	MariaDB    DBType = "mariadb"
	Mysql      DBType = "mysql"
	SQLite     DBType = "sqlite"
	ClickHouse DBType = "clickhouse"
	Oracle     DBType = "oracle"
	// Redis is not a SQL database: use the Redis* command builders (see redis.go)
	// and BuildConnectionString. The SQL builders return an error for Redis.
	Redis DBType = "redis"
	// MongoDB is not a SQL database: use MongoFind, MongoUpdate, MongoDelete and
	// MongoAggregate (see mongodb.go). The SQL builders return an error for MongoDB.
	MongoDB DBType = "mongodb"
	// Elasticsearch is not a SQL database: use ESSearch, ESCount and
	// ESDeleteByQuery (see elasticsearch.go). The SQL builders return an error
	// for Elasticsearch.
	Elasticsearch DBType = "elasticsearch"
	// Prometheus is not a SQL database: use PromSelector (see prometheus.go). The
	// SQL builders return an error for Prometheus.
	Prometheus DBType = "prometheus"
	// Qdrant is not a SQL database: use QdrantQuery, QdrantCount and QdrantDelete
	// (see qdrant.go). The SQL builders return an error for Qdrant.
	Qdrant DBType = "qdrant"
)

// DBConfig holds database connection configuration
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string // For Redis: logical database number, e.g. "0". For MongoDB: default auth database
	SSLMode  string // For PostgreSQL: disable, require, verify-ca, verify-full. For Redis/MongoDB/ClickHouse/Oracle/Elasticsearch/Prometheus/Qdrant: any value other than "disable" enables TLS
	Charset  string // For MySQL/MariaDB: utf8, utf8mb4, etc.
	FilePath string // For SQLite: path to database file
}

// QueryBuilder is a high-performance SQL query builder.
// Designed for building dynamic queries safely without database connections.
//
// Conditions are stored with ? placeholders and numbered when Build is called,
// so Build can be called repeatedly and arguments always follow query order.
//
// Chained methods modify the builder they are called on. To reuse a base query,
// call Clone and add to the copy. A builder is not safe for concurrent use,
// but Build may be called from several goroutines once building is finished.
type QueryBuilder struct {
	dbType     DBType
	d          *dialect
	table      ident
	columns    []ident
	joins      []joinClause
	where      []string
	whereArgs  []interface{}
	groupBy    []ident
	having     []string
	havingArgs []interface{}
	orderCol   ident // no ORDER BY while orderCol.name is empty
	limit      int
	offset     int
	err        error
	data       map[string]interface{}
	returning  string
	ch         *clickhouseClauses // nil until a ClickHouse-only method is used
	op         sqlOp
	distinct   bool
	orderDesc  bool
}

// sqlOp is the statement a QueryBuilder renders.
type sqlOp uint8

const (
	opNone sqlOp = iota
	opSelect
	opInsert
	opUpdate
	opDelete
)

func (o sqlOp) String() string {
	switch o {
	case opSelect:
		return "SELECT"
	case opInsert:
		return "INSERT"
	case opUpdate:
		return "UPDATE"
	case opDelete:
		return "DELETE"
	}
	return ""
}

// clickhouseClauses holds the ClickHouse-only clauses. They live behind a
// pointer so builders for other databases stay smaller.
type clickhouseClauses struct {
	final        bool
	prewhere     []string
	prewhereArgs []interface{}
	settings     []string
}

// noClickhouseClauses is read in place of a nil QueryBuilder.ch; never modified.
var noClickhouseClauses = &clickhouseClauses{}

// clickhouse returns the ClickHouse clauses for reading.
func (qb *QueryBuilder) clickhouse() *clickhouseClauses {
	if qb.ch == nil {
		return noClickhouseClauses
	}
	return qb.ch
}

type joinClause struct {
	kind  string // "LEFT JOIN ", "INNER JOIN " or "RIGHT JOIN "
	table ident
	on    string
}

// selectAll is the column list of a SELECT without columns. Its capacity equals
// its length, so Aggregate copies it before appending.
var selectAll = []ident{{name: "*", verbatim: true}}

// BuildSelect creates a new SELECT query builder for the specified database type.
//
// Example:
//
//	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id", "name")
func BuildSelect(dbType DBType, table string, columns ...string) *QueryBuilder {
	qb := NewQueryBuilder(dbType, table, columns...)
	qb.op = opSelect
	return qb
}

// BuildInsert creates a new INSERT query builder for the specified database type.
//
// Example:
//
//	qb := gqbd.BuildInsert(gqbd.PostgreSQL, "users")
func BuildInsert(dbType DBType, table string) *QueryBuilder {
	qb := NewQueryBuilder(dbType, table)
	qb.op = opInsert
	return qb
}

// BuildUpdate creates a new UPDATE query builder for the specified database type.
//
// Example:
//
//	qb := gqbd.BuildUpdate(gqbd.PostgreSQL, "users")
func BuildUpdate(dbType DBType, table string) *QueryBuilder {
	qb := NewQueryBuilder(dbType, table)
	qb.op = opUpdate
	return qb
}

// BuildDelete creates a new DELETE query builder for the specified database type.
//
// Example:
//
//	qb := gqbd.BuildDelete(gqbd.PostgreSQL, "users")
func BuildDelete(dbType DBType, table string) *QueryBuilder {
	qb := NewQueryBuilder(dbType, table)
	qb.op = opDelete
	return qb
}

// NewQueryBuilder creates a new QueryBuilder instance with optimized defaults.
// Internal function used by Build* methods.
func NewQueryBuilder(dbType DBType, table string, columns ...string) *QueryBuilder {
	qb := &QueryBuilder{dbType: dbType, d: dialectFor(dbType)}
	switch dbType {
	case PostgreSQL, MariaDB, Mysql, SQLite, ClickHouse, Oracle:
	case Redis:
		qb.err = fmt.Errorf("Redis is not a SQL database; use the Redis command builders (RedisGet, RedisSet, RedisHSet, ...)")
		return qb
	case MongoDB:
		qb.err = fmt.Errorf("MongoDB is not a SQL database; use MongoFind, MongoUpdate, MongoDelete or MongoAggregate")
		return qb
	case Elasticsearch:
		qb.err = fmt.Errorf("Elasticsearch is not a SQL database; use ESSearch, ESCount or ESDeleteByQuery")
		return qb
	case Prometheus:
		qb.err = fmt.Errorf("Prometheus is not a SQL database; use PromSelector")
		return qb
	case Qdrant:
		qb.err = fmt.Errorf("Qdrant is not a SQL database; use QdrantQuery, QdrantCount or QdrantDelete")
		return qb
	default:
		qb.err = unknownDBTypeError(dbType)
		return qb
	}
	qb.table, qb.err = qb.d.parseTable(table)
	if qb.err != nil {
		return qb
	}
	if len(columns) == 0 {
		qb.columns = selectAll
		return qb
	}
	qb.columns = make([]ident, len(columns))
	for i, col := range columns {
		if qb.columns[i], qb.err = qb.d.parse(col); qb.err != nil {
			return qb
		}
	}
	return qb
}

// Distinct adds DISTINCT clause to SELECT queries.
func (qb *QueryBuilder) Distinct() *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.distinct = true
	return qb
}

// Aggregate adds an aggregate function to SELECT queries (COUNT, SUM, AVG, etc.).
// SQL injection safe with automatic identifier escaping.
func (qb *QueryBuilder) Aggregate(function, column string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if !isPlainIdent(function) {
		qb.err = fmt.Errorf("invalid aggregate function %q", function)
		return qb
	}
	col, err := qb.d.parse(column)
	if err != nil {
		qb.err = err
		return qb
	}
	var b strings.Builder
	b.Grow(len(function) + col.size() + 2)
	b.WriteString(function)
	b.WriteByte('(')
	qb.d.writeIdent(&b, col)
	b.WriteByte(')')
	qb.columns = append(qb.columns, ident{name: b.String(), verbatim: true})
	return qb
}

// LeftJoin adds a LEFT JOIN clause to the query.
// Table names are automatically escaped for security.
func (qb *QueryBuilder) LeftJoin(joinTable, onCondition string) *QueryBuilder {
	return qb.join("LEFT JOIN ", joinTable, onCondition)
}

/*
InnerJoin

@ joinTable: Table name to join
@ onCondition: Join condition
@ Return: *QueryBuilder with INNER JOIN added
*/
func (qb *QueryBuilder) InnerJoin(joinTable, onCondition string) *QueryBuilder {
	return qb.join("INNER JOIN ", joinTable, onCondition)
}

/*
RightJoin

@ joinTable: Table name to join
@ onCondition: Join condition
@ Return: *QueryBuilder with RIGHT JOIN added
*/
func (qb *QueryBuilder) RightJoin(joinTable, onCondition string) *QueryBuilder {
	return qb.join("RIGHT JOIN ", joinTable, onCondition)
}

func (qb *QueryBuilder) join(kind, joinTable, onCondition string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	table, err := qb.d.parseTable(joinTable)
	if err != nil {
		qb.err = err
		return qb
	}
	qb.joins = append(qb.joins, joinClause{kind: kind, table: table, on: onCondition})
	return qb
}

// Where adds a WHERE condition with parameter binding.
// Use ? as the placeholder; it is rendered as $N for PostgreSQL and :N for
// Oracle at Build time. Write ?? for a literal ? operator, such as
// PostgreSQL's jsonb "data ?? 'key'".
// SQL injection safe through proper parameter binding.
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.addWhere(condition, args)
	return qb
}

// addWhere appends a condition, sizing the slices for a few conditions up front
// so typical queries grow them once instead of at 1, 2 and 4 elements.
func (qb *QueryBuilder) addWhere(condition string, args []interface{}) {
	if qb.where == nil {
		qb.where = make([]string, 0, 4)
	}
	if qb.whereArgs == nil && len(args) > 0 {
		qb.whereArgs = make([]interface{}, 0, max(4, len(args)))
	}
	qb.where = append(qb.where, condition)
	qb.whereArgs = append(qb.whereArgs, args...)
}

/*
WhereIn

@ column: Column name for IN clause
@ values: Values for the IN clause
@ Return: *QueryBuilder with IN clause added
*/
func (qb *QueryBuilder) WhereIn(column string, values []interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	col, err := qb.d.parse(column)
	if err != nil {
		qb.err = err
		return qb
	}
	if len(values) == 0 {
		qb.err = fmt.Errorf("WhereIn() requires at least one value")
		return qb
	}
	var b strings.Builder
	b.Grow(col.size() + 6 + 3*len(values))
	qb.d.writeIdent(&b, col)
	b.WriteString(" IN (?")
	for i := 1; i < len(values); i++ {
		b.WriteString(", ?")
	}
	b.WriteByte(')')
	qb.addWhere(b.String(), values)
	return qb
}

/*
WhereBetween

@ column: Column name for BETWEEN clause
@ start: Start value
@ end: End value
@ Return: *QueryBuilder with BETWEEN clause added
*/
func (qb *QueryBuilder) WhereBetween(column string, start, end interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	col, err := qb.d.parse(column)
	if err != nil {
		qb.err = err
		return qb
	}
	var b strings.Builder
	b.Grow(col.size() + 16)
	qb.d.writeIdent(&b, col)
	b.WriteString(" BETWEEN ? AND ?")
	qb.addWhere(b.String(), []interface{}{start, end})
	return qb
}

/*
AddWhereIfNotEmpty

@ column: Column name
@ value: arguments
@ Return: *QueryBuilder
*/
func (qb *QueryBuilder) AddWhereIfNotEmpty(condition string, value interface{}) *QueryBuilder {
	if value == nil {
		return qb
	}

	switch v := value.(type) {
	case string:
		if v == "" {
			return qb
		}
	case *string:
		if v == nil || *v == "" {
			return qb
		}
		// 필요에 따라 다른 타입도 처리
	}

	return qb.Where(condition, value)
}

/*
GroupBy

@ columns: Columns for GROUP BY clause
@ Return: *QueryBuilder with GROUP BY clause added
*/
func (qb *QueryBuilder) GroupBy(columns ...string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	for _, name := range columns {
		col, err := qb.d.parse(name)
		if err != nil {
			qb.err = err
			return qb
		}
		qb.groupBy = append(qb.groupBy, col)
	}
	return qb
}

/*
Having

@ condition: HAVING clause condition with placeholders
@ args: Query parameters for HAVING clause
@ Return: *QueryBuilder with HAVING clause added
*/
func (qb *QueryBuilder) Having(condition string, args ...interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.having = append(qb.having, condition)
	qb.havingArgs = append(qb.havingArgs, args...)
	return qb
}

/*
OrderBy

@ column: Column name to order by
@ direction: Order direction ("ASC" or "DESC")
@ allowedColumns: Map of allowed columns for ordering
@ Return: *QueryBuilder with ORDER BY clause added
*/
func (qb *QueryBuilder) OrderBy(column, direction string, allowedColumns map[string]bool) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	direction = ValidateDirection(direction)
	if allowedColumns != nil {
		if _, ok := allowedColumns[column]; !ok {
			column = "id"
		}
	}
	col, err := qb.d.parse(column)
	if err != nil {
		qb.err = err
		return qb
	}
	qb.orderCol, qb.orderDesc = col, direction == "DESC"
	return qb
}

/*
Limit

@ limit: Maximum number of rows to return
@ Return: *QueryBuilder with LIMIT set
*/
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.limit = limit
	return qb
}

/*
Offset

@ offset: Number of rows to skip
@ Return: *QueryBuilder with OFFSET set
*/
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.offset = offset
	return qb
}

/*
Values

@ data: Map of column names to values for INSERT
@ Return: *QueryBuilder with data set for INSERT
*/
func (qb *QueryBuilder) Values(data map[string]interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if qb.op != opInsert {
		qb.err = fmt.Errorf("Values() can only be used with INSERT operation")
		return qb
	}
	qb.data = data
	return qb
}

/*
Set

@ data: Map of column names to values for UPDATE
@ Return: *QueryBuilder with data set for UPDATE
*/
func (qb *QueryBuilder) Set(data map[string]interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if qb.op != opUpdate {
		qb.err = fmt.Errorf("Set() can only be used with UPDATE operation")
		return qb
	}
	qb.data = data
	return qb
}

/*
Returning

@ clause: RETURNING clause string (for PostgreSQL and SQLite; ignored elsewhere)
@ Return: *QueryBuilder with RETURNING clause set
*/
func (qb *QueryBuilder) Returning(clause string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if qb.op != opInsert {
		qb.err = fmt.Errorf("Returning() can only be used with INSERT operation")
		return qb
	}
	qb.returning = clause
	return qb
}

func (qb *QueryBuilder) requireClickHouse(method string, ops ...sqlOp) bool {
	if qb.err != nil {
		return false
	}
	if !qb.d.chClauses {
		qb.err = fmt.Errorf("%s() is only supported for ClickHouse", method)
		return false
	}
	for _, op := range ops {
		if qb.op == op {
			if qb.ch == nil {
				qb.ch = &clickhouseClauses{}
			}
			return true
		}
	}
	qb.err = fmt.Errorf("%s() cannot be used with %s", method, qb.op)
	return false
}

// Final adds the FINAL modifier after the table name (ClickHouse only), which
// merges rows of ReplacingMergeTree and similar engines at query time.
func (qb *QueryBuilder) Final() *QueryBuilder {
	if qb.requireClickHouse("Final", opSelect) {
		qb.ch.final = true
	}
	return qb
}

// Prewhere adds a PREWHERE condition (ClickHouse only). Conditions are ANDed and
// bound like Where.
func (qb *QueryBuilder) Prewhere(condition string, args ...interface{}) *QueryBuilder {
	if qb.requireClickHouse("Prewhere", opSelect) {
		qb.ch.prewhere = append(qb.ch.prewhere, condition)
		qb.ch.prewhereArgs = append(qb.ch.prewhereArgs, args...)
	}
	return qb
}

// Setting adds a query-level setting, rendered as SETTINGS name = value
// (ClickHouse only). value must be a number, bool or string; it is written as an
// escaped literal because ClickHouse does not accept placeholders there.
//
// Example:
//
//	gqbd.BuildUpdate(gqbd.ClickHouse, "events").Set(data).Where("id = ?", 1).Setting("mutations_sync", 2)
func (qb *QueryBuilder) Setting(name string, value interface{}) *QueryBuilder {
	if !qb.requireClickHouse("Setting", opSelect, opInsert, opUpdate, opDelete) {
		return qb
	}
	if !isPlainIdent(name) {
		qb.err = fmt.Errorf("invalid setting name %q", name)
		return qb
	}
	lit, err := clickhouseLiteral(value)
	if err != nil {
		qb.err = fmt.Errorf("setting %q: %w", name, err)
		return qb
	}
	qb.ch.settings = append(qb.ch.settings, name+" = "+lit)
	return qb
}

func clickhouseLiteral(v interface{}) (string, error) {
	switch x := v.(type) {
	case bool:
		if x {
			return "1", nil
		}
		return "0", nil
	case string:
		x = strings.ReplaceAll(x, `\`, `\\`)
		return "'" + strings.ReplaceAll(x, "'", `\'`) + "'", nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(x), nil
	case float32:
		return strconv.FormatFloat(float64(x), 'g', -1, 32), nil
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64), nil
	}
	return "", fmt.Errorf("unsupported value type %T", v)
}

// Build generates the final SQL query string and parameter arguments.
// It does not modify the builder, so it is safe to call more than once.
// Returns: (query string, arguments slice, error)
func (qb *QueryBuilder) Build() (string, []interface{}, error) {
	if qb.err != nil {
		return "", nil, qb.err
	}
	switch qb.op {
	case opSelect:
		return qb.buildSelect()
	case opInsert:
		return qb.buildInsert()
	case opUpdate:
		return qb.buildUpdate()
	case opDelete:
		return qb.buildDelete()
	default:
		return "", nil, fmt.Errorf("unsupported operation: %s", qb.op)
	}
}

/*
EscapeIdentifier

@ dbType: Database type (PostgreSQL, MariaDB, Mysql, SQLite, ClickHouse, Oracle)
@ name: Identifier to escape
@ Return: Escaped identifier and error if any
*/
func EscapeIdentifier(dbType DBType, name string) (string, error) {
	d := dialectFor(dbType)
	if d == unknownDialect {
		return "", unknownDBTypeError(dbType)
	}
	return d.escape(name)
}

func unknownDBTypeError(dbType DBType) error {
	return fmt.Errorf("unsupported database type %q; use one of the gqbd DBType constants", string(dbType))
}

// Clone returns an independent copy of the builder, so a shared base query can
// be extended without changing it:
//
//	base := gqbd.BuildSelect(gqbd.PostgreSQL, "orders").Where("tenant_id = ?", tenant)
//	open := base.Clone().Where("status = ?", "open")
//
// The Values/Set map is shared with the original, not copied; Values and Set
// replace it rather than modify it.
func (qb *QueryBuilder) Clone() *QueryBuilder {
	// Builders only ever append to their slices, so the copy can share the
	// backing arrays: clipping them to their length makes the first append on
	// either side copy instead of writing into the other's elements.
	c := *qb
	c.columns = slices.Clip(qb.columns)
	c.joins = slices.Clip(qb.joins)
	c.where = slices.Clip(qb.where)
	c.whereArgs = slices.Clip(qb.whereArgs)
	c.groupBy = slices.Clip(qb.groupBy)
	c.having = slices.Clip(qb.having)
	c.havingArgs = slices.Clip(qb.havingArgs)
	c.ch = qb.ch.clone()
	return &c
}

// clone copies ch the way Clone copies a builder; a nil ch stays nil.
func (ch *clickhouseClauses) clone() *clickhouseClauses {
	if ch == nil {
		return nil
	}
	return &clickhouseClauses{
		final:        ch.final,
		prewhere:     slices.Clip(ch.prewhere),
		prewhereArgs: slices.Clip(ch.prewhereArgs),
		settings:     slices.Clip(ch.settings),
	}
}

/*
ValidateDirection

@ direction: Order direction string
@ Return: Validated order direction ("ASC" or "DESC")
*/
func ValidateDirection(direction string) string {
	if strings.EqualFold(direction, "ASC") {
		return "ASC"
	}
	return "DESC"
}

/*
ReplacePlaceholders

@ dbType: Database type
@ condition: Condition string with placeholders
@ startIdx: Starting index for placeholders
@ Return: Condition string with replaced placeholders
*/
func ReplacePlaceholders(dbType DBType, condition string, startIdx int) string {
	d := dialectFor(dbType)
	if d.numPrefix == 0 {
		return condition
	}
	w := newSQLWriter(d, len(condition)+8)
	w.n = startIdx - 1
	w.cond(condition)
	return w.String()
}

/*
GeneratePlaceholders

@ dbType: Database type
@ startIdx: Starting index for placeholders
@ count: Number of placeholders to generate
@ Return: String of placeholders separated by comma
*/
func GeneratePlaceholders(dbType DBType, startIdx, count int) string {
	w := newSQLWriter(dialectFor(dbType), count*5)
	w.n = startIdx - 1
	w.placeholders(count)
	return w.String()
}
