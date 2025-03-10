package gqbd_test

import (
	"reflect"
	"testing"

	gqbd "github.com/donghquinn/go-query-builder"
)

func TestMariadbSelect(t *testing.T) {
	resultQueryString := "SELECT `new_id`, `new_name` FROM `new_table`"

	qb := gqbd.NewQueryBuilder("mariadb", "new_table", "new_id", "new_name")

	queryString, _, buildErr := qb.Build()

	if buildErr != nil {
		t.Fatalf("[MARIADB_SELECT_TEST] Make Query String Error: %v", buildErr)
	}

	if queryString != resultQueryString {
		t.Fatalf("[MARIADB_SELECT_TEST] Not Match: %v", queryString)
	}
}

func TestMariadbSelectWhere(t *testing.T) {
	resultQueryString := "SELECT `new_id`, `new_name` FROM `new_table` WHERE new_id = ?"

	resultArgs := []interface{}{"abc123"}

	qb := gqbd.NewQueryBuilder("mariadb", "new_table", "new_id", "new_name").
		Where("new_id = ?", "abc123")

	queryString, args, buildErr := qb.Build()

	if buildErr != nil {
		t.Fatalf("[MARIADB_SELECT_TEST] Make Query String Error: %v", buildErr)
	}

	if queryString != resultQueryString {
		t.Fatalf("[MARIADB_SELECT_TEST] Not Match: %v", queryString)
	}
	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[MARIADB_SELECT_TEST] Args Not Match: %v", args)
	}
}

func TestMariadbSelectWhereWithOrderBy(t *testing.T) {
	resultQueryString := "SELECT `new_seq`, `new_id`, `new_name` FROM `new_table` WHERE new_id = ? ORDER BY `new_seq` DESC"

	resultArgs := []interface{}{"abc123"}

	qb := gqbd.NewQueryBuilder("mariadb", "new_table", "new_seq", "new_id", "new_name").
		Where("new_id = ?", "abc123").
		OrderBy("new_seq", "DESC", nil)

	queryString, args, buildErr := qb.Build()

	if buildErr != nil {
		t.Fatalf("[MARIADB_SELECT_TEST] Make Query String Error: %v", buildErr)
	}

	if queryString != resultQueryString {
		t.Fatalf("[MARIADB_SELECT_TEST] Not Match: %v", queryString)
	}
	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[MARIADB_SELECT_TEST] Args Not Match: %v", args)
	}
}

func TestMariadbSelectPagination(t *testing.T) {
	resultQueryString := "SELECT `new_seq`, `new_id`, `new_name` FROM `new_table` WHERE new_id = ? AND new_name = ? ORDER BY `new_seq` DESC LIMIT ? OFFSET ?"

	resultArgs := []interface{}{"abc123", "testName", 10, 3}

	qb := gqbd.NewQueryBuilder("mariadb", "new_table", "new_seq", "new_id", "new_name").
		Where("new_id = ?", "abc123").
		Where("new_name = ?", "testName").
		OrderBy("new_seq", "DESC", nil).
		Offset(3).
		Limit(10)

	queryString, args, buildErr := qb.Build()

	if buildErr != nil {
		t.Fatalf("[MARIADB_SELECT_TEST] Make Query String Error: %v", buildErr)
	}

	if queryString != resultQueryString {
		t.Fatalf("[MARIADB_SELECT_TEST] Not Match: %v", queryString)
	}

	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[MARIADB_SELECT_TEST] Args Not Match: %v", args)
	}
}

func TestMariadbInsert(t *testing.T) {
	resultQueryString := "INSERT INTO `example_table` (`new_seq`, `new_id`, `new_name`) VALUES (?, ?, ?)"
	resultArgs := []interface{}{1, "abc123", "testName"}

	// INSERT 쿼리 예시
	insertData := map[string]interface{}{
		"new_seq":  1,
		"new_id":   "abc123",
		"new_name": "testName",
	}

	qb := gqbd.NewQueryBuilder("mariadb", "example_table")

	queryString, args, buildErr := qb.BuildInsert(insertData)

	if buildErr != nil {
		t.Fatalf("[MARIADB_INSERT_TEST] Make Query String Error: %v", buildErr)
	}

	if queryString != resultQueryString {
		t.Fatalf("[MARIADB_INSERT_TEST] Not Match: %v", queryString)
	}

	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[MARIADB_INSERT_TEST] Args Not Match: %v", args)
	}
}

func TestMariadbUpdate(t *testing.T) {
	resultQueryString := "UPDATE `example_table` SET `new_seq` = ?, `new_id` = ?, `new_name` = ?"
	resultArgs := []interface{}{1, "abc123", "testName"}

	// INSERT 쿼리 예시
	updateData := map[string]interface{}{
		"new_seq":  1,
		"new_id":   "abc123",
		"new_name": "testName",
	}

	qb := gqbd.NewQueryBuilder("mariadb", "example_table")

	queryString, args, buildErr := qb.BuildUpdate(updateData)

	if buildErr != nil {
		t.Fatalf("[MARIADB_UPDATE_TEST] Make Query String Error: %v", buildErr)
	}

	if queryString != resultQueryString {
		t.Fatalf("[MARIADB_UPDATE_TEST] Not Match: %v", queryString)
	}

	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[MARIADB_UPDATE_TEST] Args Not Match: %v", args)
	}
}

func TestMariadbUpdateWithConditions(t *testing.T) {
	resultQueryString := "UPDATE `example_table` SET `new_seq` = ?, `new_id` = ?, `new_name` = ? WHERE exam_id = ? AND new_name = ?"
	resultArgs := []interface{}{1, "abc123", "donghquinn", "dong15234", "testName"}

	// INSERT 쿼리 예시
	updateData := map[string]interface{}{
		"new_seq":  1,
		"new_id":   "abc123",
		"new_name": "donghquinn",
	}

	qb := gqbd.NewQueryBuilder("mariadb", "example_table").
		Where("exam_id = ?", "dong15234").
		Where("new_name = ?", "testName")

	queryString, args, buildErr := qb.BuildUpdate(updateData)

	if buildErr != nil {
		t.Fatalf("[MARIADB_UPDATE_TEST] Make Query String Error: %v", buildErr)
	}

	if queryString != resultQueryString {
		t.Fatalf("[MARIADB_UPDATE_TEST] Not Match: %v", queryString)
	}

	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[MARIADB_UPDATE_TEST] Args Not Match: %v", args)
	}
}
