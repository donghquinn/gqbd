package gqbd_test

import (
	"reflect"
	"testing"

	"github.com/donghquinn/gqbd"
)

// Expected queries below were checked by running them on Oracle Database 23ai Free with go-ora.
// Names are written in upper case because the test tables were created unquoted.

func TestOracleSelect(t *testing.T) {
	q, args, err := gqbd.BuildSelect(gqbd.Oracle, "USERS AS u", "u.ID", "u.NAME", "u.LEVEL", "o.AMOUNT").
		InnerJoin("ORDERS AS o", "o.user_id = u.id").
		Where("u.status = ?", "active").
		WhereIn("u.ID", []interface{}{1, 2, 3}).
		WhereBetween("o.AMOUNT", 100, 500).
		OrderBy("u.ID", "ASC", nil).
		Limit(2).Offset(1).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	want := `SELECT u."ID", u."NAME", u."LEVEL", o."AMOUNT" FROM "USERS" u INNER JOIN "ORDERS" o ON o.user_id = u.id ` +
		`WHERE u.status = :1 AND u."ID" IN (:2, :3, :4) AND o."AMOUNT" BETWEEN :5 AND :6 ` +
		`ORDER BY u."ID" ASC OFFSET :7 ROWS FETCH NEXT :8 ROWS ONLY`
	if q != want {
		t.Errorf("got  %s\nwant %s", q, want)
	}
	// OFFSET comes before FETCH in the query, so its value comes first too.
	if !reflect.DeepEqual(args, []interface{}{"active", 1, 2, 3, 100, 500, 1, 2}) {
		t.Errorf("args = %v", args)
	}
}

func TestOracleLimitOffsetAlone(t *testing.T) {
	for _, tc := range []struct {
		name string
		qb   *gqbd.QueryBuilder
		want string
	}{
		{"limit", gqbd.BuildSelect(gqbd.Oracle, "USERS").Limit(5), `SELECT * FROM "USERS" FETCH NEXT :1 ROWS ONLY`},
		{"offset", gqbd.BuildSelect(gqbd.Oracle, "USERS").Offset(5), `SELECT * FROM "USERS" OFFSET :1 ROWS`},
		{"aggregate", gqbd.BuildSelect(gqbd.Oracle, "USERS", "STATUS").Aggregate("COUNT", "ID").GroupBy("STATUS").Having("COUNT(*) >= ?", 1).Limit(5),
			`SELECT "STATUS", COUNT("ID") FROM "USERS" GROUP BY "STATUS" HAVING COUNT(*) >= :1 FETCH NEXT :2 ROWS ONLY`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, _, err := tc.qb.Build()
			if err != nil {
				t.Fatal(err)
			}
			if q != tc.want {
				t.Errorf("got  %s\nwant %s", q, tc.want)
			}
		})
	}
}

func TestOracleInsertUpdateDelete(t *testing.T) {
	for _, tc := range []struct {
		name     string
		qb       *gqbd.QueryBuilder
		want     string
		wantArgs []interface{}
	}{
		{
			"insert",
			gqbd.BuildInsert(gqbd.Oracle, "USERS").Values(map[string]interface{}{"ID": 1, "NAME": "a"}).Returning("id"), // RETURNING ignored
			`INSERT INTO "USERS" ("ID", "NAME") VALUES (:1, :2)`,
			[]interface{}{1, "a"},
		},
		{
			"update",
			gqbd.BuildUpdate(gqbd.Oracle, "USERS").Set(map[string]interface{}{"STATUS": "gone"}).Where("id = ?", 6),
			`UPDATE "USERS" SET "STATUS" = :1 WHERE id = :2`,
			[]interface{}{"gone", 6},
		},
		{
			"delete",
			gqbd.BuildDelete(gqbd.Oracle, "ORDERS").Where("id = ?", 6),
			`DELETE FROM "ORDERS" WHERE id = :1`,
			[]interface{}{6},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, args, err := tc.qb.Build()
			if err != nil {
				t.Fatal(err)
			}
			if q != tc.want {
				t.Errorf("got  %s\nwant %s", q, tc.want)
			}
			if !reflect.DeepEqual(args, tc.wantArgs) {
				t.Errorf("args = %v", args)
			}
		})
	}
}

func TestOracleIdentifiers(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"USERS", `"USERS"`},
		{"users", `"users"`}, // kept as written; matches only a table created as "users"
		{"MyTable", `"MyTable"`},
		{"hr.EMPLOYEES", `hr."EMPLOYEES"`},
		{"USERS AS u", `"USERS" AS u`}, // column-style alias; only table aliases drop AS
		{`x"; DROP TABLE users; --`, `"x""; DROP TABLE users; --"`},
	} {
		got, err := gqbd.EscapeIdentifier(gqbd.Oracle, tc.in)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("EscapeIdentifier(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestOraclePlaceholders(t *testing.T) {
	if got := gqbd.ReplacePlaceholders(gqbd.Oracle, "a = ? AND b = '?' AND c = ?", 3); got != "a = :3 AND b = '?' AND c = :4" {
		t.Errorf("ReplacePlaceholders = %s", got)
	}
	if got := gqbd.GeneratePlaceholders(gqbd.Oracle, 1, 3); got != ":1, :2, :3" {
		t.Errorf("GeneratePlaceholders = %s", got)
	}
}

func TestOracleConnectionString(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  gqbd.DBConfig
		want string
	}{
		{"default", gqbd.DBConfig{}, "oracle://localhost/"},
		{"full", gqbd.DBConfig{Host: "db", Port: 1521, User: "app", Password: "p@ss", DBName: "FREEPDB1"}, "oracle://app:p%40ss@db:1521/FREEPDB1"},
		{"tls", gqbd.DBConfig{Host: "db", Port: 2484, DBName: "ORCL", SSLMode: "require"}, "oracle://db:2484/ORCL?ssl=true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := gqbd.BuildConnectionString(gqbd.Oracle, tc.cfg); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
