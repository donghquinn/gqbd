package gqbd_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/donghquinn/gqbd"
)

func TestWhereInPostgreSQL(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id", "name").
		WhereIn("id", []interface{}{1, 2, 3})
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, `"id" IN ($1, $2, $3)`) {
		t.Errorf("expected IN clause with $N placeholders, got %s", query)
	}
	if !reflect.DeepEqual(args, []interface{}{1, 2, 3}) {
		t.Errorf("expected args [1, 2, 3], got %v", args)
	}
}

func TestWhereInMariaDB(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.MariaDB, "users", "id", "name").
		WhereIn("id", []interface{}{1, 2, 3})
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "`id` IN (?, ?, ?)") {
		t.Errorf("expected IN clause with ? placeholders, got %s", query)
	}
	if !reflect.DeepEqual(args, []interface{}{1, 2, 3}) {
		t.Errorf("expected args [1, 2, 3], got %v", args)
	}
}

func TestWhereBetweenPostgreSQL(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders", "id").
		WhereBetween("amount", 100, 500)
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, `"amount" BETWEEN $1 AND $2`) {
		t.Errorf("expected BETWEEN clause, got %s", query)
	}
	if !reflect.DeepEqual(args, []interface{}{100, 500}) {
		t.Errorf("expected args [100, 500], got %v", args)
	}
}

func TestWhereBetweenMariaDB(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.MariaDB, "orders", "id").
		WhereBetween("amount", 100, 500)
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "`amount` BETWEEN ? AND ?") {
		t.Errorf("expected BETWEEN clause, got %s", query)
	}
	if !reflect.DeepEqual(args, []interface{}{100, 500}) {
		t.Errorf("expected args [100, 500], got %v", args)
	}
}

func TestGroupByAndHavingPostgreSQL(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders o", "o.customer_id", "COUNT(o.id)").
		GroupBy("o.customer_id").
		Having("COUNT(o.id) > ?", 5)
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "GROUP BY") {
		t.Errorf("expected GROUP BY clause, got %s", query)
	}
	if !strings.Contains(query, "HAVING COUNT(o.id) > $1") {
		t.Errorf("expected HAVING clause with $1, got %s", query)
	}
	if !reflect.DeepEqual(args, []interface{}{5}) {
		t.Errorf("expected args [5], got %v", args)
	}
}

func TestGroupByAndHavingMariaDB(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.MariaDB, "orders o", "o.customer_id", "COUNT(o.id)").
		GroupBy("o.customer_id").
		Having("COUNT(o.id) > ?", 5)
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "GROUP BY") {
		t.Errorf("expected GROUP BY clause, got %s", query)
	}
	if !strings.Contains(query, "HAVING COUNT(o.id) > ?") {
		t.Errorf("expected HAVING clause with ?, got %s", query)
	}
	if !reflect.DeepEqual(args, []interface{}{5}) {
		t.Errorf("expected args [5], got %v", args)
	}
}

func TestDistinctPostgreSQL(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "name").Distinct()
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(query, "SELECT DISTINCT") {
		t.Errorf("expected SELECT DISTINCT, got %s", query)
	}
}

func TestDistinctMariaDB(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.MariaDB, "users", "name").Distinct()
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(query, "SELECT DISTINCT") {
		t.Errorf("expected SELECT DISTINCT, got %s", query)
	}
}

func TestInnerJoinPostgreSQL(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders o", "o.id").
		InnerJoin("users u", "u.id = o.user_id")
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, `INNER JOIN "users" u ON u.id = o.user_id`) {
		t.Errorf("expected INNER JOIN clause, got %s", query)
	}
}

func TestRightJoinPostgreSQL(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders o", "o.id").
		RightJoin("users u", "u.id = o.user_id")
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, `RIGHT JOIN "users" u ON u.id = o.user_id`) {
		t.Errorf("expected RIGHT JOIN clause, got %s", query)
	}
}

func TestAddWhereIfNotEmpty(t *testing.T) {
	t.Run("NonEmptyString", func(t *testing.T) {
		qb := gqbd.BuildSelect(gqbd.MariaDB, "users", "id").
			AddWhereIfNotEmpty("name = ?", "alice")
		query, args, err := qb.Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(query, "WHERE name = ?") {
			t.Errorf("expected WHERE clause, got %s", query)
		}
		if !reflect.DeepEqual(args, []interface{}{"alice"}) {
			t.Errorf("expected args [alice], got %v", args)
		}
	})

	t.Run("EmptyString", func(t *testing.T) {
		qb := gqbd.BuildSelect(gqbd.MariaDB, "users", "id").
			AddWhereIfNotEmpty("name = ?", "")
		query, args, err := qb.Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(query, "WHERE") {
			t.Errorf("expected no WHERE clause for empty string, got %s", query)
		}
		if len(args) != 0 {
			t.Errorf("expected no args, got %v", args)
		}
	})

	t.Run("NilValue", func(t *testing.T) {
		qb := gqbd.BuildSelect(gqbd.MariaDB, "users", "id").
			AddWhereIfNotEmpty("name = ?", nil)
		query, _, err := qb.Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(query, "WHERE") {
			t.Errorf("expected no WHERE clause for nil, got %s", query)
		}
	})

	t.Run("NilPointerString", func(t *testing.T) {
		// nil *string passed as interface{} is not == nil; the *string case handles it
		var nilStr *string
		qb := gqbd.BuildSelect(gqbd.MariaDB, "users", "id").
			AddWhereIfNotEmpty("name = ?", nilStr)
		query, _, err := qb.Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(query, "WHERE") {
			t.Errorf("expected no WHERE clause for nil *string, got %s", query)
		}
	})

	t.Run("IntZeroIsNotFiltered", func(t *testing.T) {
		// Non-string zero values are never filtered — only string/nil are
		qb := gqbd.BuildSelect(gqbd.MariaDB, "users", "id").
			AddWhereIfNotEmpty("status = ?", 0)
		query, args, err := qb.Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(query, "WHERE") {
			t.Errorf("expected WHERE clause for int 0, got %s", query)
		}
		if !reflect.DeepEqual(args, []interface{}{0}) {
			t.Errorf("expected args [0], got %v", args)
		}
	})
}

func TestOrderByWithAllowedColumns(t *testing.T) {
	allowed := map[string]bool{"name": true, "email": true}

	t.Run("AllowedColumn", func(t *testing.T) {
		qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id").
			OrderBy("name", "ASC", allowed)
		query, _, err := qb.Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(query, `ORDER BY "name" ASC`) {
			t.Errorf("expected ORDER BY name ASC, got %s", query)
		}
	})

	t.Run("DisallowedColumnFallsBackToID", func(t *testing.T) {
		qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id").
			OrderBy("injected_column", "ASC", allowed)
		query, _, err := qb.Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(query, `ORDER BY "id" ASC`) {
			t.Errorf("expected fallback to id, got %s", query)
		}
	})
}

func TestBuildConnectionStringPostgreSQL(t *testing.T) {
	config := gqbd.DBConfig{
		Host: "localhost", Port: 5432, User: "postgres",
		Password: "secret", DBName: "mydb", SSLMode: "disable",
	}
	dsn := gqbd.BuildConnectionString(gqbd.PostgreSQL, config)
	expected := "host=localhost port=5432 user=postgres password=secret dbname=mydb sslmode=disable"
	if dsn != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, dsn)
	}
}

func TestBuildConnectionStringPostgreSQLDefaultSSL(t *testing.T) {
	config := gqbd.DBConfig{
		Host: "localhost", Port: 5432, User: "postgres", DBName: "mydb",
	}
	dsn := gqbd.BuildConnectionString(gqbd.PostgreSQL, config)
	if !strings.Contains(dsn, "sslmode=disable") {
		t.Errorf("expected sslmode=disable default, got %s", dsn)
	}
}

func TestBuildConnectionStringMySQL(t *testing.T) {
	config := gqbd.DBConfig{
		Host: "localhost", Port: 3306, User: "root",
		Password: "secret", DBName: "mydb", Charset: "utf8mb4",
	}
	dsn := gqbd.BuildConnectionString(gqbd.MariaDB, config)
	if !strings.HasPrefix(dsn, "root:secret@tcp(localhost:3306)/mydb") {
		t.Errorf("unexpected DSN format: %s", dsn)
	}
	if !strings.Contains(dsn, "charset=utf8mb4") {
		t.Errorf("expected charset in DSN, got %s", dsn)
	}
	if !strings.Contains(dsn, "parseTime=True") {
		t.Errorf("expected parseTime=True in DSN, got %s", dsn)
	}
}

func TestBuildConnectionStringSQLite(t *testing.T) {
	t.Run("WithFilePath", func(t *testing.T) {
		config := gqbd.DBConfig{FilePath: "/tmp/test.db"}
		dsn := gqbd.BuildConnectionString(gqbd.SQLite, config)
		if dsn != "/tmp/test.db" {
			t.Errorf("expected /tmp/test.db, got %s", dsn)
		}
	})

	t.Run("InMemoryFallback", func(t *testing.T) {
		config := gqbd.DBConfig{}
		dsn := gqbd.BuildConnectionString(gqbd.SQLite, config)
		if dsn != ":memory:" {
			t.Errorf("expected :memory:, got %s", dsn)
		}
	})
}

func TestMysqlDBTypeSameAsMariaDB(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.Mysql, "users", "id", "name").
		Where("id = ?", 1)
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "`id`") {
		t.Errorf("expected backtick-quoted identifier for Mysql type, got %s", query)
	}
	if !reflect.DeepEqual(args, []interface{}{1}) {
		t.Errorf("expected args [1], got %v", args)
	}
}

// TestOffsetZeroIsSkipped documents that Offset(0) is a no-op.
func TestOffsetZeroIsSkipped(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id").
		Limit(10).
		Offset(0)
	_, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 1 {
		t.Errorf("expected 1 arg (limit only), got %d: %v", len(args), args)
	}
}

func TestValuesOnSelectReturnsError(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id").
		Values(map[string]interface{}{"col": 1})
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error when calling Values() on SELECT, got nil")
	}
}

func TestSetOnSelectReturnsError(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id").
		Set(map[string]interface{}{"col": 1})
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error when calling Set() on SELECT, got nil")
	}
}

func TestReturningOnUpdateReturnsError(t *testing.T) {
	qb := gqbd.BuildUpdate(gqbd.PostgreSQL, "users").Returning("id")
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error when calling Returning() on UPDATE, got nil")
	}
}

func TestBuildInsertWithoutValuesReturnsError(t *testing.T) {
	qb := gqbd.BuildInsert(gqbd.PostgreSQL, "users")
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error when building INSERT without Values(), got nil")
	}
}

func TestBuildUpdateWithoutSetReturnsError(t *testing.T) {
	qb := gqbd.BuildUpdate(gqbd.PostgreSQL, "users").Where("id = ?", 1)
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error when building UPDATE without Set(), got nil")
	}
}
