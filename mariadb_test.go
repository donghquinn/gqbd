package gqbd_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/donghquinn/gqbd"
)

/*
NewQueryBuilder

@ dbType: MariaDB
@ table: Table name
@ columns: Columns to select (variadic)
@ Return: *QueryBuilder instance
*/
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

/*
Where

@ condition: Condition string with placeholders
@ args: Query parameters
@ Return: *QueryBuilder with WHERE clause added
*/
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

/*
OrderBy

@ column: Column name to order by
@ direction: Order direction ("ASC" or "DESC")
@ allowedColumns: Map of allowed columns for ordering
@ Return: *QueryBuilder with ORDER BY clause added
*/
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

/*
Limit & Offset

@ limit: Maximum number of rows to return
@ offset: Number of rows to skip
@ Return: Final SELECT query string, arguments slice, and error if any
*/
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

/*
BuildInsert

@ data: Map of column names to values for INSERT
@ Return: INSERT query string, arguments slice, and error if any
*/
func TestMariadbInsert(t *testing.T) {
	resultQueryString := "INSERT INTO `example_table` (`new_seq`) VALUES (?)"
	resultArgs := []interface{}{1}

	insertData := map[string]interface{}{
		"new_seq": 1,
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

/*
BuildUpdate

@ data: Map of column names to values for UPDATE
@ Return: UPDATE query string, arguments slice, and error if any
*/
func TestMariadbUpdate(t *testing.T) {
	resultQueryString := "UPDATE `example_table` SET `new_seq` = ?"
	resultArgs := []interface{}{1}

	updateData := map[string]interface{}{
		"new_seq": 1,
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

/*
BuildUpdate with Conditions

@ data: Map of column names to values for UPDATE
@ Return: UPDATE query string, arguments slice, and error if any
*/
func TestMariadbUpdateWithConditions(t *testing.T) {
	expectedWhere := "WHERE exam_id = ? AND new_name = ?"
	expectedSetAssignments := []string{
		"`new_seq` = ?",
		"`new_id` = ?",
		"`new_name` = ?",
	}
	expectedSetArgs := []interface{}{1, "abc123", "donghquinn"}
	expectedWhereArgs := []interface{}{"dong15234", "testName"}

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

	prefix := "UPDATE `example_table` SET "
	if !strings.HasPrefix(queryString, prefix) {
		t.Fatalf("[MARIADB_UPDATE_TEST] Query does not start with expected prefix: %v", queryString)
	}
	parts := strings.Split(queryString, " WHERE ")
	if len(parts) != 2 {
		t.Fatalf("[MARIADB_UPDATE_TEST] Query does not contain a proper WHERE clause: %v", queryString)
	}
	setClause := strings.TrimPrefix(parts[0], prefix)
	whereClause := "WHERE " + parts[1]
	if whereClause != expectedWhere {
		t.Fatalf("[MARIADB_UPDATE_TEST] WHERE clause not match: got %v, expected %v", whereClause, expectedWhere)
	}

	setAssignments := strings.Split(setClause, ",")
	for i, assign := range setAssignments {
		setAssignments[i] = strings.TrimSpace(assign)
	}
	for _, expectedAssign := range expectedSetAssignments {
		found := false
		for _, assign := range setAssignments {
			if assign == expectedAssign {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("[MARIADB_UPDATE_TEST] Expected assignment %q not found in SET clause: %v", expectedAssign, setAssignments)
		}
	}

	if len(args) != len(expectedSetArgs)+len(expectedWhereArgs) {
		t.Fatalf("[MARIADB_UPDATE_TEST] Args length mismatch: got %d, expected %d", len(args), len(expectedSetArgs)+len(expectedWhereArgs))
	}
	setArgs := args[:len(expectedSetArgs)]
	whereArgs := args[len(expectedSetArgs):]
	if !reflect.DeepEqual(whereArgs, expectedWhereArgs) {
		t.Fatalf("[MARIADB_UPDATE_TEST] WHERE args not match: got %v, expected %v", whereArgs, expectedWhereArgs)
	}

	for _, expectedArg := range expectedSetArgs {
		found := false
		for _, arg := range setArgs {
			if reflect.DeepEqual(arg, expectedArg) || reflect.DeepEqual(arg, expectedSetArgs[0]) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("[MARIADB_UPDATE_TEST] Expected SET arg %v not found in SET args: %v", expectedArg, setArgs)
		}
	}
}

/*
generatePlaceholders

@ dbType: MariaDB
@ startIdx: Starting index for placeholders
@ count: Number of placeholders to generate
@ Return: String of placeholders separated by comma
*/
func TestGeneratePlaceholdersMariaDB(t *testing.T) {
	placeholders := gqbd.GeneratePlaceholders("mariadb", 1, 3)
	expected := "?, ?, ?"
	if placeholders != expected {
		t.Errorf("expected placeholders %s, got %s", expected, placeholders)
	}
}
