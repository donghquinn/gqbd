package gqbd_test

import (
	"reflect"
	"testing"

	"github.com/donghquinn/gqbd"
)

func TestBuildIsRepeatable(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users").Where("a = ?", 1).Limit(10).Offset(5)
	q1, a1, _ := qb.Build()
	q2, a2, _ := qb.Build()
	if q1 != q2 || !reflect.DeepEqual(a1, a2) {
		t.Errorf("Build() not repeatable:\n%s %v\n%s %v", q1, a1, q2, a2)
	}
	if want := `SELECT * FROM "users" WHERE a = $1 LIMIT $2 OFFSET $3`; q1 != want {
		t.Errorf("got %s, want %s", q1, want)
	}
}

func TestArgsFollowQueryOrder(t *testing.T) {
	for _, tc := range []struct {
		dbType gqbd.DBType
		want   string
	}{
		{gqbd.Mysql, "SELECT `g` FROM `t` WHERE x = ? GROUP BY `g` HAVING COUNT(*) > ?"},
		{gqbd.PostgreSQL, `SELECT "g" FROM "t" WHERE x = $1 GROUP BY "g" HAVING COUNT(*) > $2`},
	} {
		// Having is called before Where; args must still match placeholder order.
		q, args, err := gqbd.BuildSelect(tc.dbType, "t", "g").
			GroupBy("g").Having("COUNT(*) > ?", 5).Where("x = ?", "X").Build()
		if err != nil {
			t.Fatal(err)
		}
		if q != tc.want {
			t.Errorf("got %s, want %s", q, tc.want)
		}
		if !reflect.DeepEqual(args, []interface{}{"X", 5}) {
			t.Errorf("%s: args = %v, want [X 5]", tc.dbType, args)
		}
	}
}

func TestEscapeIdentifierQuoteInjection(t *testing.T) {
	for _, tc := range []struct {
		dbType gqbd.DBType
		in     string
		want   string
	}{
		{gqbd.PostgreSQL, `name"; DROP TABLE users; --`, `"name""; DROP TABLE users; --"`},
		{gqbd.MariaDB, "name`; DROP TABLE users; --", "`name``; DROP TABLE users; --`"},
	} {
		got, err := gqbd.EscapeIdentifier(tc.dbType, tc.in)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("got %s, want %s", got, tc.want)
		}
	}
}

func TestEscapeIdentifierInvalidAliasAndPrefix(t *testing.T) {
	for _, in := range []string{"users u;DROP", "users AS u--", "x;y.col", "t."} {
		if _, err := gqbd.EscapeIdentifier(gqbd.PostgreSQL, in); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
}

func TestEscapeIdentifierSchemaQualified(t *testing.T) {
	got, err := gqbd.EscapeIdentifier(gqbd.PostgreSQL, "public.users u")
	if err != nil {
		t.Fatal(err)
	}
	if want := `public."users" u`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestAggregateInvalidFunction(t *testing.T) {
	_, _, err := gqbd.BuildSelect(gqbd.PostgreSQL, "t").Aggregate("COUNT(1); --", "id").Build()
	if err == nil {
		t.Error("expected error for invalid aggregate function")
	}
}

func TestWhereInEmpty(t *testing.T) {
	_, _, err := gqbd.BuildSelect(gqbd.PostgreSQL, "t").WhereIn("id", nil).Build()
	if err == nil {
		t.Error("expected error for empty WhereIn values")
	}
}

func TestPlaceholderInStringLiteralIsKept(t *testing.T) {
	q, _, _ := gqbd.BuildSelect(gqbd.PostgreSQL, "t").Where("note <> 'why?' AND id = ?", 1).Build()
	if want := `SELECT * FROM "t" WHERE note <> 'why?' AND id = $1`; q != want {
		t.Errorf("got %s, want %s", q, want)
	}
}

func TestPostgreSQLUpdateNumbering(t *testing.T) {
	q, args, err := gqbd.BuildUpdate(gqbd.PostgreSQL, "users").
		Set(map[string]interface{}{"b": 2, "a": 1}).
		Where("id = ?", 9).
		WhereIn("role", []interface{}{"x", "y"}).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	if want := `UPDATE "users" SET "a" = $1, "b" = $2 WHERE id = $3 AND "role" IN ($4, $5)`; q != want {
		t.Errorf("got %s, want %s", q, want)
	}
	if !reflect.DeepEqual(args, []interface{}{1, 2, 9, "x", "y"}) {
		t.Errorf("args = %v", args)
	}
}

func TestEarlierErrorNotOverwritten(t *testing.T) {
	_, _, err := gqbd.BuildSelect(gqbd.PostgreSQL, "").Values(map[string]interface{}{"a": 1}).Build()
	if err == nil || err.Error() != "empty identifier not allowed" {
		t.Errorf("expected original error, got %v", err)
	}
}

func TestPlaceholderHelpers(t *testing.T) {
	if got := gqbd.ReplacePlaceholders(gqbd.PostgreSQL, "a = ? AND b = ?", 3); got != "a = $3 AND b = $4" {
		t.Errorf("ReplacePlaceholders = %s", got)
	}
	if got := gqbd.ReplacePlaceholders(gqbd.Mysql, "a = ?", 3); got != "a = ?" {
		t.Errorf("ReplacePlaceholders = %s", got)
	}
	if got := gqbd.GeneratePlaceholders(gqbd.PostgreSQL, 2, 3); got != "$2, $3, $4" {
		t.Errorf("GeneratePlaceholders = %s", got)
	}
	if got := gqbd.GeneratePlaceholders(gqbd.SQLite, 1, 2); got != "?, ?" {
		t.Errorf("GeneratePlaceholders = %s", got)
	}
}

func TestPostgreSQLConnectionStringQuoting(t *testing.T) {
	dsn := gqbd.BuildConnectionString(gqbd.PostgreSQL, gqbd.DBConfig{Host: "h", User: "u", Password: `p w'\`})
	if want := `host=h user=u password='p w\'\\' sslmode=disable`; dsn != want {
		t.Errorf("got %s, want %s", dsn, want)
	}
}
