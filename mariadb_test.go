package gqbd_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/donghquinn/gqbd"
)

func TestBuildSelectMariaDB(t *testing.T) {
	qb := gqbd.BuildSelect(gqbd.MariaDB, "table_name", "col1", "col2").
		Where("col1 = ?", 100).
		OrderBy("col1", "ASC", nil).
		Limit(10).
		Offset(5)

	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Query String :%s", query)

	expectedQuery := "SELECT `col1`, `col2` FROM `table_name` WHERE col1 = ? ORDER BY `col1` ASC LIMIT ? OFFSET ?"
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

func TestBuildInsertMariaDB(t *testing.T) {
	data := map[string]interface{}{
		"col1": 200,
		"col2": "test",
	}
	qb := gqbd.BuildInsert(gqbd.MariaDB, "table_name").
		Values(data)
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("Query String :%s", query)

	expectedQuery := "INSERT INTO `table_name` (`col1`, `col2`) VALUES (?, ?)"
	normalizedQuery := strings.Join(strings.Fields(query), " ")
	normalizedExpected := strings.Join(strings.Fields(expectedQuery), " ")
	if normalizedQuery != normalizedExpected {
		t.Errorf("expected query:\n%s\ngot:\n%s", normalizedExpected, normalizedQuery)
	}
	expectedArgs := []interface{}{200, "test"}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, args)
	}
}

func TestBuildUpdateMariaDB(t *testing.T) {
	data := map[string]interface{}{
		"col1": 300,
		"col2": "update",
	}
	qb := gqbd.BuildUpdate(gqbd.MariaDB, "table_name").
		Set(data).
		Where("col1 = ?", 100)
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("Query String :%s", query)

	expectedPrefix := "UPDATE `table_name` SET "
	if !strings.HasPrefix(query, expectedPrefix) {
		t.Errorf("expected query to start with %s, got %s", expectedPrefix, query)
	}
	if !strings.Contains(query, "WHERE col1 = ?") {
		t.Errorf("expected query to contain WHERE clause, got %s", query)
	}
	expectedArgs := []interface{}{300, "update", 100}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, args)
	}
}

func TestBuildDeleteMariaDB(t *testing.T) {
	qb := gqbd.BuildDelete(gqbd.MariaDB, "table_name").
		Where("col1 = ?", 100)
	query, args, err := qb.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedQuery := "DELETE FROM `table_name` WHERE col1 = ?"
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
