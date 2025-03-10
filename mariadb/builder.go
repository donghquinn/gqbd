package utils

import (
	"fmt"
	"strings"
)

// QueryBuilder 구조체
type QueryBuilder struct {
	table      string
	columns    []string
	joins      []string
	conditions []string
	orderBy    string
	limit      int
	offset     int
	args       []interface{}
}

// NewQueryBuilder: 테이블 및 컬럼 이스케이프 처리
func NewQueryBuilder(table string, columns ...string) *QueryBuilder {
	safeTable := escapeIdentifier(table)
	safeColumns := make([]string, len(columns))
	for i, col := range columns {
		safeColumns[i] = escapeIdentifier(col)
	}
	if len(safeColumns) == 0 {
		safeColumns = []string{"*"}
	}
	return &QueryBuilder{
		table:   safeTable,
		columns: safeColumns,
	}
}

// LeftJoin: 테이블명 이스케이프 처리
func (qb *QueryBuilder) LeftJoin(joinTable string, onCondition string) *QueryBuilder {
	safeTable := escapeIdentifier(joinTable)
	joinStatement := fmt.Sprintf("LEFT JOIN %s ON %s", safeTable, onCondition)
	qb.joins = append(qb.joins, joinStatement)
	return qb
}

// Where: 안전한 파라미터 바인딩 처리
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	baseIndex := len(qb.args) + 1
	updatedCondition := replacePlaceholders(condition, baseIndex)

	qb.conditions = append(qb.conditions, updatedCondition)
	qb.args = append(qb.args, args...)
	return qb
}

// OrderBy: 컬럼명 이스케이프 및 방향 검증
func (qb *QueryBuilder) OrderBy(column string, direction string) *QueryBuilder {
	direction = validateDirection(direction)
	safeCol := escapeIdentifier(column)
	qb.orderBy = fmt.Sprintf("%s %s", safeCol, direction)
	return qb
}

// DynamicOrderBy: 안전한 동적 정렬 처리
func (qb *QueryBuilder) DynamicOrderBy(dynamicColumn, defaultColumn, direction string) *QueryBuilder {
	direction = validateDirection(direction)
	targetColumn := defaultColumn
	if dynamicColumn != "" {
		targetColumn = dynamicColumn
	}
	safeCol := escapeIdentifier(targetColumn)
	qb.orderBy = fmt.Sprintf("%s %s", safeCol, direction)
	return qb
}

// Build: 최종 쿼리 생성 (나머지 메소드는 동일)

// Helper functions
func escapeIdentifier(name string) string {
	if name == "*" {
		return name
	}
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(name, `"`, `""`))
}

func validateDirection(direction string) string {
	direction = strings.ToUpper(direction)
	if direction != "ASC" && direction != "DESC" {
		return "DESC"
	}
	return direction
}

func replacePlaceholders(condition string, startIdx int) string {
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
