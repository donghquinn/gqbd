package gqbd_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/donghquinn/gqbd"
)

// Expected queries below were checked by running them on ClickHouse 26.9 (clickhouse local).

func TestClickHouseSelect(t *testing.T) {
	q, args, err := gqbd.BuildSelect(gqbd.ClickHouse, "events e", "e.id", "e.name", "u.country").Final().
		LeftJoin("users u", "u.id = e.id").
		Prewhere("e.status = ?", "active").
		Where("e.v >= ?", 10).
		WhereIn("e.id", []interface{}{1, 2, 3}).
		OrderBy("e.id", "ASC", nil).
		Limit(10).Offset(5).
		Setting("max_threads", 2).Setting("log_comment", `it's a \ test`).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT e.`id`, e.`name`, u.`country` FROM `events` e FINAL LEFT JOIN `users` u ON u.id = e.id " +
		"PREWHERE e.status = ? WHERE e.v >= ? AND e.`id` IN (?, ?, ?) ORDER BY e.`id` ASC LIMIT ? OFFSET ? " +
		`SETTINGS max_threads = 2, log_comment = 'it\'s a \\ test'`
	if q != want {
		t.Errorf("got  %s\nwant %s", q, want)
	}
	// PREWHERE args come first because PREWHERE comes first in the query.
	if !reflect.DeepEqual(args, []interface{}{"active", 10, 1, 2, 3, 10, 5}) {
		t.Errorf("args = %v", args)
	}
}

func TestClickHouseInsert(t *testing.T) {
	q, args, err := gqbd.BuildInsert(gqbd.ClickHouse, "events").
		Values(map[string]interface{}{"id": 1, "name": "a"}).
		Setting("async_insert", true).
		Returning("id"). // not supported by ClickHouse; ignored
		Build()
	if err != nil {
		t.Fatal(err)
	}
	if want := "INSERT INTO `events` (`id`, `name`) SETTINGS async_insert = 1 VALUES (?, ?)"; q != want {
		t.Errorf("got  %s\nwant %s", q, want)
	}
	if !reflect.DeepEqual(args, []interface{}{1, "a"}) {
		t.Errorf("args = %v", args)
	}
}

func TestClickHouseUpdateIsMutation(t *testing.T) {
	q, args, err := gqbd.BuildUpdate(gqbd.ClickHouse, "events").
		Set(map[string]interface{}{"name": "x", "v": 2}).
		Where("id = ?", 1).
		Setting("mutations_sync", 2).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	if want := "ALTER TABLE `events` UPDATE `name` = ?, `v` = ? WHERE id = ? SETTINGS mutations_sync = 2"; q != want {
		t.Errorf("got  %s\nwant %s", q, want)
	}
	if !reflect.DeepEqual(args, []interface{}{"x", 2, 1}) {
		t.Errorf("args = %v", args)
	}
}

func TestClickHouseDelete(t *testing.T) {
	q, args, err := gqbd.BuildDelete(gqbd.ClickHouse, "events").
		Where("status = ?", "gone").
		Setting("lightweight_deletes_sync", 2).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	if want := "DELETE FROM `events` WHERE status = ? SETTINGS lightweight_deletes_sync = 2"; q != want {
		t.Errorf("got  %s\nwant %s", q, want)
	}
	if !reflect.DeepEqual(args, []interface{}{"gone"}) {
		t.Errorf("args = %v", args)
	}
}

func TestClickHouseRequiresWhere(t *testing.T) {
	for _, qb := range []*gqbd.QueryBuilder{
		gqbd.BuildUpdate(gqbd.ClickHouse, "events").Set(map[string]interface{}{"a": 1}),
		gqbd.BuildDelete(gqbd.ClickHouse, "events"),
	} {
		if _, _, err := qb.Build(); err == nil || !strings.Contains(err.Error(), "WHERE") {
			t.Errorf("expected missing-WHERE error, got %v", err)
		}
	}
}

func TestClickHouseIdentifierBackslashEscape(t *testing.T) {
	got, err := gqbd.EscapeIdentifier(gqbd.ClickHouse, "x\\`; DROP TABLE events; --")
	if err != nil {
		t.Fatal(err)
	}
	if want := "`x\\\\``; DROP TABLE events; --`"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestClickHouseSettingLiterals(t *testing.T) {
	q, _, err := gqbd.BuildSelect(gqbd.ClickHouse, "t").
		Setting("a", false).Setting("b", int64(-3)).Setting("c", uint8(4)).
		Setting("d", 0.5).Setting("e", float32(1.25)).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	if want := "SELECT * FROM `t` SETTINGS a = 0, b = -3, c = 4, d = 0.5, e = 1.25"; q != want {
		t.Errorf("got %s, want %s", q, want)
	}
}

func TestClickHouseOnlyMethodErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		qb   *gqbd.QueryBuilder
		want string
	}{
		{"final on postgres", gqbd.BuildSelect(gqbd.PostgreSQL, "t").Final(), "only supported for ClickHouse"},
		{"prewhere on mysql", gqbd.BuildSelect(gqbd.Mysql, "t").Prewhere("a = ?", 1), "only supported for ClickHouse"},
		{"setting on sqlite", gqbd.BuildSelect(gqbd.SQLite, "t").Setting("a", 1), "only supported for ClickHouse"},
		{"final on insert", gqbd.BuildInsert(gqbd.ClickHouse, "t").Final(), "cannot be used with INSERT"},
		{"prewhere on delete", gqbd.BuildDelete(gqbd.ClickHouse, "t").Prewhere("a = 1"), "cannot be used with DELETE"},
		{"bad setting name", gqbd.BuildSelect(gqbd.ClickHouse, "t").Setting("a = 1, b", 1), "invalid setting name"},
		{"bad setting value", gqbd.BuildSelect(gqbd.ClickHouse, "t").Setting("a", []int{1}), "unsupported value type"},
		{"error kept", gqbd.BuildSelect(gqbd.ClickHouse, "").Final().Setting("a", 1), "empty identifier"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := tc.qb.Build()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestClickHouseConnectionString(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  gqbd.DBConfig
		want string
	}{
		{"default", gqbd.DBConfig{}, "clickhouse://localhost/"},
		{"full", gqbd.DBConfig{Host: "ch", Port: 9000, User: "default", Password: "p@ss", DBName: "analytics"}, "clickhouse://default:p%40ss@ch:9000/analytics"},
		{"tls", gqbd.DBConfig{Host: "ch", Port: 9440, SSLMode: "require"}, "clickhouse://ch:9440/?secure=true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := gqbd.BuildConnectionString(gqbd.ClickHouse, tc.cfg); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
