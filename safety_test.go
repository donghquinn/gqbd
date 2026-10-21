package gqbd_test

import (
	"reflect"
	"sync"
	"testing"

	"github.com/donghquinn/gqbd"
)

func TestConditionPlaceholders(t *testing.T) {
	for _, tc := range []struct {
		name   string
		dbType gqbd.DBType
		cond   string
		want   string
	}{
		{"jsonb operators", gqbd.PostgreSQL, "data ?? ? AND tags ??| ? AND tags ??& ?", "data ? $1 AND tags ?| $2 AND tags ?& $3"},
		{"single quotes", gqbd.PostgreSQL, "a = '?' AND b = ?", "a = '?' AND b = $1"},
		{"doubled single quote", gqbd.PostgreSQL, "a = 'it''s ?' AND b = ?", "a = 'it''s ?' AND b = $1"},
		{"double-quoted name", gqbd.PostgreSQL, `"what?" = ?`, `"what?" = $1`},
		{"backquoted name", gqbd.Mysql, "`what?` = ?", "`what?` = ?"},
		{"oracle", gqbd.Oracle, `"A?" = ? AND b ?? 1`, `"A?" = :1 AND b ? 1`},
		{"clickhouse backslash escape", gqbd.ClickHouse, `a = 'it\'s ?' AND b = ?`, `a = 'it\'s ?' AND b = ?`},
		{"backslash outside clickhouse", gqbd.PostgreSQL, `a = '\' AND b = ?`, `a = '\' AND b = $1`},
		{"question at end", gqbd.PostgreSQL, "a ??", "a ?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, _, err := gqbd.BuildSelect(tc.dbType, "t").Where(tc.cond).Build()
			if err != nil {
				t.Fatal(err)
			}
			want := "SELECT * FROM " + mustEscape(t, tc.dbType, "t") + " WHERE " + tc.want
			if q != want {
				t.Errorf("got  %s\nwant %s", q, want)
			}
		})
	}
}

func mustEscape(t *testing.T, dbType gqbd.DBType, name string) string {
	t.Helper()
	s, err := gqbd.EscapeIdentifier(dbType, name)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCloneIsIndependent(t *testing.T) {
	base := gqbd.BuildSelect(gqbd.ClickHouse, "orders", "id").
		Where("tenant_id = ?", 7).
		Prewhere("day = ?", "2026-10-21").
		Setting("max_threads", 2)
	open := base.Clone().Where("status = ?", "open").Prewhere("x = ?", 1).Setting("a", 1).
		LeftJoin("users u", "u.id = orders.user_id").GroupBy("id").Having("COUNT(*) > ?", 1).Aggregate("COUNT", "id")
	closed := base.Clone().Where("status = ?", "closed")

	q, args, err := base.Build()
	if err != nil {
		t.Fatal(err)
	}
	if want := "SELECT `id` FROM `orders` PREWHERE day = ? WHERE tenant_id = ? SETTINGS max_threads = 2"; q != want {
		t.Errorf("base changed:\ngot  %s\nwant %s", q, want)
	}
	if !reflect.DeepEqual(args, []interface{}{"2026-10-21", 7}) {
		t.Errorf("base args = %v", args)
	}
	q, args, _ = closed.Build()
	if want := "SELECT `id` FROM `orders` PREWHERE day = ? WHERE tenant_id = ? AND status = ? SETTINGS max_threads = 2"; q != want {
		t.Errorf("closed:\ngot  %s\nwant %s", q, want)
	}
	if !reflect.DeepEqual(args, []interface{}{"2026-10-21", 7, "closed"}) {
		t.Errorf("closed args = %v", args)
	}
	if _, _, err := open.Build(); err != nil {
		t.Fatal(err)
	}
}

func TestCloneConcurrentUse(t *testing.T) {
	// Run with -race: each goroutine extends its own clone of a shared base.
	base := gqbd.BuildSelect(gqbd.PostgreSQL, "orders").Where("tenant_id = ?", 7)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			q, args, err := base.Clone().Where("id = ?", i).Limit(10).Build()
			if err != nil || q != `SELECT * FROM "orders" WHERE tenant_id = $1 AND id = $2 LIMIT $3` || len(args) != 3 {
				t.Errorf("got %q %v %v", q, args, err)
			}
		}(i)
	}
	wg.Wait()
}
