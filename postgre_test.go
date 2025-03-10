package gqbd_test

import (
	"reflect"
	"testing"

	gqbd "github.com/donghquinn/go-query-builder"
)

func TestPostgresSelect(t *testing.T) {
	resultQueryString := `SELECT "new_id", "new_name" FROM "new_table"`

	qb := gqbd.NewQueryBuilder("postgres", "new_table", "new_id", "new_name")

	queryString, _, buildErr := qb.Build()

	if buildErr != nil {
		t.Fatalf("[POSTGRE_SELECT_TEST] Make Query String Error: %v", buildErr)
	}

	if queryString != resultQueryString {
		t.Fatalf("[POSTGRE_SELECT_TEST] Not Match: %v", queryString)
	}
}

func TestPostgresSelectWhere(t *testing.T) {
	resultQueryString := `SELECT "new_id", "new_name" FROM "new_table" WHERE new_id = $1`

	resultArgs := []interface{}{"abc123"}

	qb := gqbd.NewQueryBuilder("postgres", "new_table", "new_id", "new_name").
		Where("new_id = ?", "abc123")

	queryString, args, buildErr := qb.Build()

	if buildErr != nil {
		t.Fatalf("[POSTGRE_SELECT_TEST] Make Query String Error: %v", buildErr)
	}

	if queryString != resultQueryString {
		t.Fatalf("[POSTGRE_SELECT_TEST] Not Match: %v", queryString)
	}
	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[POSTGRE_SELECT_TEST] Args Not Match: %v", args)
	}
}

func TestPostgresSelectWhereWithOrderBy(t *testing.T) {
	resultQueryString := `SELECT "new_seq", "new_id", "new_name" FROM "new_table" WHERE new_id = $1 ORDER BY "new_seq" DESC`

	resultArgs := []interface{}{"abc123"}

	qb := gqbd.NewQueryBuilder("postgres", "new_table", "new_seq", "new_id", "new_name").
		Where("new_id = ?", "abc123").
		OrderBy("new_seq", "DESC", nil)

	queryString, args, buildErr := qb.Build()

	if buildErr != nil {
		t.Fatalf("[POSTGRE_SELECT_TEST] Make Query String Error: %v", buildErr)
	}

	if queryString != resultQueryString {
		t.Fatalf("[POSTGRE_SELECT_TEST] Not Match: %v", queryString)
	}
	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[POSTGRE_SELECT_TEST] Args Not Match: %v", args)
	}
}

func TestPostgresSelectPagination(t *testing.T) {
	resultQueryString := `SELECT "new_seq", "new_id", "new_name" FROM "new_table" WHERE new_id = $1 AND new_name = $2 ORDER BY "new_seq" DESC LIMIT $3 OFFSET $4`

	resultArgs := []interface{}{"abc123", "testName", 10, 3}

	qb := gqbd.NewQueryBuilder("postgres", "new_table", "new_seq", "new_id", "new_name").
		Where("new_id = ?", "abc123").
		Where("new_name = ?", "testName").
		OrderBy("new_seq", "DESC", nil).
		Offset(3).
		Limit(10)

	queryString, args, buildErr := qb.Build()

	if buildErr != nil {
		t.Fatalf("[POSTGRE_SELECT_TEST] Make Query String Error: %v", buildErr)
	}

	if queryString != resultQueryString {
		t.Fatalf("[POSTGRE_SELECT_TEST] Not Match: %v", queryString)
	}

	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[POSTGRE_SELECT_TEST] Args Not Match: %v", args)
	}
}
