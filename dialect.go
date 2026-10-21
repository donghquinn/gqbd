package gqbd

import (
	"fmt"
	"strconv"
	"strings"
)

// dialect captures everything that differs between the supported SQL databases.
type dialect struct {
	quote           byte // identifier quote character; 0 leaves identifiers unescaped
	numbered        bool // $1, $2, ... placeholders instead of ?
	returning       bool // supports INSERT ... RETURNING
	backslashEscape bool // backslash is an escape character inside quoted identifiers
	alterUpdate     bool // UPDATE is written as ALTER TABLE ... UPDATE (a mutation)
	requireWhere    bool // UPDATE and DELETE must have a WHERE clause
	chClauses       bool // supports ClickHouse's FINAL, PREWHERE and SETTINGS
}

var (
	postgresDialect   = &dialect{quote: '"', numbered: true, returning: true}
	mysqlDialect      = &dialect{quote: '`'}
	sqliteDialect     = &dialect{quote: '"', returning: true}
	clickhouseDialect = &dialect{quote: '`', backslashEscape: true, alterUpdate: true, requireWhere: true, chClauses: true}
	unknownDialect    = &dialect{}
)

func dialectFor(dbType DBType) *dialect {
	switch dbType {
	case PostgreSQL:
		return postgresDialect
	case MariaDB, Mysql:
		return mysqlDialect
	case SQLite:
		return sqliteDialect
	case ClickHouse:
		return clickhouseDialect
	default:
		return unknownDialect
	}
}

// quoteName wraps a single identifier in the dialect's quote character, doubling
// any embedded quote characters (and escaping backslashes where they are escape
// characters) so the identifier cannot break out of its quotes.
func (d *dialect) quoteName(name string) string {
	if d.quote == 0 {
		return name
	}
	q := string(d.quote)
	if d.backslashEscape && strings.IndexByte(name, '\\') >= 0 {
		name = strings.ReplaceAll(name, `\`, `\\`)
	}
	if strings.IndexByte(name, d.quote) >= 0 {
		name = strings.ReplaceAll(name, q, q+q)
	}
	return q + name + q
}

// escape quotes an identifier, understanding "*", function calls, "table alias",
// "table AS alias" and "prefix.name" forms.
func (d *dialect) escape(name string) (string, error) {
	if name == "*" {
		return name, nil
	}
	if name == "" {
		return "", fmt.Errorf("empty identifier not allowed")
	}

	// Function calls (e.g. "COUNT(id)") are passed through unescaped.
	if strings.IndexByte(name, '(') >= 0 && strings.IndexByte(name, ')') >= 0 {
		return name, nil
	}

	if strings.IndexByte(name, ' ') >= 0 {
		parts := strings.Fields(name)
		switch {
		case len(parts) == 3 && strings.EqualFold(parts[1], "AS"):
			return d.escapeAliased(parts[0], " AS ", parts[2])
		case len(parts) == 2:
			return d.escapeAliased(parts[0], " ", parts[1])
		}
	}

	return d.escapeQualified(name)
}

func (d *dialect) escapeAliased(name, sep, alias string) (string, error) {
	if !isPlainIdent(alias) {
		return "", fmt.Errorf("invalid alias %q", alias)
	}
	escaped, err := d.escapeQualified(name)
	if err != nil {
		return "", err
	}
	return escaped + sep + alias, nil
}

// escapeQualified quotes the last segment of a dotted name ("t.column" or
// "schema.table"), leaving the prefix unquoted so it can refer to an alias.
func (d *dialect) escapeQualified(name string) (string, error) {
	dot := strings.LastIndexByte(name, '.')
	if dot < 0 {
		return d.quoteName(name), nil
	}
	prefix, last := name[:dot], name[dot+1:]
	if last == "" {
		return "", fmt.Errorf("invalid identifier %q", name)
	}
	for _, p := range strings.Split(prefix, ".") {
		if !isPlainIdent(p) {
			return "", fmt.Errorf("invalid identifier prefix %q", prefix)
		}
	}
	return prefix + "." + d.quoteName(last), nil
}

// isPlainIdent reports whether s consists only of ASCII letters, digits and underscores.
func isPlainIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// sqlWriter renders SQL text, numbering placeholders in the order they appear.
type sqlWriter struct {
	strings.Builder
	d       *dialect
	n       int
	scratch [20]byte
}

func newSQLWriter(d *dialect, size int) *sqlWriter {
	w := &sqlWriter{d: d}
	w.Grow(size)
	return w
}

func (w *sqlWriter) placeholder() {
	w.n++
	if !w.d.numbered {
		w.WriteByte('?')
		return
	}
	w.WriteByte('$')
	w.Write(strconv.AppendInt(w.scratch[:0], int64(w.n), 10))
}

// cond writes a user-supplied condition, converting each ? outside of single-quoted
// string literals into the dialect's placeholder.
func (w *sqlWriter) cond(s string) {
	inQuote := false
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			inQuote = !inQuote
		case '?':
			if inQuote {
				continue
			}
			w.WriteString(s[start:i])
			w.placeholder()
			start = i + 1
		}
	}
	w.WriteString(s[start:])
}

func (w *sqlWriter) list(items []string, sep string) {
	for i, s := range items {
		if i > 0 {
			w.WriteString(sep)
		}
		w.WriteString(s)
	}
}

func (w *sqlWriter) condList(items []string, sep string) {
	for i, s := range items {
		if i > 0 {
			w.WriteString(sep)
		}
		w.cond(s)
	}
}

func (w *sqlWriter) placeholders(count int) {
	for i := 0; i < count; i++ {
		if i > 0 {
			w.WriteString(", ")
		}
		w.placeholder()
	}
}

func totalLen(items []string) int {
	n := 0
	for _, s := range items {
		n += len(s) + 2
	}
	return n
}
