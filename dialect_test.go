package gqbd

import (
	"strings"
	"testing"
)

func TestFields3(t *testing.T) {
	tests := []struct {
		give string
	}{
		{give: ""},
		{give: "   "},
		{give: "users"},
		{give: "users u"},
		{give: "users AS u"},
		{give: "  users \t AS\nu  "},
		{give: "a b c d e"},
		{give: "a b"},
		{give: "a b c"},
		{give: "\xff b"},
	}
	for _, tt := range tests {
		checkFields3(t, tt.give)
	}
}

func FuzzFields3(f *testing.F) {
	for _, s := range []string{"users u", "users AS u", " a  b ", "a b", "\xffx y"} {
		f.Add(s)
	}
	f.Fuzz(checkFields3)
}

// checkFields3 compares fields3 with strings.Fields, which it replaces.
func checkFields3(t *testing.T, s string) {
	want := strings.Fields(s)
	got, n := fields3(s)
	if wantN := min(len(want), 4); n != wantN {
		t.Fatalf("fields3(%q) count = %d, want %d", s, n, wantN)
	}
	for i := 0; i < min(n, 3); i++ {
		if got[i] != want[i] {
			t.Fatalf("fields3(%q)[%d] = %q, want %q", s, i, got[i], want[i])
		}
	}
}

func TestBuildDoesNotAllocateForIdentifiers(t *testing.T) {
	qb := BuildSelect(PostgreSQL, "app.users AS u", "u.id", `we"ird`, "COUNT(*)").
		InnerJoin("orders o", "o.user_id = u.id").
		GroupBy("u.id").
		OrderBy("u.id", "asc", nil)
	want := `SELECT u."id", "we""ird", COUNT(*) FROM app."users" AS u INNER JOIN "orders" o ON o.user_id = u.id GROUP BY u."id" ORDER BY u."id" ASC`
	got, _, err := qb.Build()
	if err != nil || got != want {
		t.Fatalf("Build() = %q, %v; want %q", got, err, want)
	}
	// One allocation for the query text and one for the (empty) args slice.
	if allocs := testing.AllocsPerRun(100, func() { _, _, _ = qb.Build() }); allocs > 2 {
		t.Errorf("Build() allocs = %v, want <= 2", allocs)
	}
}
