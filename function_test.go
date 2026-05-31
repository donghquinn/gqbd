package gqbd_test

import (
	"strings"
	"testing"

	"github.com/donghquinn/gqbd"
)

// TestBuildSelectWithCountFunction tests using COUNT() as a column
func TestBuildSelectWithCountFunction(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "toon_table t", "COUNT(t.toon_seq)")

	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Query String: %s", query)

	expectedQuery := `SELECT COUNT(t.toon_seq) FROM "toon_table" t`
	normalizedQuery := strings.Join(strings.Fields(query), " ")
	normalizedExpected := strings.Join(strings.Fields(expectedQuery), " ")

	if normalizedQuery != normalizedExpected {
		t.Errorf("expected query:\n%s\ngot:\n%s", normalizedExpected, normalizedQuery)
	}

	if len(args) != 0 {
		t.Errorf("expected 0 args, got %d", len(args))
	}
}

// TestBuildSelectWithCountAndJoin tests COUNT() with JOIN and WHERE conditions
func TestBuildSelectWithCountAndJoin(t *testing.T) {
	userName := "test"
	title := "sample"

	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "toon_table t", "COUNT(t.toon_seq)")

	if userName != "" {
		qb = qb.LeftJoin("user_table u", "u.user_id = t.user_id").
			Where("u.user_name LIKE ?", "%"+userName+"%")
	}

	if title != "" {
		qb = qb.Where("t.toon_title LIKE ?", "%"+title+"%")
	}

	query, args, err := qb.Where("t.toon_status = ?", "1").Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Query String: %s", query)
	t.Logf("Args: %v", args)

	// Verify the query contains all expected parts
	if !strings.Contains(query, "COUNT(t.toon_seq)") {
		t.Errorf("expected query to contain COUNT(t.toon_seq), got %s", query)
	}
	if !strings.Contains(query, "LEFT JOIN") {
		t.Errorf("expected query to contain LEFT JOIN, got %s", query)
	}
	if !strings.Contains(query, "WHERE") {
		t.Errorf("expected query to contain WHERE, got %s", query)
	}

	// Verify args
	if len(args) != 3 {
		t.Errorf("expected 3 args, got %d", len(args))
	}
}

// TestBuildSelectWithMultipleFunctions tests multiple aggregate functions
func TestBuildSelectWithMultipleFunctions(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "orders o", "o.customer_id", "COUNT(o.id)", "SUM(o.total)", "AVG(o.amount)")

	query, args, err := qb.
		Where("o.status = ?", "completed").
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Query String: %s", query)

	// Verify the query contains all functions without escaping
	if !strings.Contains(query, "COUNT(o.id)") {
		t.Errorf("expected query to contain COUNT(o.id), got %s", query)
	}
	if !strings.Contains(query, "SUM(o.total)") {
		t.Errorf("expected query to contain SUM(o.total), got %s", query)
	}
	if !strings.Contains(query, "AVG(o.amount)") {
		t.Errorf("expected query to contain AVG(o.amount), got %s", query)
	}
	// But regular columns should still be escaped (alias is not escaped, only column name)
	if !strings.Contains(query, `o."customer_id"`) {
		t.Errorf("expected query to contain escaped customer_id, got %s", query)
	}

	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(args))
	}
}

// TestBuildSelectWithCountStar tests COUNT(*) function
func TestBuildSelectWithCountStar(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "users", "COUNT(*)")

	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Query String: %s", query)

	if !strings.Contains(query, "COUNT(*)") {
		t.Errorf("expected query to contain COUNT(*), got %s", query)
	}

	if len(args) != 0 {
		t.Errorf("expected 0 args, got %d", len(args))
	}
}

// TestBuildSelectWithCountFunctionMySQL tests COUNT() for MySQL/MariaDB
func TestBuildSelectWithCountFunctionMySQL(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.MariaDB, "orders o", "COUNT(o.id)", "SUM(o.total)").
		Where("o.status = ?", "completed")

	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Query String: %s", query)

	// Verify functions are not escaped
	if !strings.Contains(query, "COUNT(o.id)") {
		t.Errorf("expected query to contain COUNT(o.id), got %s", query)
	}
	if !strings.Contains(query, "SUM(o.total)") {
		t.Errorf("expected query to contain SUM(o.total), got %s", query)
	}

	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(args))
	}
}

// TestBuildSelectWithCountFunctionSQLite tests COUNT() for SQLite
func TestBuildSelectWithCountFunctionSQLite(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.SQLite, "products p", "COUNT(p.id)").
		Where("p.active = ?", true)

	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Query String: %s", query)

	if !strings.Contains(query, "COUNT(p.id)") {
		t.Errorf("expected query to contain COUNT(p.id), got %s", query)
	}

	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(args))
	}
}

// TestUserExactCase tests the exact user scenario with dynamic conditions
func TestUserExactCase(t *testing.T) {
	// Test case 1: with userName and title
	t.Run("WithUserNameAndTitle", func(t *testing.T) {
		userName := "test_user"
		title := "test_title"

		qb := gqbd.BuildSelect(gqbd.PostgreSQL, "toon_table t", "COUNT(t.toon_seq)")

		if userName != "" {
			qb = qb.LeftJoin("user_table u", "u.user_id = t.user_id").
				Where("u.user_name LIKE ?", "%"+userName+"%")
		}

		if title != "" {
			qb = qb.Where("t.toon_title LIKE ?", "%"+title+"%")
		}

		query, args, err := qb.Where("t.toon_status = ?", "1").Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.Logf("Query: %s", query)
		t.Logf("Args: %v", args)

		// Verify the query is valid
		if !strings.Contains(query, "COUNT(t.toon_seq)") {
			t.Errorf("expected query to contain COUNT(t.toon_seq), got %s", query)
		}
		if !strings.Contains(query, "FROM") {
			t.Errorf("expected query to contain FROM, got %s", query)
		}
		if strings.Contains(query, `"COUNT(t.toon_seq)"`) {
			t.Errorf("COUNT should not be escaped, got %s", query)
		}

		// Verify args
		if len(args) != 3 {
			t.Errorf("expected 3 args, got %d: %v", len(args), args)
		}
	})

	// Test case 2: without userName
	t.Run("WithoutUserName", func(t *testing.T) {
		userName := ""
		title := "test_title"

		qb := gqbd.BuildSelect(gqbd.PostgreSQL, "toon_table t", "COUNT(t.toon_seq)")

		if userName != "" {
			qb = qb.LeftJoin("user_table u", "u.user_id = t.user_id").
				Where("u.user_name LIKE ?", "%"+userName+"%")
		}

		if title != "" {
			qb = qb.Where("t.toon_title LIKE ?", "%"+title+"%")
		}

		query, args, err := qb.Where("t.toon_status = ?", "1").Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.Logf("Query: %s", query)
		t.Logf("Args: %v", args)

		// Should not have JOIN
		if strings.Contains(query, "JOIN") {
			t.Errorf("expected no JOIN when userName is empty, got %s", query)
		}

		// Verify args (only 2 now: title and status)
		if len(args) != 2 {
			t.Errorf("expected 2 args, got %d: %v", len(args), args)
		}
	})
}
