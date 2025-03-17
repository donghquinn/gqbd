package gqbd_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/donghquinn/gqbd"
)

/*
BuildSelect

@ Return: Final SELECT query string, arguments slice, and error if any
*/
func TestBuildSelectPostgreSQL(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "table_name", "col1", "col2").
		Where("col1 = ?", 100).
		OrderBy("col1", "ASC", nil).
		Limit(10).
		Offset(5)

	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedQuery := "SELECT \"col1\", \"col2\" FROM \"table_name\" WHERE col1 = $1 ORDER BY \"col1\" ASC LIMIT $2 OFFSET $3"
	normalizedQuery := strings.Join(strings.Fields(query), " ")
	normalizedExpected := strings.Join(strings.Fields(expectedQuery), " ")
	if normalizedQuery != normalizedExpected {
		t.Errorf("expected query:\n%s\ngot:\n%s", normalizedExpected, normalizedQuery)
	}
	expectedArgs := []interface{}{100, 10, 5}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, args)
	}
}

/*
BuildInsert with Returning

@ Return: INSERT query string, arguments slice, and error if any
*/
func TestBuildInsertPostgreSQL(t *testing.T) {
	data := map[string]interface{}{
		"col1": 200,
		"col2": "test",
	}
	qb := gqbd.BuildInsert(gqbd.PostgreSQL, "table_name").
		Values(data).
		Returning("col1")
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedQuery := "INSERT INTO \"table_name\" (\"col1\", \"col2\") VALUES ($1, $2) RETURNING col1"
	normalizedQuery := strings.Join(strings.Fields(query), " ")
	normalizedExpected := strings.Join(strings.Fields(expectedQuery), " ")
	if normalizedQuery != normalizedExpected {
		t.Errorf("expected query:\n%s\ngot:\n%s", normalizedExpected, normalizedQuery)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
}

/*
BuildUpdate

@ Return: UPDATE query string, arguments slice, and error if any
*/
func TestBuildUpdatePostgreSQL(t *testing.T) {
	data := map[string]interface{}{
		"col1": 300,
		"col2": "update",
	}
	qb := gqbd.BuildUpdate(gqbd.PostgreSQL, "table_name").
		Set(data).
		Where("col1 = ?", 100)
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedPrefix := "UPDATE \"table_name\" SET "
	if !strings.HasPrefix(query, expectedPrefix) {
		t.Errorf("expected query to start with %s, got %s", expectedPrefix, query)
	}
	if !strings.Contains(query, "WHERE col1 = $") {
		t.Errorf("expected query to contain WHERE clause, got %s", query)
	}
	expectedArgs := []interface{}{300, "update", 100}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, args)
	}
}

/*
BuildDelete

@ Return: DELETE query string, arguments slice, and error if any
*/
func TestBuildDeletePostgreSQL(t *testing.T) {
	qb := gqbd.BuildDelete(gqbd.PostgreSQL, "table_name").
		Where("col1 = ?", 100)
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedQuery := "DELETE FROM \"table_name\" WHERE col1 = $1"
	normalizedQuery := strings.Join(strings.Fields(query), " ")
	normalizedExpected := strings.Join(strings.Fields(expectedQuery), " ")
	if normalizedQuery != normalizedExpected {
		t.Errorf("expected query:\n%s\ngot:\n%s", normalizedExpected, normalizedQuery)
	}
	expectedArgs := []interface{}{100}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, args)
	}
}
