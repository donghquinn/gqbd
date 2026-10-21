package gqbd

import (
	"fmt"
	"sort"
)

func (qb *QueryBuilder) buildSelect() (string, []interface{}, error) {
	w := newSQLWriter(qb.d, 32+len(qb.table)+len(qb.orderBy)+
		totalLen(qb.columns)+totalLen(qb.joins)+totalLen(qb.where)+totalLen(qb.groupBy)+totalLen(qb.having)+
		totalLen(qb.prewhere)+totalLen(qb.settings))

	w.WriteString("SELECT ")
	if qb.distinct {
		w.WriteString("DISTINCT ")
	}
	w.list(qb.columns, ", ")
	w.WriteString(" FROM ")
	w.WriteString(qb.table)
	if qb.final {
		w.WriteString(" FINAL")
	}
	if len(qb.joins) > 0 {
		w.WriteByte(' ')
		w.list(qb.joins, " ")
	}
	if len(qb.prewhere) > 0 {
		w.WriteString(" PREWHERE ")
		w.condList(qb.prewhere, " AND ")
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

	args := make([]interface{}, 0, len(qb.prewhereArgs)+len(qb.whereArgs)+len(qb.havingArgs)+2)
	args = append(args, qb.prewhereArgs...)
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
	qb.writeSettings(w)
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
	w.WriteByte(')')
	qb.writeSettings(w)
	w.WriteString(" VALUES (")
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
	if err := qb.checkRequiredWhere("UPDATE"); err != nil {
		return "", nil, err
	}
	keys := sortedKeys(qb.data)
	w := newSQLWriter(qb.d, 32+len(qb.table)+totalLen(keys)+10*len(keys)+totalLen(qb.where)+totalLen(qb.settings))
	args := make([]interface{}, len(keys), len(keys)+len(qb.whereArgs))

	if qb.d.alterUpdate {
		w.WriteString("ALTER TABLE ")
		w.WriteString(qb.table)
		w.WriteString(" UPDATE ")
	} else {
		w.WriteString("UPDATE ")
		w.WriteString(qb.table)
		w.WriteString(" SET ")
	}
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
	qb.writeSettings(w)
	return w.String(), append(args, qb.whereArgs...), nil
}

func (qb *QueryBuilder) buildDelete() (string, []interface{}, error) {
	if err := qb.checkRequiredWhere("DELETE"); err != nil {
		return "", nil, err
	}
	w := newSQLWriter(qb.d, 20+len(qb.table)+totalLen(qb.where)+totalLen(qb.settings))
	w.WriteString("DELETE FROM ")
	w.WriteString(qb.table)
	qb.writeWhere(w)
	qb.writeSettings(w)
	return w.String(), append([]interface{}(nil), qb.whereArgs...), nil
}

func (qb *QueryBuilder) checkRequiredWhere(op string) error {
	if qb.d.requireWhere && len(qb.where) == 0 {
		return fmt.Errorf("%s requires a WHERE clause for %s; use Where(\"1 = 1\") to affect every row", qb.dbType, op)
	}
	return nil
}

func (qb *QueryBuilder) writeSettings(w *sqlWriter) {
	if len(qb.settings) > 0 {
		w.WriteString(" SETTINGS ")
		w.list(qb.settings, ", ")
	}
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
