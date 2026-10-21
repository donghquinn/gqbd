package gqbd_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/donghquinn/gqbd"
)

// ---- Aggregate ----

func TestAggregatePostgreSQL(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders", "customer_id").
		Aggregate("COUNT", "id")
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, `COUNT("id")`) {
		t.Errorf("expected COUNT(\"id\"), got %s", query)
	}
}

func TestAggregateMariaDB(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.MariaDB, "orders", "customer_id").
		Aggregate("SUM", "amount")
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "SUM(`amount`)") {
		t.Errorf("expected SUM(`amount`), got %s", query)
	}
}

func TestAggregateSQLite(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.SQLite, "orders", "customer_id").
		Aggregate("AVG", "total")
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, `AVG("total")`) {
		t.Errorf("expected AVG(\"total\"), got %s", query)
	}
}

func TestAggregateErrorPropagation(t *testing.T) {
	qb := gqbd.NewQueryBuilder(gqbd.PostgreSQL, "") // empty table sets error
	qb = qb.Aggregate("COUNT", "id")
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error to propagate through Aggregate(), got nil")
	}
}

// ---- EscapeIdentifier edge cases ----

func TestEscapeIdentifierEmpty(t *testing.T) {
	_, err := gqbd.EscapeIdentifier(gqbd.PostgreSQL, "")
	if err == nil {
		t.Error("expected error for empty identifier, got nil")
	}
}

func TestEscapeIdentifierTableASAliasPostgreSQL(t *testing.T) {
	result, err := gqbd.EscapeIdentifier(gqbd.PostgreSQL, "users AS u")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `"users" AS u` {
		t.Errorf(`expected "users" AS u, got %s`, result)
	}
}

func TestEscapeIdentifierTableASAliasMariaDB(t *testing.T) {
	result, err := gqbd.EscapeIdentifier(gqbd.MariaDB, "orders AS o")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "`orders` AS o" {
		t.Errorf("expected `orders` AS o, got %s", result)
	}
}

func TestEscapeIdentifierUnknownDBType(t *testing.T) {
	result, err := gqbd.EscapeIdentifier(gqbd.DBType("unknown"), "mycolumn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "mycolumn" {
		t.Errorf("expected unescaped identifier for unknown DB type, got %s", result)
	}
}

// ---- ValidateDirection ----

func TestValidateDirectionInvalid(t *testing.T) {
	if gqbd.ValidateDirection("INVALID") != "DESC" {
		t.Error("expected DESC fallback for invalid direction")
	}
}

func TestValidateDirectionLowercase(t *testing.T) {
	if gqbd.ValidateDirection("asc") != "ASC" {
		t.Error("expected ASC for lowercase asc")
	}
	if gqbd.ValidateDirection("desc") != "DESC" {
		t.Error("expected DESC for lowercase desc")
	}
}

// ---- Build() with no op ----

func TestBuildWithNoOperation(t *testing.T) {
	// NewQueryBuilder does not set op, so Build() hits the default case
	qb := gqbd.NewQueryBuilder(gqbd.PostgreSQL, "users")
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty op, got nil")
	}
}

// ---- Unknown DBType falls through to MySQL in build* ----

func TestBuildSelectUnknownDBType(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.DBType("custom"), "users", "id")
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// default falls back to MySQL (backtick quoting)
	if !strings.Contains(query, "SELECT") {
		t.Errorf("expected SELECT query, got %s", query)
	}
}

func TestBuildInsertUnknownDBType(t *testing.T) {
	qb := gqbd.BuildInsert(gqbd.DBType("custom"), "users").
		Values(map[string]interface{}{"name": "alice"})
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "INSERT INTO") {
		t.Errorf("expected INSERT INTO, got %s", query)
	}
}

func TestBuildUpdateUnknownDBType(t *testing.T) {
	qb := gqbd.BuildUpdate(gqbd.DBType("custom"), "users").
		Set(map[string]interface{}{"name": "bob"}).
		Where("id = ?", 1)
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "UPDATE") {
		t.Errorf("expected UPDATE, got %s", query)
	}
}

func TestBuildConnectionStringUnknownDBType(t *testing.T) {
	config := gqbd.DBConfig{Host: "localhost", Port: 3306, User: "root", DBName: "db"}
	dsn := gqbd.BuildConnectionString(gqbd.DBType("custom"), config)
	// default falls back to MySQL format
	if !strings.Contains(dsn, "@tcp(") {
		t.Errorf("expected MySQL-format DSN for unknown type, got %s", dsn)
	}
}

// ---- BuildConnectionString MySQL type ----

func TestBuildConnectionStringMysqlType(t *testing.T) {
	config := gqbd.DBConfig{
		Host: "localhost", Port: 3306, User: "root",
		Password: "pass", DBName: "mydb", Charset: "utf8",
	}
	dsn := gqbd.BuildConnectionString(gqbd.Mysql, config)
	if !strings.HasPrefix(dsn, "root:pass@tcp(localhost:3306)/mydb") {
		t.Errorf("unexpected MySQL DSN: %s", dsn)
	}
}

// ---- SQLite INSERT with RETURNING ----

func TestBuildInsertSQLiteWithReturning(t *testing.T) {
	qb := gqbd.BuildInsert(gqbd.SQLite, "users").
		Values(map[string]interface{}{"name": "alice"}).
		Returning("id")
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "RETURNING id") {
		t.Errorf("expected RETURNING id, got %s", query)
	}
}

// ---- INSERT/UPDATE without data for MariaDB and SQLite ----

func TestBuildInsertMariaDBWithoutDataReturnsError(t *testing.T) {
	qb := gqbd.BuildInsert(gqbd.MariaDB, "users")
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for MariaDB INSERT without data, got nil")
	}
}

func TestBuildInsertSQLiteWithoutDataReturnsError(t *testing.T) {
	qb := gqbd.BuildInsert(gqbd.SQLite, "users")
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for SQLite INSERT without data, got nil")
	}
}

func TestBuildUpdateMariaDBWithoutDataReturnsError(t *testing.T) {
	qb := gqbd.BuildUpdate(gqbd.MariaDB, "users").Where("id = ?", 1)
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for MariaDB UPDATE without data, got nil")
	}
}

func TestBuildUpdateSQLiteWithoutDataReturnsError(t *testing.T) {
	qb := gqbd.BuildUpdate(gqbd.SQLite, "users").Where("id = ?", 1)
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for SQLite UPDATE without data, got nil")
	}
}

// ---- prepareInsertParts: EscapeIdentifier error on key ----

func TestBuildInsertWithEmptyKeyReturnsError(t *testing.T) {
	qb := gqbd.BuildInsert(gqbd.PostgreSQL, "users").
		Values(map[string]interface{}{"": "value"})
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty column key in Values(), got nil")
	}
}

// ---- NewQueryBuilder with invalid table/column ----

func TestNewQueryBuilderWithEmptyTable(t *testing.T) {
	qb := gqbd.NewQueryBuilder(gqbd.PostgreSQL, "")
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty table name, got nil")
	}
}

func TestNewQueryBuilderWithEmptyColumn(t *testing.T) {
	qb := gqbd.NewQueryBuilder(gqbd.PostgreSQL, "users", "")
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty column name, got nil")
	}
}

// ---- Error propagation through chained methods ----

func TestErrorPropagationChain(t *testing.T) {
	// Build a builder that immediately has an error (empty table)
	qb := gqbd.NewQueryBuilder(gqbd.PostgreSQL, "")

	// Every subsequent method should skip and preserve the error
	qb = qb.
		Distinct().
		Aggregate("COUNT", "id").
		LeftJoin("other", "other.id = t.id").
		InnerJoin("other2", "other2.id = t.id").
		RightJoin("other3", "other3.id = t.id").
		Where("id = ?", 1).
		WhereIn("id", []interface{}{1, 2}).
		WhereBetween("age", 18, 65).
		GroupBy("col1").
		Having("COUNT(id) > ?", 1).
		OrderBy("id", "ASC", nil).
		Limit(10).
		Offset(5)

	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error to propagate through all chained methods, got nil")
	}
}

// ---- WhereIn and WhereBetween for SQLite ----

func TestWhereInSQLite(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.SQLite, "users", "id").
		WhereIn("id", []interface{}{1, 2, 3})
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, `"id" IN (?, ?, ?)`) {
		t.Errorf("expected IN clause with ?, got %s", query)
	}
	if !reflect.DeepEqual(args, []interface{}{1, 2, 3}) {
		t.Errorf("expected args [1,2,3], got %v", args)
	}
}

func TestWhereBetweenSQLite(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.SQLite, "orders", "id").
		WhereBetween("amount", 100, 500)
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, `"amount" BETWEEN ? AND ?`) {
		t.Errorf("expected BETWEEN clause, got %s", query)
	}
	if !reflect.DeepEqual(args, []interface{}{100, 500}) {
		t.Errorf("expected args [100,500], got %v", args)
	}
}

// ---- SQLite SELECT features ----

func TestSQLiteSelectDistinct(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.SQLite, "users", "name").Distinct()
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(query, "SELECT DISTINCT") {
		t.Errorf("expected SELECT DISTINCT, got %s", query)
	}
}

func TestSQLiteSelectWithJoins(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.SQLite, "orders o", "o.id").
		LeftJoin("users u", "u.id = o.user_id").
		InnerJoin("products p", "p.id = o.product_id").
		Where("o.status = ?", "active")
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "LEFT JOIN") {
		t.Errorf("expected LEFT JOIN, got %s", query)
	}
	if !strings.Contains(query, "INNER JOIN") {
		t.Errorf("expected INNER JOIN, got %s", query)
	}
	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(args))
	}
}

func TestSQLiteSelectGroupByHaving(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.SQLite, "orders", "customer_id", "COUNT(id)").
		GroupBy("customer_id").
		Having("COUNT(id) > ?", 3)
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "GROUP BY") {
		t.Errorf("expected GROUP BY, got %s", query)
	}
	if !strings.Contains(query, "HAVING COUNT(id) > ?") {
		t.Errorf("expected HAVING clause, got %s", query)
	}
	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(args))
	}
}

// ---- MariaDB SELECT with additional join types ----

func TestMariaDBSelectWithAllJoinTypes(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.MariaDB, "orders o", "o.id").
		InnerJoin("users u", "u.id = o.user_id").
		RightJoin("products p", "p.id = o.product_id")
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "INNER JOIN") {
		t.Errorf("expected INNER JOIN, got %s", query)
	}
	if !strings.Contains(query, "RIGHT JOIN") {
		t.Errorf("expected RIGHT JOIN, got %s", query)
	}
}

// ---- DELETE without WHERE ----

func TestDeleteWithoutWhere(t *testing.T) {
	for _, tc := range []struct {
		dbType gqbd.DBType
		name   string
	}{
		{gqbd.PostgreSQL, "PostgreSQL"},
		{gqbd.MariaDB, "MariaDB"},
		{gqbd.SQLite, "SQLite"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			qb := gqbd.BuildDelete(tc.dbType, "temp_logs")
			query, args, err := qb.Build()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(query, "DELETE FROM") {
				t.Errorf("expected DELETE FROM, got %s", query)
			}
			if len(args) != 0 {
				t.Errorf("expected no args, got %v", args)
			}
		})
	}
}

// ---- SELECT with wildcard (no columns) ----

func TestSelectNoColumnsWildcard(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users")
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "SELECT *") {
		t.Errorf("expected SELECT *, got %s", query)
	}
}

// ---- OrderBy with invalid direction ----

func TestOrderByInvalidDirection(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id").
		OrderBy("name", "RANDOM", nil)
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "DESC") {
		t.Errorf("expected DESC fallback for invalid direction, got %s", query)
	}
}

// ---- Multiple chained WHERE conditions ----

func TestMultipleWhereConditions(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id").
		Where("status = ?", "active").
		Where("age > ?", 18)
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "AND") {
		t.Errorf("expected AND between WHERE conditions, got %s", query)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}
}

// ---- Mysql type routes through same paths as MariaDB ----

func TestMysqlBuildInsert(t *testing.T) {
	qb := gqbd.BuildInsert(gqbd.Mysql, "users").
		Values(map[string]interface{}{"name": "alice"})
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "`users`") {
		t.Errorf("expected backtick-quoted table for Mysql type, got %s", query)
	}
}

func TestMysqlBuildUpdate(t *testing.T) {
	qb := gqbd.BuildUpdate(gqbd.Mysql, "users").
		Set(map[string]interface{}{"name": "bob"}).
		Where("id = ?", 1)
	query, _, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(query, "`users`") {
		t.Errorf("expected backtick-quoted table for Mysql type, got %s", query)
	}
}

// ---- EscapeIdentifier error paths in chained methods ----

func TestAggregateWithEmptyColumnReturnsError(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id").
		Aggregate("COUNT", "")
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty column in Aggregate(), got nil")
	}
}

func TestLeftJoinWithEmptyTableReturnsError(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id").
		LeftJoin("", "other.id = users.id")
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty joinTable in LeftJoin(), got nil")
	}
}

func TestInnerJoinWithEmptyTableReturnsError(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id").
		InnerJoin("", "other.id = users.id")
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty joinTable in InnerJoin(), got nil")
	}
}

func TestRightJoinWithEmptyTableReturnsError(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id").
		RightJoin("", "other.id = users.id")
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty joinTable in RightJoin(), got nil")
	}
}

func TestWhereInWithEmptyColumnReturnsError(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id").
		WhereIn("", []interface{}{1, 2, 3})
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty column in WhereIn(), got nil")
	}
}

func TestWhereBetweenWithEmptyColumnReturnsError(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders", "id").
		WhereBetween("", 10, 100)
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty column in WhereBetween(), got nil")
	}
}

func TestGroupByWithEmptyColumnReturnsError(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders", "customer_id").
		GroupBy("")
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty column in GroupBy(), got nil")
	}
}

func TestOrderByWithEmptyColumnReturnsError(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "id").
		OrderBy("", "ASC", nil)
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty column in OrderBy(), got nil")
	}
}

// ---- UPDATE with empty key in data map ----

func TestBuildUpdateWithEmptyKeyPostgreSQL(t *testing.T) {
	qb := gqbd.BuildUpdate(gqbd.PostgreSQL, "users").
		Set(map[string]interface{}{"": "value"}).
		Where("id = ?", 1)
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty key in PostgreSQL UPDATE data, got nil")
	}
}

func TestBuildUpdateWithEmptyKeyMariaDB(t *testing.T) {
	qb := gqbd.BuildUpdate(gqbd.MariaDB, "users").
		Set(map[string]interface{}{"": "value"}).
		Where("id = ?", 1)
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty key in MariaDB UPDATE data, got nil")
	}
}

func TestBuildUpdateWithEmptyKeySQLite(t *testing.T) {
	qb := gqbd.BuildUpdate(gqbd.SQLite, "users").
		Set(map[string]interface{}{"": "value"}).
		Where("id = ?", 1)
	_, _, err := qb.Build()
	if err == nil {
		t.Error("expected error for empty key in SQLite UPDATE data, got nil")
	}
}
