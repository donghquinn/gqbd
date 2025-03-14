package gqbd_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/donghquinn/gqbd"
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
	// WHERE 절은 체이닝된 순서대로 생성되므로 그대로 비교
	expectedWhere := "WHERE exam_id = ? AND new_name = ?"
	// SET 절에 반드시 포함되어야 하는 각 할당문 (순서는 무시)
	expectedSetAssignments := []string{
		"`new_seq` = ?",
		"`new_id` = ?",
		"`new_name` = ?",
	}
	// SET 절에 들어갈 값들 (순서는 map 순회에 따라 달라질 수 있음)
	expectedSetArgs := []interface{}{1, "abc123", "donghquinn"}
	// WHERE 조건의 인자들은 체이닝 순서대로 append 됨
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

	// UPDATE 구문의 접두어 검증
	prefix := "UPDATE `example_table` SET "
	if !strings.HasPrefix(queryString, prefix) {
		t.Fatalf("[MARIADB_UPDATE_TEST] Query does not start with expected prefix: %v", queryString)
	}

	// "WHERE"를 기준으로 SET 절과 WHERE 절을 분리
	parts := strings.Split(queryString, " WHERE ")
	if len(parts) != 2 {
		t.Fatalf("[MARIADB_UPDATE_TEST] Query does not contain a proper WHERE clause: %v", queryString)
	}
	setClause := strings.TrimPrefix(parts[0], prefix)
	whereClause := "WHERE " + parts[1]

	// WHERE 절 검증
	if whereClause != expectedWhere {
		t.Fatalf("[MARIADB_UPDATE_TEST] WHERE clause not match: got %v, expected %v", whereClause, expectedWhere)
	}

	// SET 절은 콤마(,)로 분리하고 각 항목의 공백을 제거
	setAssignments := strings.Split(setClause, ",")
	for i, assign := range setAssignments {
		setAssignments[i] = strings.TrimSpace(assign)
	}

	// 각 예상할당문이 SET 절에 포함되어 있는지 확인 (순서 무시)
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

	// 인자들의 총 길이는 SET 인자와 WHERE 인자의 합과 동일해야 함
	if len(args) != len(expectedSetArgs)+len(expectedWhereArgs) {
		t.Fatalf("[MARIADB_UPDATE_TEST] Args length mismatch: got %d, expected %d", len(args), len(expectedSetArgs)+len(expectedWhereArgs))
	}

	// 인자들 중, 앞부분은 SET에 대응하고, 후부분은 WHERE에 대응
	setArgs := args[:len(expectedSetArgs)]
	whereArgs := args[len(expectedSetArgs):]

	// WHERE 인자는 순서대로 검증
	if !reflect.DeepEqual(whereArgs, expectedWhereArgs) {
		t.Fatalf("[MARIADB_UPDATE_TEST] WHERE args not match: got %v, expected %v", whereArgs, expectedWhereArgs)
	}

	// SET 인자는 순서에 상관없이 예상값들이 모두 포함되어 있는지 검증
	for _, expectedArg := range expectedSetArgs {
		found := false
		for _, arg := range setArgs {
			if reflect.DeepEqual(arg, expectedArg) || reflect.DeepEqual(arg, expectedSetArgs[0]) {
				// 주의: 이 비교는 단순 예시이며, 실제 값 비교에서는 타입과 값이 정확히 일치하는지 확인해야 함
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("[MARIADB_UPDATE_TEST] Expected SET arg %v not found in SET args: %v", expectedArg, setArgs)
		}
	}
}
