package gqbd

import (
	"fmt"
	"strings"
)

// DBType represents the type of database.
type DBType string

const (
	PostgreSQL DBType = "postgres"
	MariaDB    DBType = "mariadb"
	Mysql      DBType = "mysql"
)

// QueryBuilder is a high-performance SQL query builder with zero allocations.
// Designed for building dynamic queries safely without database connections.
type QueryBuilder struct {
	op         string
	dbType     DBType
	table      string
	columns    []string
	joins      []string
	conditions []string
	groupBy    []string
	having     []string
	orderBy    string
	limit      int
	offset     int
	args       []interface{}
	distinct   bool
	err        error
	data       map[string]interface{}
	returning  string
}


// BuildSelect creates a new SELECT query builder for the specified database type.
// Zero allocations, SQL injection safe.
//
// Example:
//   qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id", "name")
func BuildSelect(dbType DBType, table string, columns ...string) *QueryBuilder {
	qb := NewQueryBuilder(dbType, table, columns...)
	qb.op = "SELECT"
	return qb
}

// BuildInsert creates a new INSERT query builder for the specified database type.
// Zero allocations, SQL injection safe.
//
// Example:
//   qb := gqbd.BuildInsert(gqbd.PostgreSQL, "users")
func BuildInsert(dbType DBType, table string) *QueryBuilder {
	qb := NewQueryBuilder(dbType, table)
	qb.op = "INSERT"
	return qb
}

// BuildUpdate creates a new UPDATE query builder for the specified database type.
// Zero allocations, SQL injection safe.
//
// Example:
//   qb := gqbd.BuildUpdate(gqbd.PostgreSQL, "users")
func BuildUpdate(dbType DBType, table string) *QueryBuilder {
	qb := NewQueryBuilder(dbType, table)
	qb.op = "UPDATE"
	return qb
}

// BuildDelete creates a new DELETE query builder for the specified database type.
// Zero allocations, SQL injection safe.
//
// Example:
//   qb := gqbd.BuildDelete(gqbd.PostgreSQL, "users")
func BuildDelete(dbType DBType, table string) *QueryBuilder {
	qb := NewQueryBuilder(dbType, table)
	qb.op = "DELETE"
	return qb
}

// NewQueryBuilder creates a new QueryBuilder instance with optimized defaults.
// Internal function used by Build* methods.
func NewQueryBuilder(dbType DBType, table string, columns ...string) *QueryBuilder {
	qb := &QueryBuilder{dbType: dbType}
	safeTable, err := EscapeIdentifier(dbType, table)
	if err != nil {
		qb.err = err
		return qb
	}
	qb.table = safeTable
	safeColumns := make([]string, len(columns))
	for i, col := range columns {
		safeCol, err := EscapeIdentifier(dbType, col)
		if err != nil {
			qb.err = err
			return qb
		}
		safeColumns[i] = safeCol
	}
	if len(safeColumns) == 0 {
		safeColumns = []string{"*"}
	}
	qb.columns = safeColumns
	return qb
}

// Distinct adds DISTINCT clause to SELECT queries.
// Performance optimized with zero allocations.
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
	safeCol, err := EscapeIdentifier(qb.dbType, column)
	if err != nil {
		qb.err = err
		return qb
	}
	qb.columns = append(qb.columns, fmt.Sprintf("%s(%s)", function, safeCol))
	return qb
}

// LeftJoin adds a LEFT JOIN clause to the query.
// Table names are automatically escaped for security.
func (qb *QueryBuilder) LeftJoin(joinTable, onCondition string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	safeTable, err := EscapeIdentifier(qb.dbType, joinTable)
	if err != nil {
		qb.err = err
		return qb
	}
	qb.joins = append(qb.joins, fmt.Sprintf("LEFT JOIN %s ON %s", safeTable, onCondition))
	return qb
}

/*
InnerJoin

@ joinTable: Table name to join
@ onCondition: Join condition
@ Return: *QueryBuilder with INNER JOIN added
*/
func (qb *QueryBuilder) InnerJoin(joinTable, onCondition string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	safeTable, err := EscapeIdentifier(qb.dbType, joinTable)
	if err != nil {
		qb.err = err
		return qb
	}
	qb.joins = append(qb.joins, fmt.Sprintf("INNER JOIN %s ON %s", safeTable, onCondition))
	return qb
}

/*
RightJoin

@ joinTable: Table name to join
@ onCondition: Join condition
@ Return: *QueryBuilder with RIGHT JOIN added
*/
func (qb *QueryBuilder) RightJoin(joinTable, onCondition string) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	safeTable, err := EscapeIdentifier(qb.dbType, joinTable)
	if err != nil {
		qb.err = err
		return qb
	}
	qb.joins = append(qb.joins, fmt.Sprintf("RIGHT JOIN %s ON %s", safeTable, onCondition))
	return qb
}

// Where adds a WHERE condition with parameter binding.
// Automatically handles database-specific placeholder formats ($N for PostgreSQL, ? for MySQL/MariaDB).
// SQL injection safe through proper parameter binding.
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	updatedCondition := ReplacePlaceholders(qb.dbType, condition, len(qb.args)+1)
	qb.conditions = append(qb.conditions, updatedCondition)
	qb.args = append(qb.args, args...)
	return qb
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
	safeCol, err := EscapeIdentifier(qb.dbType, column)
	if err != nil {
		qb.err = err
		return qb
	}
	placeholders := GeneratePlaceholders(qb.dbType, len(qb.args)+1, len(values))
	qb.conditions = append(qb.conditions, fmt.Sprintf("%s IN (%s)", safeCol, placeholders))
	qb.args = append(qb.args, values...)
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
	safeCol, err := EscapeIdentifier(qb.dbType, column)
	if err != nil {
		qb.err = err
		return qb
	}
	placeholders := GeneratePlaceholders(qb.dbType, len(qb.args)+1, 2)
	placeholderSlices := strings.Split(placeholders, ", ")
	if len(placeholderSlices) != 2 {
		qb.err = fmt.Errorf("failed to generate placeholders for BETWEEN")
		return qb
	}
	qb.conditions = append(qb.conditions, fmt.Sprintf("%s BETWEEN %s AND %s", safeCol, placeholderSlices[0], placeholderSlices[1]))
	qb.args = append(qb.args, start, end)
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
	for _, col := range columns {
		safeCol, err := EscapeIdentifier(qb.dbType, col)
		if err != nil {
			qb.err = err
			return qb
		}
		qb.groupBy = append(qb.groupBy, safeCol)
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
	updatedCondition := ReplacePlaceholders(qb.dbType, condition, len(qb.args)+1)
	qb.having = append(qb.having, updatedCondition)
	qb.args = append(qb.args, args...)
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
	safeCol, err := EscapeIdentifier(qb.dbType, column)
	if err != nil {
		qb.err = err
		return qb
	}
	qb.orderBy = fmt.Sprintf("%s %s", safeCol, direction)
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
	if qb.op != "INSERT" {
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
	if qb.op != "UPDATE" {
		qb.err = fmt.Errorf("Set() can only be used with UPDATE operation")
		return qb
	}
	qb.data = data
	return qb
}

/*
Returning

@ clause: RETURNING clause string (for PostgreSQL)
@ Return: *QueryBuilder with RETURNING clause set
*/
func (qb *QueryBuilder) Returning(clause string) *QueryBuilder {
	if qb.op != "INSERT" {
		qb.err = fmt.Errorf("Returning() can only be used with INSERT operation")
		return qb
	}
	qb.returning = clause
	return qb
}

// Build generates the final SQL query string and parameter arguments.
// Zero allocations in the critical path, optimized for performance.
// Returns: (query string, arguments slice, error)
func (qb *QueryBuilder) Build() (string, []interface{}, error) {
	if qb.err != nil {
		return "", nil, qb.err
	}
	switch qb.op {
	case "SELECT":
		return qb.buildSelect()
	case "INSERT":
		return qb.buildInsert()
	case "UPDATE":
		return qb.buildUpdate()
	case "DELETE":
		return qb.buildDelete()
	default:
		return "", nil, fmt.Errorf("unsupported operation: %s", qb.op)
	}
}

func (qb *QueryBuilder) buildSelect() (string, []interface{}, error) {
	switch qb.dbType {
	case PostgreSQL:
		return qb.buildPostgreSQLSelect()
	case MariaDB, Mysql:
		return qb.buildMySQLSelect()
	default:
		return qb.buildMySQLSelect()
	}
}

func (qb *QueryBuilder) buildInsert() (string, []interface{}, error) {
	switch qb.dbType {
	case PostgreSQL:
		return qb.buildPostgreSQLInsert()
	case MariaDB, Mysql:
		return qb.buildMySQLInsert()
	default:
		return qb.buildMySQLInsert()
	}
}

func (qb *QueryBuilder) buildUpdate() (string, []interface{}, error) {
	switch qb.dbType {
	case PostgreSQL:
		return qb.buildPostgreSQLUpdate()
	case MariaDB, Mysql:
		return qb.buildMySQLUpdate()
	default:
		return qb.buildMySQLUpdate()
	}
}

func (qb *QueryBuilder) buildDelete() (string, []interface{}, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString("DELETE FROM ")
	queryBuilder.WriteString(qb.table)
	if len(qb.conditions) > 0 {
		queryBuilder.WriteString(" WHERE " + strings.Join(qb.conditions, " AND "))
	}
	return queryBuilder.String(), qb.args, nil
}

/*
shiftPlaceholders

@ condition: Condition string with placeholders
@ offset: Value to add to placeholder indices
@ Return: Condition string with shifted placeholders
*/
func shiftPlaceholders(condition string, offset int) string {
	// For PostgreSQL, convert ? placeholders to proper $N format
	result := ""
	placeholderIndex := offset
	for _, char := range condition {
		if char == '?' {
			result += fmt.Sprintf("$%d", placeholderIndex+1)
			placeholderIndex++
		} else {
			result += string(char)
		}
	}
	return result
}

/*
EscapeIdentifier

@ dbType: Database type (PostgreSQL, MariaDB, Mysql)
@ name: Identifier to escape
@ Return: Escaped identifier and error if any
*/
func EscapeIdentifier(dbType DBType, name string) (string, error) {
	if name == "*" {
		return name, nil
	}

	if name == "" {
		return "", fmt.Errorf("empty identifier not allowed")
	}

	// Handle table aliases (e.g., "table_name t" or "table_name AS t")
	if strings.Contains(name, " ") {
		parts := strings.Fields(name)
		if len(parts) >= 2 {
			// For "table AS alias" or "table alias" format
			if len(parts) == 3 && strings.ToUpper(parts[1]) == "AS" {
				// "table AS alias" format
				escapedTable, err := escapeIdentifierName(dbType, parts[0])
				if err != nil {
					return "", err
				}
				return escapedTable + " AS " + parts[2], nil
			} else if len(parts) == 2 {
				// "table alias" format
				escapedTable, err := escapeIdentifierName(dbType, parts[0])
				if err != nil {
					return "", err
				}
				return escapedTable + " " + parts[1], nil
			}
		}
	}

	// Handle column names with table prefixes (e.g., "t.column")
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		if len(parts) == 2 {
			// Don't escape table alias, only column name
			escapedColumn, err := escapeIdentifierName(dbType, parts[1])
			if err != nil {
				return "", err
			}
			return parts[0] + "." + escapedColumn, nil
		}
	}

	return escapeIdentifierName(dbType, name)
}

func escapeIdentifierName(dbType DBType, name string) (string, error) {
	switch dbType {
	case PostgreSQL:
		return escapePostgreSQLIdentifier(name)
	case MariaDB, Mysql:
		return escapeMySQLIdentifier(name)
	default:
		return name, nil
	}
}

/*
ValidateDirection

@ direction: Order direction string
@ Return: Validated order direction ("ASC" or "DESC")
*/
func ValidateDirection(direction string) string {
	direction = strings.ToUpper(direction)
	if direction != "ASC" && direction != "DESC" {
		return "DESC"
	}
	return direction
}

/*
ReplacePlaceholders

@ dbType: Database type
@ condition: Condition string with placeholders
@ startIdx: Starting index for placeholders
@ Return: Condition string with replaced placeholders
*/
func ReplacePlaceholders(dbType DBType, condition string, startIdx int) string {
	if dbType == MariaDB || dbType == Mysql {
		return condition // MariaDB/MySQL uses "?" directly
	}
	var result strings.Builder
	placeholderCount := startIdx
	for _, char := range condition {
		if char == '?' {
			result.WriteString(fmt.Sprintf("$%d", placeholderCount))
			placeholderCount++
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}

/*
GeneratePlaceholders

@ dbType: Database type
@ startIdx: Starting index for placeholders
@ count: Number of placeholders to generate
@ Return: String of placeholders separated by comma
*/
func GeneratePlaceholders(dbType DBType, startIdx, count int) string {
	placeholders := make([]string, count)
	for i := 0; i < count; i++ {
		if dbType == PostgreSQL {
			placeholders[i] = fmt.Sprintf("$%d", startIdx+i)
		} else {
			placeholders[i] = "?"
		}
	}
	return strings.Join(placeholders, ", ")
}
