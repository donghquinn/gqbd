package gqbd

import (
	"fmt"
	"strings"
)

// DBType represents the type of database (PostgreSQL or MariaDB).
type DBType string

const (
	PostgreSQL DBType = "postgres"
	MariaDB    DBType = "mariadb"
	Mysql      DBType = "mysql"
)

// QueryBuilder is a flexible SQL query builder that supports both PostgreSQL and MariaDB.
// It allows constructing complex queries with WHERE, JOIN, GROUP BY, ORDER BY, LIMIT, and more.
type QueryBuilder struct {
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
}

// NewQueryBuilder initializes a new QueryBuilder instance for a given table and column selection.
// It ensures that table and column names are safely escaped.
func NewQueryBuilder(dbType DBType, table string, columns ...string) (*QueryBuilder, error) {
	safeTable, identifyErr := escapeIdentifier(dbType, table)
	if identifyErr != nil {
		return nil, identifyErr
	}
	safeColumns := make([]string, len(columns))
	for i, col := range columns {
		safeCol, identifyErr := escapeIdentifier(dbType, col)
		if identifyErr != nil {
			return nil, identifyErr
		}
		safeColumns[i] = safeCol
	}
	if len(safeColumns) == 0 {
		safeColumns = []string{"*"}
	}
	return &QueryBuilder{
		dbType:  dbType,
		table:   safeTable,
		columns: safeColumns,
	}, nil
}

// Distinct enables DISTINCT in the SQL query.
func (qb *QueryBuilder) Distinct() *QueryBuilder {
	qb.distinct = true
	return qb
}

// Aggregate adds aggregate functions (e.g., COUNT, SUM, AVG) to the query.
func (qb *QueryBuilder) Aggregate(function, column string) (*QueryBuilder, error) {
	safeCol, identifyErr := escapeIdentifier(qb.dbType, column)
	if identifyErr != nil {
		return nil, identifyErr
	}
	qb.columns = append(qb.columns, fmt.Sprintf("%s(%s)", function, safeCol))
	return qb, nil
}

// LeftJoin adds a LEFT JOIN clause to the query.
func (qb *QueryBuilder) LeftJoin(joinTable, onCondition string) (*QueryBuilder, error) {
	safeTable, identifyErr := escapeIdentifier(qb.dbType, joinTable)
	if identifyErr != nil {
		return nil, identifyErr
	}
	qb.joins = append(qb.joins, fmt.Sprintf("LEFT JOIN %s ON %s", safeTable, onCondition))
	return qb, nil
}

// InnerJoin adds an INNER JOIN clause to the query.
func (qb *QueryBuilder) InnerJoin(joinTable, onCondition string) (*QueryBuilder, error) {
	safeTable, identifyErr := escapeIdentifier(qb.dbType, joinTable)
	if identifyErr != nil {
		return nil, identifyErr
	}
	qb.joins = append(qb.joins, fmt.Sprintf("INNER JOIN %s ON %s", safeTable, onCondition))
	return qb, nil
}

// RightJoin adds a RIGHT JOIN clause to the query.
func (qb *QueryBuilder) RightJoin(joinTable, onCondition string) (*QueryBuilder, error) {
	safeTable, identifyErr := escapeIdentifier(qb.dbType, joinTable)
	if identifyErr != nil {
		return nil, identifyErr
	}
	qb.joins = append(qb.joins, fmt.Sprintf("RIGHT JOIN %s ON %s", safeTable, onCondition))
	return qb, nil
}

// Where adds a WHERE clause with safely parameterized conditions.
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	updatedCondition := replacePlaceholders(qb.dbType, condition, len(qb.args)+1)
	qb.conditions = append(qb.conditions, updatedCondition)
	qb.args = append(qb.args, args...)
	return qb
}

// WhereIn adds an IN clause with multiple values.
func (qb *QueryBuilder) WhereIn(column string, values []interface{}) (*QueryBuilder, error) {
	safeCol, identifyErr := escapeIdentifier(qb.dbType, column)
	if identifyErr != nil {
		return nil, identifyErr
	}
	placeholders := generatePlaceholders(qb.dbType, len(qb.args)+1, len(values))
	qb.conditions = append(qb.conditions, fmt.Sprintf("%s IN (%s)", safeCol, placeholders))
	qb.args = append(qb.args, values...)
	return qb, nil
}

// WhereBetween adds a BETWEEN clause to the query.
func (qb *QueryBuilder) WhereBetween(column string, start, end interface{}) (*QueryBuilder, error) {
	safeCol, identifyErr := escapeIdentifier(qb.dbType, column)
	if identifyErr != nil {
		return nil, identifyErr
	}
	placeholders := generatePlaceholders(qb.dbType, len(qb.args)+1, 2)
	qb.conditions = append(qb.conditions, fmt.Sprintf(" BETWEEN %s AND %s", safeCol, placeholders))
	qb.args = append(qb.args, start, end)
	return qb, nil
}

// GroupBy adds GROUP BY clauses to the query.
func (qb *QueryBuilder) GroupBy(columns ...string) (*QueryBuilder, error) {
	for _, col := range columns {
		safeCol, identifyErr := escapeIdentifier(qb.dbType, col)
		if identifyErr != nil {
			return nil, identifyErr
		}
		qb.groupBy = append(qb.groupBy, safeCol)
	}
	return qb, nil
}

// Having adds a HAVING clause to filter aggregated results.
func (qb *QueryBuilder) Having(condition string, args ...interface{}) *QueryBuilder {
	updatedCondition := replacePlaceholders(qb.dbType, condition, len(qb.args)+1)
	qb.having = append(qb.having, updatedCondition)
	qb.args = append(qb.args, args...)
	return qb
}

// OrderBy adds an ORDER BY clause with SQL injection protection via allowed columns.
func (qb *QueryBuilder) OrderBy(column, direction string, allowedColumns map[string]bool) (*QueryBuilder, error) {
	direction = validateDirection(direction)
	if allowedColumns != nil {
		if _, ok := allowedColumns[column]; !ok {
			column = "id" // Default sorting column
		}
	}
	safeCol, identifyErr := escapeIdentifier(qb.dbType, column)
	if identifyErr != nil {
		return nil, identifyErr
	}
	qb.orderBy = fmt.Sprintf("%s %s", safeCol, direction)
	return qb, nil
}

// Limit sets the query's LIMIT value.
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset sets the query's OFFSET value.
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// escapeIdentifier safely escapes table and column names to prevent SQL injection.
func escapeIdentifier(dbType DBType, name string) (string, error) {
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

// validateDirection ensures only "ASC" or "DESC" are used in ORDER BY clauses.
func validateDirection(direction string) string {
	direction = strings.ToUpper(direction)
	if direction != "ASC" && direction != "DESC" {
		return "DESC"
	}
	return direction
}

// replacePlaceholders replaces placeholders with parameterized values for safe SQL execution.
func replacePlaceholders(dbType DBType, condition string, startIdx int) string {
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

// generatePlaceholders generates SQL placeholders for parameterized queries.
func generatePlaceholders(dbType DBType, startIdx, count int) string {
	placeholders := make([]string, count)

	for i := 0; i < count; i++ {
		if dbType == PostgreSQL {
			placeholders[i] = fmt.Sprintf("$%d", startIdx+i)
		} else { // MariaDB
			placeholders[i] = "?"
		}
	}

	return strings.Join(placeholders, ", ")
}

// Build constructs the final SQL query string with safely parameterized values.
func (qb *QueryBuilder) Build() (string, []interface{}) {
	var queryBuilder strings.Builder

	// SELECT clause
	queryBuilder.WriteString("SELECT ")
	queryBuilder.WriteString(strings.Join(qb.columns, ", "))
	queryBuilder.WriteString(" FROM ")
	queryBuilder.WriteString(qb.table)

	// JOIN clauses
	if len(qb.joins) > 0 {
		queryBuilder.WriteString(" " + strings.Join(qb.joins, " "))
	}

	// WHERE clause
	if len(qb.conditions) > 0 {
		queryBuilder.WriteString(" WHERE " + strings.Join(qb.conditions, " AND "))
	}

	// ORDER BY clause
	if qb.orderBy != "" {
		queryBuilder.WriteString(" ORDER BY " + qb.orderBy)
	}

	// LIMIT & OFFSET handling
	argIdx := len(qb.args) + 1
	if qb.limit > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d", argIdx))
		qb.args = append(qb.args, qb.limit)
		argIdx++
	}
	if qb.offset > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" OFFSET $%d", argIdx))
		qb.args = append(qb.args, qb.offset)
	}

	return queryBuilder.String(), qb.args
}
