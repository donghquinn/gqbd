package gqbd

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// dialect captures everything that differs between the supported SQL databases.
type dialect struct {
	quote           byte // identifier quote character; 0 leaves identifiers unescaped
	numPrefix       byte // prefix of numbered placeholders ($1 or :1); 0 writes ?
	returning       bool // supports INSERT ... RETURNING
	backslashEscape bool // backslash is an escape character inside quoted identifiers
	alterUpdate     bool // UPDATE is written as ALTER TABLE ... UPDATE (a mutation)
	requireWhere    bool // UPDATE and DELETE must have a WHERE clause
	chClauses       bool // supports ClickHouse's FINAL, PREWHERE and SETTINGS
	noTableAS       bool // table aliases cannot be introduced with AS
	fetchFirst      bool // OFFSET n ROWS FETCH NEXT m ROWS ONLY instead of LIMIT/OFFSET
}

var (
	postgresDialect   = &dialect{quote: '"', numPrefix: '$', returning: true}
	mysqlDialect      = &dialect{quote: '`'}
	sqliteDialect     = &dialect{quote: '"', returning: true}
	clickhouseDialect = &dialect{quote: '`', backslashEscape: true, alterUpdate: true, requireWhere: true, chClauses: true}
	oracleDialect     = &dialect{quote: '"', numPrefix: ':', noTableAS: true, fetchFirst: true}
	// unknownDialect only numbers placeholders, for ReplacePlaceholders and
	// GeneratePlaceholders. It cannot escape identifiers, so the builders and
	// EscapeIdentifier reject unknown database types instead of using it.
	unknownDialect = &dialect{}
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
	case Oracle:
		return oracleDialect
	default:
		return unknownDialect
	}
}

// ident is a parsed identifier: [prefix.]name[sep alias]. Builders keep idents
// and write them straight into the query at Build time, so escaping a column or
// table does not allocate.
type ident struct {
	prefix   string // unquoted qualifier, such as a table alias or schema
	name     string
	alias    string
	as       bool // the alias is introduced with AS
	verbatim bool // name is written unchanged: "*" or a function call
}

// parse validates an identifier, understanding "*", function calls,
// "table alias", "table AS alias" and "prefix.name" forms.
func (d *dialect) parse(name string) (ident, error) {
	if name == "*" {
		return ident{name: name, verbatim: true}, nil
	}
	if name == "" {
		return ident{}, fmt.Errorf("empty identifier not allowed")
	}

	// Function calls (e.g. "COUNT(id)") are passed through unescaped.
	if strings.IndexByte(name, '(') >= 0 && strings.IndexByte(name, ')') >= 0 {
		return ident{name: name, verbatim: true}, nil
	}

	if strings.IndexByte(name, ' ') >= 0 {
		switch parts, n := fields3(name); {
		case n == 3 && strings.EqualFold(parts[1], "AS"):
			return parseAliased(parts[0], true, parts[2])
		case n == 2:
			return parseAliased(parts[0], false, parts[1])
		}
	}

	return parseQualified(name)
}

// parseTable is parse for table names, dropping AS before the alias in
// dialects that do not accept it there.
func (d *dialect) parseTable(name string) (ident, error) {
	id, err := d.parse(name)
	if d.noTableAS {
		id.as = false
	}
	return id, err
}

func parseAliased(name string, as bool, alias string) (ident, error) {
	if !isPlainIdent(alias) {
		return ident{}, fmt.Errorf("invalid alias %q", alias)
	}
	id, err := parseQualified(name)
	if err != nil {
		return ident{}, err
	}
	id.as, id.alias = as, alias
	return id, nil
}

// parseQualified splits off the prefix of a dotted name ("t.column" or
// "schema.table"); only the last segment is quoted, so the prefix can refer to
// an alias.
func parseQualified(name string) (ident, error) {
	dot := strings.LastIndexByte(name, '.')
	if dot < 0 {
		return ident{name: name}, nil
	}
	prefix, last := name[:dot], name[dot+1:]
	if last == "" {
		return ident{}, fmt.Errorf("invalid identifier %q", name)
	}
	for rest := prefix; ; {
		seg, tail, more := strings.Cut(rest, ".")
		if !isPlainIdent(seg) {
			return ident{}, fmt.Errorf("invalid identifier prefix %q", prefix)
		}
		if !more {
			break
		}
		rest = tail
	}
	return ident{prefix: prefix, name: last}, nil
}

// fields3 splits s around white space like strings.Fields without allocating.
// It returns the first three fields and the field count, capped at 4.
func fields3(s string) (f [3]string, n int) {
	for i := 0; i < len(s) && n < 4; {
		r, size := utf8.DecodeRuneInString(s[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		start := i
		for i < len(s) {
			r, size = utf8.DecodeRuneInString(s[i:])
			if unicode.IsSpace(r) {
				break
			}
			i += size
		}
		if n < len(f) {
			f[n] = s[start:i] // #nosec G602 -- 0 <= start <= i <= len(s) by construction
		}
		n++
	}
	return f, n
}

// size is an estimate of the escaped length of id, for sizing buffers.
func (id ident) size() int {
	return len(id.prefix) + len(id.name) + len(id.alias) + 7
}

// writeIdent writes id, wrapping its name in the dialect's quote character.
// Embedded quote characters are doubled (and backslashes too where they are
// escape characters) so the name cannot break out of its quotes.
func (d *dialect) writeIdent(b *strings.Builder, id ident) {
	if id.prefix != "" {
		b.WriteString(id.prefix)
		b.WriteByte('.')
	}
	if id.verbatim || d.quote == 0 {
		b.WriteString(id.name)
	} else {
		b.WriteByte(d.quote)
		start := 0
		for i := 0; i < len(id.name); i++ {
			if c := id.name[i]; c == d.quote || c == '\\' && d.backslashEscape {
				b.WriteString(id.name[start : i+1])
				b.WriteByte(c)
				start = i + 1
			}
		}
		b.WriteString(id.name[start:])
		b.WriteByte(d.quote)
	}
	if id.alias != "" {
		if id.as {
			b.WriteString(" AS ")
		} else {
			b.WriteByte(' ')
		}
		b.WriteString(id.alias)
	}
}

// escape returns the escaped form of an identifier as a string.
func (d *dialect) escape(name string) (string, error) {
	id, err := d.parse(name)
	if err != nil {
		return "", err
	}
	if id.verbatim {
		return id.name, nil
	}
	var b strings.Builder
	b.Grow(id.size())
	d.writeIdent(&b, id)
	return b.String(), nil
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

func (w *sqlWriter) ident(id ident) {
	w.d.writeIdent(&w.Builder, id)
}

func (w *sqlWriter) idents(ids []ident, sep string) {
	for i, id := range ids {
		if i > 0 {
			w.WriteString(sep)
		}
		w.ident(id)
	}
}

func (w *sqlWriter) placeholder() {
	w.n++
	if w.d.numPrefix == 0 {
		w.WriteByte('?')
		return
	}
	w.WriteByte(w.d.numPrefix)
	w.Write(strconv.AppendInt(w.scratch[:0], int64(w.n), 10))
}

// cond writes a user-supplied condition, converting each ? into the dialect's
// placeholder. A ? inside a '...', "..." or `...` quoted string or name is left
// alone, and ?? is written as a literal ?, for operators such as PostgreSQL's
// jsonb ?, ?| and ?&.
func (w *sqlWriter) cond(s string) {
	var quote byte // the open quote character, or 0
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == '\\' && w.d.backslashEscape {
				i++ // skip the escaped character
			} else if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"' || c == '`':
			quote = c
		case c == '?':
			w.WriteString(s[start:i])
			if i+1 < len(s) && s[i+1] == '?' {
				w.WriteByte('?')
				i++
			} else {
				w.placeholder()
			}
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

func identsLen(ids []ident) int {
	n := 0
	for _, id := range ids {
		n += id.size() + 2
	}
	return n
}
