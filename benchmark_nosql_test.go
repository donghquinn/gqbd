package gqbd

import (
	"testing"
	"time"
)

func BenchmarkRedisSet(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = RedisSet("session:42", "token-value").TTL(30 * time.Minute).NX().Build()
	}
}

func BenchmarkRedisHSet(b *testing.B) {
	fields := map[string]interface{}{"name": "John", "email": "john@example.com", "age": 30, "active": true}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = RedisHSet("user:42", fields).BuildArgs()
	}
}

func BenchmarkRedisZAdd(b *testing.B) {
	members := map[string]float64{"alice": 10, "bob": 7.5, "carol": 3}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = RedisZAdd("leaderboard", members).Build()
	}
}

func BenchmarkRedisKey(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = RedisKey("user", 42, "profile")
	}
}

func BenchmarkMongoFind(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = MongoFind("users").
			Eq("status", "active").
			Gte("age", 18).
			Lt("age", 65).
			In("role", []string{"admin", "editor"}).
			Or(MongoFilter().Eq("country", "KR"), MongoFilter().Eq("country", "US")).
			Sort("created_at", -1).
			Sort("name", 1).
			Project("name", "email").
			Limit(10).
			Skip(20).
			Build()
	}
}

func BenchmarkMongoUpdate(b *testing.B) {
	data := map[string]interface{}{"name": "John", "email": "john@example.com"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = MongoUpdate("users").
			Eq("_id", 42).
			Set(data).
			Inc("logins", 1).
			Unset("temp").
			Upsert().
			Build()
	}
}

func BenchmarkMongoDelete(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = MongoDelete("sessions").Lt("expires_at", 1700000000).Eq("revoked", true).Build()
	}
}

func BenchmarkMongoAggregate(b *testing.B) {
	group := map[string]interface{}{"total": map[string]interface{}{"$sum": "$amount"}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = MongoAggregate("orders").
			Match(MongoFilter().Eq("status", "paid")).
			Unwind("items").
			Lookup("customers", "customer_id", "_id", "customer").
			Group("$customer_id", group).
			Sort("total", -1).
			Sort("_id", 1).
			Limit(10).
			Build()
	}
}

func BenchmarkMarshalBSON(b *testing.B) {
	doc := D{
		{Key: "name", Value: "John"},
		{Key: "age", Value: 30},
		{Key: "score", Value: 9.5},
		{Key: "active", Value: true},
		{Key: "created", Value: time.Unix(1700000000, 0)},
		{Key: "tags", Value: []interface{}{"a", "b", "c"}},
		{Key: "meta", Value: map[string]interface{}{"x": 1, "y": "z"}},
		{Key: "sort", Value: D{{Key: "created_at", Value: -1}, {Key: "name", Value: 1}}},
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = doc.MarshalBSON()
	}
}

func BenchmarkESSearch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = ESSearch("products").
			Match("title", "wireless headphones").
			Term("status", "active").
			Terms("brand", []string{"sony", "bose"}).
			Gte("price", 10).
			Lt("price", 100).
			Not(ESQuery().Term("discontinued", true)).
			Sort("price", "asc").
			Source("title", "price").
			Size(10).
			From(20).
			Build()
	}
}

func BenchmarkPromSelector(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = PromSelector("http_requests_total").
			Label("job", "api").
			LabelRegex("method", "GET|POST").
			Rate("5m").
			Sum("status").
			Build()
	}
}

func BenchmarkPromHistogram(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = PromSelector("http_request_duration_seconds_bucket").
			Label("job", "api").
			Rate("5m").
			Sum("le").
			HistogramQuantile(0.99).
			Gt(0.5).
			Build()
	}
}

func BenchmarkPromBinary(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = PromSelector("errors_total").Rate("5m").Sum().
			Div(PromSelector("requests_total").Rate("5m").Sum()).
			Mul(100).
			Build()
	}
}

func BenchmarkQdrantQuery(b *testing.B) {
	vec := []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = QdrantQuery("docs").
			Nearest(vec).
			Match("lang", "en").
			MatchAny("tag", []string{"go", "db"}).
			Gte("year", 2020).
			Lt("year", 2025).
			Not(QdrantFilter().Match("draft", true)).
			Limit(5).
			WithPayload("title", "url").
			ScoreThreshold(0.75).
			Build()
	}
}

func BenchmarkConnectionString(b *testing.B) {
	cfg := DBConfig{Host: "db.example.com", Port: 5432, User: "app", Password: "p@ss w0rd", DBName: "mydb", SSLMode: "require"}
	for _, tc := range []struct {
		name string
		db   DBType
	}{
		{"PostgreSQL", PostgreSQL},
		{"MySQL", Mysql},
		{"Redis", Redis},
		{"MongoDB", MongoDB},
		{"Elasticsearch", Elasticsearch},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = BuildConnectionString(tc.db, cfg)
			}
		})
	}
}
