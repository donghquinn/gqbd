package gqbd

import (
	"fmt"
	"sort"
)

func (qb *QueryBuilder) buildSelect() (string, []interface{}, error) {
	ch := qb.clickhouse()
	size := 48 + qb.table.size() + qb.orderCol.size() + identsLen(qb.columns) + identsLen(qb.groupBy) +
		totalLen(qb.where) + totalLen(qb.having) + totalLen(ch.prewhere) + totalLen(ch.settings)
	for _, j := range qb.joins {
		size += len(j.kind) + j.table.size() + len(j.on) + 5
	}
	w := newSQLWriter(qb.d, size)

	w.WriteString("SELECT ")
	if qb.distinct {
		w.WriteString("DISTINCT ")
	}
	w.idents(qb.columns, ", ")
	w.WriteString(" FROM ")
	w.ident(qb.table)
	if ch.final {
		w.WriteString(" FINAL")
	}
	for _, j := range qb.joins {
		w.WriteByte(' ')
		w.WriteString(j.kind)
		w.ident(j.table)
		w.WriteString(" ON ")
		w.WriteString(j.on)
	}
	if len(ch.prewhere) > 0 {
		w.WriteString(" PREWHERE ")
		w.condList(ch.prewhere, " AND ")
	}
	qb.writeWhere(w)
	if len(qb.groupBy) > 0 {
		w.WriteString(" GROUP BY ")
		w.idents(qb.groupBy, ", ")
	}
	if len(qb.having) > 0 {
		w.WriteString(" HAVING ")
		w.condList(qb.having, " AND ")
	}
	if qb.orderCol.name != "" {
		w.WriteString(" ORDER BY ")
		w.ident(qb.orderCol)
		if qb.orderDesc {
			w.WriteString(" DESC")
		} else {
			w.WriteString(" ASC")
		}
	}

	args := make([]interface{}, 0, len(ch.prewhereArgs)+len(qb.whereArgs)+len(qb.havingArgs)+2)
	args = append(args, ch.prewhereArgs...)
	args = append(args, qb.whereArgs...)
	args = append(args, qb.havingArgs...)
	args = qb.writeLimit(w, args)
	qb.writeSettings(w)
	return w.String(), args, nil
}

func (qb *QueryBuilder) buildInsert() (string, []interface{}, error) {
	if qb.data == nil {
		return "", nil, fmt.Errorf("no data provided for INSERT")
	}
	keys := sortedKeys(qb.data)
	w := newSQLWriter(qb.d, 32+qb.table.size()+len(qb.returning)+totalLen(keys)+8*len(keys)+totalLen(qb.clickhouse().settings))
	args := make([]interface{}, len(keys))

	w.WriteString("INSERT INTO ")
	w.ident(qb.table)
	w.WriteString(" (")
	for i, key := range keys {
		col, err := qb.d.parse(key)
		if err != nil {
			return "", nil, err
		}
		if i > 0 {
			w.WriteString(", ")
		}
		w.ident(col)
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
	w := newSQLWriter(qb.d, 32+qb.table.size()+totalLen(keys)+10*len(keys)+totalLen(qb.where)+totalLen(qb.clickhouse().settings))
	args := make([]interface{}, len(keys), len(keys)+len(qb.whereArgs))

	if qb.d.alterUpdate {
		w.WriteString("ALTER TABLE ")
		w.ident(qb.table)
		w.WriteString(" UPDATE ")
	} else {
		w.WriteString("UPDATE ")
		w.ident(qb.table)
		w.WriteString(" SET ")
	}
	for i, key := range keys {
		col, err := qb.d.parse(key)
		if err != nil {
			return "", nil, err
		}
		if i > 0 {
			w.WriteString(", ")
		}
		w.ident(col)
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
	w := newSQLWriter(qb.d, 20+qb.table.size()+totalLen(qb.where)+totalLen(qb.clickhouse().settings))
	w.WriteString("DELETE FROM ")
	w.ident(qb.table)
	qb.writeWhere(w)
	qb.writeSettings(w)
	return w.String(), append([]interface{}(nil), qb.whereArgs...), nil
}

// writeLimit writes LIMIT/OFFSET, or OFFSET ... ROWS FETCH NEXT ... ROWS ONLY
// where LIMIT is not supported, appending the bound values in query order.
func (qb *QueryBuilder) writeLimit(w *sqlWriter, args []interface{}) []interface{} {
	if qb.d.fetchFirst {
		if qb.offset > 0 {
			w.WriteString(" OFFSET ")
			w.placeholder()
			w.WriteString(" ROWS")
			args = append(args, qb.offset)
		}
		if qb.limit > 0 {
			w.WriteString(" FETCH NEXT ")
			w.placeholder()
			w.WriteString(" ROWS ONLY")
			args = append(args, qb.limit)
		}
		return args
	}
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
	return args
}

func (qb *QueryBuilder) checkRequiredWhere(op string) error {
	if qb.d.requireWhere && len(qb.where) == 0 {
		return fmt.Errorf("%s requires a WHERE clause for %s; use Where(\"1 = 1\") to affect every row", qb.dbType, op)
	}
	return nil
}

func (qb *QueryBuilder) writeSettings(w *sqlWriter) {
	if settings := qb.clickhouse().settings; len(settings) > 0 {
		w.WriteString(" SETTINGS ")
		w.list(settings, ", ")
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
