package gqbd_test

import (
	"reflect"
	"testing"

	"github.com/donghquinn/go-query-builder/builder"
)

func TestSelect(t *testing.T) {
	resultQueryString := `SELECT "new_id", "new_name" FROM "new_table"`

	qb := builder.NewQueryBuilder("postgres", "new_table", "new_id", "new_name")

	queryString, _ := qb.Build()

	if queryString != resultQueryString {
		t.Fatalf("[SELECT_TEST] Not Match: %v", queryString)
	}
}

func TestSelectWhere(t *testing.T) {
	resultQueryString := `SELECT "new_id", "new_name" FROM "new_table" WHERE new_id = $1`

	resultArgs := []interface{}{"abc123"}

	qb := builder.NewQueryBuilder("postgres", "new_table", "new_id", "new_name").
		Where("new_id = ?", "abc123")

	queryString, args := qb.Build()

	if queryString != resultQueryString {
		t.Fatalf("[SELECT_TEST] Not Match: %v", queryString)
	}
	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[SELECT_TEST] Args Not Match: %v", args)
	}
}

func TestSelectWhereWithOrderBy(t *testing.T) {
	resultQueryString := `SELECT "new_id", "new_name" FROM "new_table" WHERE new_id = $1 ORDER BY "new_seq" DESC`

	resultArgs := []interface{}{"abc123"}

	qb := builder.NewQueryBuilder("postgres", "new_seq", "new_table", "new_id", "new_name").
		Where("new_id = ?", "abc123").
		OrderBy("new_seq", "DESC", nil)

	queryString, args := qb.Build()

	if queryString != resultQueryString {
		t.Fatalf("[SELECT_TEST] Not Match: %v", queryString)
	}
	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[SELECT_TEST] Args Not Match: %v", args)
	}
}
