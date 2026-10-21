package gqbd

import "testing"

func BenchmarkSelectPostgreSQL(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = BuildSelect(PostgreSQL, "users u", "u.id", "u.name", "p.title").
			LeftJoin("posts p", "p.user_id = u.id").
			Where("u.status = ?", "active").
			Where("u.age > ?", 18).
			WhereIn("u.role", []interface{}{"admin", "editor", "viewer"}).
			OrderBy("u.created_at", "DESC", nil).
			Limit(10).
			Offset(20).
			Build()
	}
}

func BenchmarkSelectMySQL(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = BuildSelect(Mysql, "users u", "u.id", "u.name", "p.title").
			LeftJoin("posts p", "p.user_id = u.id").
			Where("u.status = ?", "active").
			Where("u.age > ?", 18).
			WhereIn("u.role", []interface{}{"admin", "editor", "viewer"}).
			OrderBy("u.created_at", "DESC", nil).
			Limit(10).
			Offset(20).
			Build()
	}
}

func BenchmarkInsertPostgreSQL(b *testing.B) {
	data := map[string]interface{}{"name": "John", "email": "john@example.com", "age": 30, "active": true}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = BuildInsert(PostgreSQL, "users").Values(data).Returning("id").Build()
	}
}

func BenchmarkUpdatePostgreSQL(b *testing.B) {
	data := map[string]interface{}{"name": "John", "email": "john@example.com", "age": 30}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = BuildUpdate(PostgreSQL, "users").Set(data).Where("id = ?", 1).Where("status = ?", "active").Build()
	}
}

func BenchmarkDeleteMySQL(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = BuildDelete(Mysql, "users").Where("id = ?", 1).Build()
	}
}
