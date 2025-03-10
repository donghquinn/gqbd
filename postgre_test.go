package gqbd_test

import (
	"reflect"
	"strings"
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

func TestPostgresInsert(t *testing.T) {
	resultQueryString := `INSERT INTO "example_table" ("new_seq", "new_id", "new_name") VALUES ($1, $2, $3)`
	resultArgs := []interface{}{1, "abc123", "testName"}

	// INSERT 쿼리 예시
	insertData := map[string]interface{}{
		"new_seq":  1,
		"new_id":   "abc123",
		"new_name": "testName",
	}

	qb := gqbd.NewQueryBuilder("postgres", "example_table")

	queryString, args, buildErr := qb.BuildInsert(insertData)

	if buildErr != nil {
		t.Fatalf("[POSTGRE_INSERT_TEST] Make Query String Error: %v", buildErr)
	}

	if queryString != resultQueryString {
		t.Fatalf("[POSTGRE_INSERT_TEST] Not Match: %v", queryString)
	}

	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[POSTGRE_INSERT_TEST] Args Not Match: %v", args)
	}
}

func TestPostgresUpdate(t *testing.T) {
	resultQueryString := `UPDATE "example_table" SET "new_seq" = $1, "new_id" = $2, "new_name" = $3`
	resultArgs := []interface{}{1, "abc123", "testName"}

	// INSERT 쿼리 예시
	insertData := map[string]interface{}{
		"new_seq":  1,
		"new_id":   "abc123",
		"new_name": "testName",
	}

	queryString, args, buildErr := gqbd.NewQueryBuilder("postgres", "example_table").
		BuildUpdate(insertData)

	if buildErr != nil {
		t.Fatalf("[POSTGRE_UPDATE_TEST] Make Query String Error: %v", buildErr)
	}

	if queryString != resultQueryString {
		t.Fatalf("[POSTGRE_UPDATE_TEST] Not Match: %v", queryString)
	}

	if !reflect.DeepEqual(resultArgs, args) {
		t.Fatalf("[POSTGRE_UPDATE_TEST] Args Not Match: %v", args)
	}
}

func TestPostgresUpdateWithConditions(t *testing.T) {
	// 조건(WHERE 절)은 체이닝된 순서대로 생성되므로 순서가 보장됩니다.
	expectedWhere := `WHERE exam_id = $4 AND new_name = $5`
	// SET 절에 들어가야 할 각 컬럼의 할당문 (순서는 상관없음)
	expectedSetAssignments := []string{
		`"new_seq" = $1`,
		`"new_id" = $2`,
		`"new_name" = $3`,
	}
	// 전체 인자 순서도 조건까지 포함되어 있어야 합니다.
	expectedArgs := []interface{}{1, "abc123", "donghquinn", "dong15234", "testName"}

	insertData := map[string]interface{}{
		"new_seq":  1,
		"new_id":   "abc123",
		"new_name": "donghquinn",
	}

	queryString, args, buildErr := gqbd.NewQueryBuilder("postgres", "example_table").
		Where("exam_id = ?", "dong15234").
		Where("new_name = ?", "testName").
		BuildUpdate(insertData)
	if buildErr != nil {
		t.Fatalf("[POSTGRE_UPDATE_TEST] Make Query String Error: %v", buildErr)
	}

	// 인자 순서 검증
	if !reflect.DeepEqual(expectedArgs, args) {
		t.Fatalf("[POSTGRE_UPDATE_TEST] Args Not Match: got %v, expected %v", args, expectedArgs)
	}

	// 쿼리 문자열에서 UPDATE ... SET 절을 분리합니다.
	prefix := `UPDATE "example_table" SET `
	if !strings.HasPrefix(queryString, prefix) {
		t.Fatalf("[POSTGRE_UPDATE_TEST] Query does not start with expected prefix: %v", queryString)
	}
	// " WHERE " 기준으로 SET 절과 WHERE 절 분리
	parts := strings.Split(queryString, " WHERE ")
	if len(parts) != 2 {
		t.Fatalf("[POSTGRE_UPDATE_TEST] Query does not contain a proper WHERE clause: %v", queryString)
	}
	setClause := strings.TrimPrefix(parts[0], prefix)
	whereClause := "WHERE " + parts[1]

	// WHERE 절이 예상과 동일한지 비교
	if whereClause != expectedWhere {
		t.Fatalf("[POSTGRE_UPDATE_TEST] WHERE clause not match: got %v, expected %v", whereClause, expectedWhere)
	}

	// SET 절의 각 할당문은 콤마로 구분되어 있으므로 분리 후 공백을 제거
	setAssignments := strings.Split(setClause, ",")
	for i, assign := range setAssignments {
		setAssignments[i] = strings.TrimSpace(assign)
	}

	// 각 예상 할당문이 실제 SET 절에 포함되어 있는지 확인 (순서는 무시)
	for _, expectedAssign := range expectedSetAssignments {
		found := false
		for _, assign := range setAssignments {
			if assign == expectedAssign {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("[POSTGRE_UPDATE_TEST] Expected assignment %q not found in SET clause: %v", expectedAssign, setAssignments)
		}
	}
}
