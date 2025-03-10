package builder

import (
	"fmt"
	"strings"
)

// DB 타입 정의 (PostgreSQL, MariaDB)
type DBType string

const (
	PostgreSQL DBType = "postgres"
	MariaDB    DBType = "mariadb"
)

// QueryBuilder 구조체
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

// NewQueryBuilder: 테이블 및 컬럼 이스케이프 처리
func NewQueryBuilder(dbType DBType, table string, columns ...string) *QueryBuilder {
	safeTable := escapeIdentifier(dbType, table)
	safeColumns := make([]string, len(columns))
	for i, col := range columns {
		safeColumns[i] = escapeIdentifier(dbType, col)
	}
	if len(safeColumns) == 0 {
		safeColumns = []string{"*"}
	}
	return &QueryBuilder{
		dbType:  dbType,
		table:   safeTable,
		columns: safeColumns,
	}
}

// Distinct: DISTINCT 추가
func (qb *QueryBuilder) Distinct() *QueryBuilder {
	qb.distinct = true
	return qb
}

// Aggregate: COUNT, SUM, AVG 등 집계 함수 지원
func (qb *QueryBuilder) Aggregate(function, column string) *QueryBuilder {
	safeCol := escapeIdentifier(qb.dbType, column)
	qb.columns = append(qb.columns, fmt.Sprintf("%s(%s)", function, safeCol))
	return qb
}

// LeftJoin: LEFT JOIN 추가
func (qb *QueryBuilder) LeftJoin(joinTable string, onCondition string) *QueryBuilder {
	safeTable := escapeIdentifier(qb.dbType, joinTable)
	qb.joins = append(qb.joins, fmt.Sprintf("LEFT JOIN %s ON %s", safeTable, onCondition))
	return qb
}

// InnerJoin: INNER JOIN 추가
func (qb *QueryBuilder) InnerJoin(joinTable string, onCondition string) *QueryBuilder {
	safeTable := escapeIdentifier(qb.dbType, joinTable)
	qb.joins = append(qb.joins, fmt.Sprintf("INNER JOIN %s ON %s", safeTable, onCondition))
	return qb
}

// RightJoin: RIGHT JOIN 추가
func (qb *QueryBuilder) RightJoin(joinTable string, onCondition string) *QueryBuilder {
	safeTable := escapeIdentifier(qb.dbType, joinTable)
	qb.joins = append(qb.joins, fmt.Sprintf("RIGHT JOIN %s ON %s", safeTable, onCondition))
	return qb
}

// Where: 안전한 WHERE 처리 (PostgreSQL: $1, $2 / MariaDB: ?)
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	updatedCondition := replacePlaceholders(qb.dbType, condition, len(qb.args)+1)
	qb.conditions = append(qb.conditions, updatedCondition)
	qb.args = append(qb.args, args...)
	return qb
}

// WhereIn: IN 조건 추가
func (qb *QueryBuilder) WhereIn(column string, values []interface{}) *QueryBuilder {
	safeCol := escapeIdentifier(qb.dbType, column)
	placeholders := generatePlaceholders(qb.dbType, len(qb.args)+1, len(values))
	qb.conditions = append(qb.conditions, fmt.Sprintf("%s IN (%s)", safeCol, placeholders))
	qb.args = append(qb.args, values...)
	return qb
}

// WhereBetween: BETWEEN 조건 추가
func (qb *QueryBuilder) WhereBetween(column string, start, end interface{}) *QueryBuilder {
	safeCol := escapeIdentifier(qb.dbType, column)
	placeholders := generatePlaceholders(qb.dbType, len(qb.args)+1, 2)
	qb.conditions = append(qb.conditions, fmt.Sprintf("%s BETWEEN %s AND %s", safeCol, placeholders))
	qb.args = append(qb.args, start, end)
	return qb
}

// GroupBy: GROUP BY 추가
func (qb *QueryBuilder) GroupBy(columns ...string) *QueryBuilder {
	for _, col := range columns {
		qb.groupBy = append(qb.groupBy, escapeIdentifier(qb.dbType, col))
	}
	return qb
}

// Having: HAVING 조건 추가
func (qb *QueryBuilder) Having(condition string, args ...interface{}) *QueryBuilder {
	updatedCondition := replacePlaceholders(qb.dbType, condition, len(qb.args)+1)
	qb.having = append(qb.having, updatedCondition)
	qb.args = append(qb.args, args...)
	return qb
}

// OrderBy: 정렬 추가
func (qb *QueryBuilder) OrderBy(column string, direction string, allowedColumns map[string]bool) *QueryBuilder {
	direction = validateDirection(direction)
	if allowedColumns != nil {
		if _, ok := allowedColumns[column]; !ok {
			column = "id" // 기본 정렬 컬럼 (변경 가능)
		}
	}

	safeCol := escapeIdentifier(qb.dbType, column)
	qb.orderBy = fmt.Sprintf("%s %s", safeCol, direction)
	return qb
}

// DynamicOrderBy: 안전한 동적 정렬 처리
func (qb *QueryBuilder) DynamicOrderBy(dynamicColumn, defaultColumn, direction string, allowedColumns map[string]bool) *QueryBuilder {
	direction = validateDirection(direction)
	targetColumn := defaultColumn
	if dynamicColumn != "" && allowedColumns[dynamicColumn] {
		targetColumn = dynamicColumn
	}
	safeCol := escapeIdentifier(qb.dbType, targetColumn)
	qb.orderBy = fmt.Sprintf("%s %s", safeCol, direction)
	return qb
}

// Build: 최종 쿼리 생성 (나머지 메소드는 동일)

func escapeIdentifier(dbType DBType, name string) string {
	if name == "*" {
		return name
	}
	if dbType == PostgreSQL {
		return fmt.Sprintf(`"%s"`, strings.ReplaceAll(name, `"`, `""`))
	}
	return fmt.Sprintf("`%s`", strings.ReplaceAll(name, "`", "``"))
}

// 🔹 정렬 방향 검증 (ASC / DESC만 허용)
func validateDirection(direction string) string {
	direction = strings.ToUpper(direction)
	if direction != "ASC" && direction != "DESC" {
		return "DESC"
	}
	return direction
}

// 🔹 플레이스홀더 변환 (PostgreSQL: $N / MariaDB: ?)
func replacePlaceholders(dbType DBType, condition string, startIdx int) string {
	if dbType == MariaDB {
		return condition // MariaDB는 그냥 ? 사용
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

// Limit: 제한 추가
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset: 페이지네이션 추가
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// generatePlaceholders: PostgreSQL($N) & MariaDB(?) 플레이스홀더 생성
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

// Build: 최종 SQL 쿼리 생성
func (qb *QueryBuilder) Build() (string, []interface{}) {
	var queryBuilder strings.Builder

	// SELECT 절
	queryBuilder.WriteString("SELECT ")
	queryBuilder.WriteString(strings.Join(qb.columns, ", "))
	queryBuilder.WriteString(" FROM ")
	queryBuilder.WriteString(qb.table)

	// JOIN 절 추가
	if len(qb.joins) > 0 {
		queryBuilder.WriteString(" ")
		queryBuilder.WriteString(strings.Join(qb.joins, " "))
	}

	// WHERE 절
	if len(qb.conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(qb.conditions, " AND "))
	}

	// ORDER BY 절
	if qb.orderBy != "" {
		queryBuilder.WriteString(" ORDER BY ")
		queryBuilder.WriteString(qb.orderBy)
	}

	// LIMIT & OFFSET 추가 (PostgreSQL은 $N 형식 사용)
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
