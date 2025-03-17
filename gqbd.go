package gqbd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DBType represents the type of database.
type DBType string

const (
	PostgreSQL DBType = "postgres"
	MariaDB    DBType = "mariadb"
	Mysql      DBType = "mysql"
)

// QueryBuilder is a flexible SQL query builder.
type QueryBuilder struct {
	op         string // "SELECT", "INSERT", "UPDATE", "DELETE"
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
	data       map[string]interface{} // for INSERT and UPDATE
	returning  string                 // for INSERT, Postgres only
}

var placeholderRegexp = regexp.MustCompile(`\$(\d+)`)

/*
BuildSelect

@ dbType: Database type (PostgreSQL, MariaDB, Mysql)
@ table: Table name
@ columns: Columns to select
@ Return: *QueryBuilder with SELECT operation
*/
func BuildSelect(dbType DBType, table string, columns ...string) *QueryBuilder {
	qb := NewQueryBuilder(dbType, table, columns...)
	qb.op = "SELECT"
	return qb
}

/*
BuildInsert

@ dbType: Database type (PostgreSQL, MariaDB, Mysql)
@ table: Table name
@ Return: *QueryBuilder with INSERT operation
*/
func BuildInsert(dbType DBType, table string) *QueryBuilder {
	qb := NewQueryBuilder(dbType, table)
	qb.op = "INSERT"
	return qb
}

/*
BuildUpdate

@ dbType: Database type (PostgreSQL, MariaDB, Mysql)
@ table: Table name
@ Return: *QueryBuilder with UPDATE operation
*/
func BuildUpdate(dbType DBType, table string) *QueryBuilder {
	qb := NewQueryBuilder(dbType, table)
	qb.op = "UPDATE"
	return qb
}

/*
BuildDelete

@ dbType: Database type (PostgreSQL, MariaDB, Mysql)
@ table: Table name
@ Return: *QueryBuilder with DELETE operation
*/
func BuildDelete(dbType DBType, table string) *QueryBuilder {
	qb := NewQueryBuilder(dbType, table)
	qb.op = "DELETE"
	return qb
}

/*
NewQueryBuilder

@ dbType: Database type (PostgreSQL, MariaDB, Mysql)
@ table: Table name
@ columns: Columns to select (variadic)
@ Return: *QueryBuilder instance
*/
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

/*
Distinct

@ Return: *QueryBuilder with DISTINCT enabled
*/
func (qb *QueryBuilder) Distinct() *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	qb.distinct = true
	return qb
}

/*
Aggregate

@ function: Aggregate function (e.g., COUNT, SUM, AVG)
@ column: Column name to aggregate
@ Return: *QueryBuilder with aggregate function added
*/
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

/*
LeftJoin

@ joinTable: Table name to join
@ onCondition: Join condition
@ Return: *QueryBuilder with LEFT JOIN added
*/
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

/*
Where

@ condition: Condition string with placeholders
@ args: Query parameters
@ Return: *QueryBuilder with WHERE clause added
*/
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
	qb.conditions = append(qb.conditions, fmt.Sprintf(" BETWEEN %s AND %s", safeCol, placeholders))
	qb.args = append(qb.args, start, end)
	return qb
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

/*
Build

@ Return: Final query string, arguments slice, and error if any
*/
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
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT ")
	if qb.distinct {
		queryBuilder.WriteString("DISTINCT ")
	}
	queryBuilder.WriteString(strings.Join(qb.columns, ", "))
	queryBuilder.WriteString(" FROM ")
	queryBuilder.WriteString(qb.table)
	if len(qb.joins) > 0 {
		queryBuilder.WriteString(" " + strings.Join(qb.joins, " "))
	}
	if len(qb.conditions) > 0 {
		queryBuilder.WriteString(" WHERE " + strings.Join(qb.conditions, " AND "))
	}
	if len(qb.groupBy) > 0 {
		queryBuilder.WriteString(" GROUP BY " + strings.Join(qb.groupBy, ", "))
	}
	if len(qb.having) > 0 {
		queryBuilder.WriteString(" HAVING " + strings.Join(qb.having, " AND "))
	}
	if qb.orderBy != "" {
		queryBuilder.WriteString(" ORDER BY " + qb.orderBy)
	}
	argIdx := len(qb.args) + 1
	if qb.limit > 0 {
		if qb.dbType == PostgreSQL {
			queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d", argIdx))
		} else {
			queryBuilder.WriteString(" LIMIT ?")
		}
		qb.args = append(qb.args, qb.limit)
		argIdx++
	}
	if qb.offset > 0 {
		if qb.dbType == PostgreSQL {
			queryBuilder.WriteString(fmt.Sprintf(" OFFSET $%d", argIdx))
		} else {
			queryBuilder.WriteString(" OFFSET ?")
		}
		qb.args = append(qb.args, qb.offset)
	}
	return queryBuilder.String(), qb.args, nil
}

func (qb *QueryBuilder) buildInsert() (string, []interface{}, error) {
	if qb.data == nil {
		return "", nil, fmt.Errorf("no data provided for INSERT")
	}
	var cols []string
	var placeholders []string
	var args []interface{}
	idx := 1
	for col, val := range qb.data {
		safeCol, err := EscapeIdentifier(qb.dbType, col)
		if err != nil {
			return "", nil, err
		}
		cols = append(cols, safeCol)
		if qb.dbType == PostgreSQL {
			placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
		} else {
			placeholders = append(placeholders, "?")
		}
		args = append(args, val)
		idx++
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", qb.table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	if qb.dbType == PostgreSQL && qb.returning != "" {
		query += " RETURNING " + qb.returning
	}
	return query, args, nil
}

func (qb *QueryBuilder) buildUpdate() (string, []interface{}, error) {
	if qb.data == nil {
		return "", nil, fmt.Errorf("no data provided for UPDATE")
	}
	var setClauses []string
	var updateArgs []interface{}
	idx := 1
	for col, val := range qb.data {
		safeCol, err := EscapeIdentifier(qb.dbType, col)
		if err != nil {
			return "", nil, err
		}
		var placeholder string
		if qb.dbType == PostgreSQL {
			placeholder = fmt.Sprintf("$%d", idx)
		} else {
			placeholder = "?"
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = %s", safeCol, placeholder))
		updateArgs = append(updateArgs, val)
		idx++
	}
	query := fmt.Sprintf("UPDATE %s SET %s", qb.table, strings.Join(setClauses, ", "))
	if len(qb.conditions) > 0 {
		if qb.dbType == PostgreSQL {
			shiftedConds := make([]string, len(qb.conditions))
			for i, cond := range qb.conditions {
				shiftedConds[i] = shiftPlaceholders(cond, len(qb.data))
			}
			query += " WHERE " + strings.Join(shiftedConds, " AND ")
		} else {
			query += " WHERE " + strings.Join(qb.conditions, " AND ")
		}
		updateArgs = append(updateArgs, qb.args...)
	}
	return query, updateArgs, nil
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
	return placeholderRegexp.ReplaceAllStringFunc(condition, func(match string) string {
		numStr := match[1:]
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return match
		}
		return fmt.Sprintf("$%d", num+offset)
	})
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
	if dbType == PostgreSQL {
		return fmt.Sprintf(`"%s"`, strings.ReplaceAll(name, `"`, `""`)), nil
	}
	if dbType == MariaDB || dbType == Mysql {
		return fmt.Sprintf("`%s`", strings.ReplaceAll(name, "`", "``")), nil
	}
	return "", fmt.Errorf("unsupported db type: %v", dbType)
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
	if dbType == MariaDB {
		return condition // MariaDB uses "?" directly
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
