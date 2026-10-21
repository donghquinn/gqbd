package gqbd

import (
	"fmt"
	"sort"
)

func (qb *QueryBuilder) buildSelect() (string, []interface{}, error) {
	w := newSQLWriter(qb.d, 32+len(qb.table)+len(qb.orderBy)+
		totalLen(qb.columns)+totalLen(qb.joins)+totalLen(qb.where)+totalLen(qb.groupBy)+totalLen(qb.having))

	w.WriteString("SELECT ")
	if qb.distinct {
		w.WriteString("DISTINCT ")
	}
	w.list(qb.columns, ", ")
	w.WriteString(" FROM ")
	w.WriteString(qb.table)
	if len(qb.joins) > 0 {
		w.WriteByte(' ')
		w.list(qb.joins, " ")
	}
	qb.writeWhere(w)
	if len(qb.groupBy) > 0 {
		w.WriteString(" GROUP BY ")
		w.list(qb.groupBy, ", ")
	}
	if len(qb.having) > 0 {
		w.WriteString(" HAVING ")
		w.condList(qb.having, " AND ")
	}
	if qb.orderBy != "" {
		w.WriteString(" ORDER BY ")
		w.WriteString(qb.orderBy)
	}

	args := make([]interface{}, 0, len(qb.whereArgs)+len(qb.havingArgs)+2)
	args = append(args, qb.whereArgs...)
	args = append(args, qb.havingArgs...)
	if qb.limit > 0 {
		w.WriteString(" LIMIT ")
		w.placeholder()
		args = append(args, qb.limit)
	}
	if qb.offset > 0 {
		w.WriteString(" OFFSET ")
		w.placeholder()
		args = append(args, qb.offset)
	}
	return w.String(), args, nil
}

func (qb *QueryBuilder) buildInsert() (string, []interface{}, error) {
	if qb.data == nil {
		return "", nil, fmt.Errorf("no data provided for INSERT")
	}
	keys := sortedKeys(qb.data)
	w := newSQLWriter(qb.d, 32+len(qb.table)+len(qb.returning)+totalLen(keys)+8*len(keys))
	args := make([]interface{}, len(keys))

	w.WriteString("INSERT INTO ")
	w.WriteString(qb.table)
	w.WriteString(" (")
	for i, key := range keys {
		safeCol, err := qb.d.escape(key)
		if err != nil {
			return "", nil, err
		}
		if i > 0 {
			w.WriteString(", ")
		}
		w.WriteString(safeCol)
		args[i] = qb.data[key]
	}
	w.WriteString(") VALUES (")
	w.placeholders(len(keys))
	w.WriteByte(')')
	if qb.returning != "" && qb.d.returning {
		w.WriteString(" RETURNING ")
		w.WriteString(qb.returning)
	}
	return w.String(), args, nil
}

func (qb *QueryBuilder) buildUpdate() (string, []interface{}, error) {
	if qb.data == nil {
		return "", nil, fmt.Errorf("no data provided for UPDATE")
	}
	keys := sortedKeys(qb.data)
	w := newSQLWriter(qb.d, 32+len(qb.table)+totalLen(keys)+10*len(keys)+totalLen(qb.where))
	args := make([]interface{}, len(keys), len(keys)+len(qb.whereArgs))

	w.WriteString("UPDATE ")
	w.WriteString(qb.table)
	w.WriteString(" SET ")
	for i, key := range keys {
		safeCol, err := qb.d.escape(key)
		if err != nil {
			return "", nil, err
		}
		if i > 0 {
			w.WriteString(", ")
		}
		w.WriteString(safeCol)
		w.WriteString(" = ")
		w.placeholder()
		args[i] = qb.data[key]
	}
	qb.writeWhere(w)
	return w.String(), append(args, qb.whereArgs...), nil
}

func (qb *QueryBuilder) buildDelete() (string, []interface{}, error) {
	w := newSQLWriter(qb.d, 20+len(qb.table)+totalLen(qb.where))
	w.WriteString("DELETE FROM ")
	w.WriteString(qb.table)
	qb.writeWhere(w)
	return w.String(), append([]interface{}(nil), qb.whereArgs...), nil
}

func (qb *QueryBuilder) writeWhere(w *sqlWriter) {
	if len(qb.where) > 0 {
		w.WriteString(" WHERE ")
		w.condList(qb.where, " AND ")
	}
}

func sortedKeys(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
