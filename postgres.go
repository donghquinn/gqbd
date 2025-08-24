package gqbd

import (
	"fmt"
	"sort"
	"strings"
)

func (qb *QueryBuilder) buildPostgreSQLSelect() (string, []interface{}, error) {
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
	if qb.limit > 0 {
		placeholder := fmt.Sprintf("$%d", len(qb.args)+1)
		queryBuilder.WriteString(" LIMIT " + placeholder)
		qb.args = append(qb.args, qb.limit)
	}
	if qb.offset > 0 {
		placeholder := fmt.Sprintf("$%d", len(qb.args)+1)
		queryBuilder.WriteString(" OFFSET " + placeholder)
		qb.args = append(qb.args, qb.offset)
	}
	return queryBuilder.String(), qb.args, nil
}

func (qb *QueryBuilder) buildPostgreSQLInsert() (string, []interface{}, error) {
	if qb.data == nil {
		return "", nil, fmt.Errorf("no data provided for INSERT")
	}
	var cols []string
	var placeholders []string
	var args []interface{}

	for col, val := range qb.data {
		safeCol, err := EscapeIdentifier(qb.dbType, col)
		if err != nil {
			return "", nil, err
		}
		cols = append(cols, safeCol)
		placeholders = append(placeholders, "?")
		args = append(args, val)
	}

	placeholdersStr := strings.Join(placeholders, ", ")
	placeholdersStr = ReplacePlaceholders(qb.dbType, placeholdersStr, 1)

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", qb.table, strings.Join(cols, ", "), placeholdersStr)
	if qb.returning != "" {
		query += " RETURNING " + qb.returning
	}

	return query, args, nil
}

func (qb *QueryBuilder) buildPostgreSQLUpdate() (string, []interface{}, error) {
	if qb.data == nil {
		return "", nil, fmt.Errorf("no data provided for UPDATE")
	}
	var setClauses []string
	var updateArgs []interface{}

	var keys []string
	for key := range qb.data {
		keys = append(keys, key)
	}
	
	sort.Strings(keys)
	for _, key := range keys {
		safeCol, err := EscapeIdentifier(qb.dbType, key)
		if err != nil {
			return "", nil, err
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", safeCol))
		updateArgs = append(updateArgs, qb.data[key])
	}

	setClausesStr := strings.Join(setClauses, ", ")
	setClausesStr = ReplacePlaceholders(qb.dbType, setClausesStr, 1)

	query := fmt.Sprintf("UPDATE %s SET %s", qb.table, setClausesStr)

	allArgs := updateArgs
	if len(qb.conditions) > 0 {
		whereConditions := make([]string, len(qb.conditions))
		for i, condition := range qb.conditions {
			whereConditions[i] = shiftPlaceholders(condition, len(updateArgs))
		}
		query += " WHERE " + strings.Join(whereConditions, " AND ")
		allArgs = append(allArgs, qb.args...)
	}

	return query, allArgs, nil
}

func escapePostgreSQLIdentifier(name string) (string, error) {
	return `"` + name + `"`, nil
}