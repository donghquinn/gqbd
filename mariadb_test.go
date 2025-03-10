package gqbd_test

import (
	"reflect"
	"testing"

	gqbd "github.com/donghquinn/go-query-builder"
)

func TestMariadbSelect(t *testing.T) {
	resultQueryString := `SELECT "new_id", "new_name" FROM "new_table"`

	qb, qbErr := gqbd.NewQueryBuilder("mariadb", "new_table", "new_id", "new_name")

	queryString, _ := qb.Build()

	if queryString != resultQueryString {
		t.Fatalf("[MARIADB_SELECT_TEST] Not Match: %v", queryString)
	}
}

func TestMariadbSelectWhere(t *testing.T) {
	resultQueryString := `SELECT "new_id", "new_name" FROM "new_table" WHERE new_id = ?`

	resultArgs := []interface{}{"abc123"}

	qb := gqbd.NewQueryBuilder("mariadb", "new_table", "new_id", "new_name").
		Where("new_id = ?", "abc123")

	queryString, args := qb.Build()

	if queryString != resultQueryString {
		t.Fatalf("[MARIADB_SELECT_TEST] Not Match: %v", queryString)
	}
	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[MARIADB_SELECT_TEST] Args Not Match: %v", args)
	}
}

func TestMariadbSelectWhereWithOrderBy(t *testing.T) {
	resultQueryString := `SELECT "new_seq", "new_id", "new_name" FROM "new_table" WHERE new_id = ? ORDER BY "new_seq" DESC`

	resultArgs := []interface{}{"abc123"}

	qb, qbErr := gqbd.NewQueryBuilder("mariadb", "new_table", "new_seq", "new_id", "new_name").
		Where("new_id = ?", "abc123").
		OrderBy("new_seq", "DESC", nil)

	queryString, args := qb.Build()

	if queryString != resultQueryString {
		t.Fatalf("[MARIADB_SELECT_TEST] Not Match: %v", queryString)
	}
	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[MARIADB_SELECT_TEST] Args Not Match: %v", args)
	}
}

func TestMariadbSelectPagination(t *testing.T) {
	resultQueryString := `
		SELECT "new_seq", "new_id", "new_name" FROM "new_table" WHERE new_id = ? AND new_name = ? ORDER BY "new_seq" DESC LIMIT ? OFFSET ?
	`

	resultArgs := []interface{}{"abc123", "testName", 10, 3}

	qb := gqbd.NewQueryBuilder("mariadb", "new_table", "new_seq", "new_id", "new_name").
		Where("new_id = ?", "abc123").
		Where("new_name = ?", "testName").
		OrderBy("new_seq", "DESC", nil).
		Offset(3).
		Limit(10)

	queryString, args := qb.Build()

	if queryString != resultQueryString {
		t.Fatalf("[MARIADB_SELECT_TEST] Not Match: %v", queryString)
	}

	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[MARIADB_SELECT_TEST] Args Not Match: %v", args)
	}
}
