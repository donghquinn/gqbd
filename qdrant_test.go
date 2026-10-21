package gqbd_test

import (
	"strings"
	"testing"

	"github.com/donghquinn/gqbd"
)

// Expected bodies below were checked by sending them to Qdrant 1.15 over the REST API.

func TestQdrantRequests(t *testing.T) {
	const uuid = "5c56c793-69f3-4fbf-87e6-c4bf54c28c26"
	for _, tc := range []struct {
		name     string
		b        *gqbd.QdrantBuilder
		wantPath string
		want     string
	}{
		{"search", gqbd.QdrantQuery("docs").Nearest([]float32{1, 0, 0}).Match("lang", "en").Gte("price", 10).Lt("price", 50).Limit(5).WithPayload("lang", "price"),
			"/collections/docs/points/query",
			`{"filter":{"must":[{"key":"lang","match":{"value":"en"}},{"key":"price","range":{"gte":10,"lt":50}}]},"limit":5,"query":[1,0,0],"with_payload":["lang","price"]}`},
		{"search options", gqbd.QdrantQuery("docs").Nearest([]float32{1, 0, 0}).Using("image").MatchAny("lang", []string{"ko", "ja"}).WithVector().WithPayload().ScoreThreshold(-1).Exact(),
			"/collections/docs/points/query",
			`{"filter":{"must":[{"key":"lang","match":{"any":["ko","ja"]}}]},"params":{"exact":true},"query":[1,0,0],"score_threshold":-1,"using":"image","with_payload":true,"with_vector":true}`},
		{"nearest point with not", gqbd.QdrantQuery("docs").NearestPoint(1).MatchExcept("lang", []string{"ko"}).Not(gqbd.QdrantFilter().IsEmpty("tags")).Limit(3),
			"/collections/docs/points/query",
			`{"filter":{"must":[{"key":"lang","match":{"except":["ko"]}}],"must_not":[{"must":[{"is_empty":{"key":"tags"}}]}]},"limit":3,"query":1}`},
		{"or", gqbd.QdrantQuery("docs").Match("lang", "en").Or(gqbd.QdrantFilter().Match("vip", true), gqbd.QdrantFilter().HasID(2)),
			"/collections/docs/points/query",
			`{"filter":{"must":[{"key":"lang","match":{"value":"en"}},{"should":[{"must":[{"key":"vip","match":{"value":true}}]},{"must":[{"has_id":[2]}]}]}]}}`},
		{"text and null", gqbd.QdrantQuery("docs").MatchText("body", "good").IsNull("note"),
			"/collections/docs/points/query",
			`{"filter":{"must":[{"key":"body","match":{"text":"good"}},{"is_null":{"key":"note"}}]}}`},
		{"ids and paging", gqbd.QdrantQuery("docs").HasID(1, uuid).Offset(1).Limit(1),
			"/collections/docs/points/query",
			`{"filter":{"must":[{"has_id":[1,"` + uuid + `"]}]},"limit":1,"offset":1}`},
		{"integer id types", gqbd.QdrantQuery("docs").HasID(int64(3), int32(4), uint64(5), uint(6)),
			"/collections/docs/points/query", `{"filter":{"must":[{"has_id":[3,4,5,6]}]}}`},
		{"count", gqbd.QdrantCount("docs").Gt("price", 20).Lte("price", 100),
			"/collections/docs/points/count", `{"filter":{"must":[{"key":"price","range":{"gt":20,"lte":100}}]}}`},
		{"count all", gqbd.QdrantCount("docs"), "/collections/docs/points/count", `{}`},
		{"delete", gqbd.QdrantDelete("docs").Match("lang", "ja"),
			"/collections/docs/points/delete", `{"filter":{"must":[{"key":"lang","match":{"value":"ja"}}]}}`},
		{"collection name is encoded", gqbd.QdrantCount("a/../b c"), "/collections/a%2F..%2Fb%20c/points/count", `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path, body, err := tc.b.Build()
			if err != nil {
				t.Fatal(err)
			}
			if path != tc.wantPath {
				t.Errorf("path = %s, want %s", path, tc.wantPath)
			}
			if string(body) != tc.want {
				t.Errorf("got  %s\nwant %s", body, tc.want)
			}
		})
	}
}

func TestQdrantBuildIsRepeatable(t *testing.T) {
	vec := []float32{1, 2}
	b := gqbd.QdrantQuery("docs").Nearest(vec).Gte("price", 1)
	_, first, _ := b.Build()
	vec[0] = 9 // the builder keeps its own copy
	b.Lt("price", 5)
	_, second, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != `{"filter":{"must":[{"key":"price","range":{"gte":1}}]},"query":[1,2]}` {
		t.Errorf("first = %s", first)
	}
	if string(second) != `{"filter":{"must":[{"key":"price","range":{"gte":1,"lt":5}}]},"query":[1,2]}` {
		t.Errorf("second = %s", second)
	}
}

func TestQdrantErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		b    *gqbd.QdrantBuilder
		want string
	}{
		{"empty collection", gqbd.QdrantQuery(""), "empty Qdrant collection name"},
		{"empty key", gqbd.QdrantQuery("c").Match("", 1), "empty Qdrant payload key"},
		{"object match", gqbd.QdrantQuery("c").Match("a", map[string]interface{}{"any": []string{"x"}}), "must be a string, integer or bool"},
		{"float match", gqbd.QdrantQuery("c").Match("a", 1.5), "must be a string, integer or bool"},
		{"any not slice", gqbd.QdrantQuery("c").MatchAny("a", "x"), "must be a slice"},
		{"any empty", gqbd.QdrantQuery("c").MatchAny("a", []string{}), "at least one value"},
		{"except bad value", gqbd.QdrantQuery("c").MatchExcept("a", []interface{}{nil}), "must be strings, integers or bools"},
		{"empty key condition", gqbd.QdrantQuery("c").IsNull(""), "empty Qdrant payload key"},
		{"no ids", gqbd.QdrantQuery("c").HasID(), "at least one ID"},
		{"negative id", gqbd.QdrantQuery("c").HasID(-1), "invalid Qdrant point ID"},
		{"float id", gqbd.QdrantQuery("c").HasID(1.5), "invalid Qdrant point ID"},
		{"empty range key", gqbd.QdrantQuery("c").Gt("", 1), "empty Qdrant payload key"},
		{"duplicate range", gqbd.QdrantQuery("c").Lte("a", 1).Lte("a", 2), "duplicate lte"},
		{"or empty", gqbd.QdrantQuery("c").Or(), "at least one filter"},
		{"or nil", gqbd.QdrantQuery("c").Or(nil), "nil filter"},
		{"or empty filter", gqbd.QdrantQuery("c").Or(gqbd.QdrantFilter()), "empty filter"},
		{"not child error", gqbd.QdrantQuery("c").Not(gqbd.QdrantFilter().Match("", 1)), "empty Qdrant payload key"},
		{"empty vector", gqbd.QdrantQuery("c").Nearest(nil), "non-empty vector"},
		{"bad nearest point", gqbd.QdrantQuery("c").NearestPoint(""), "invalid Qdrant point ID"},
		{"two queries", gqbd.QdrantQuery("c").Nearest([]float32{1}).NearestPoint(1), "already set"},
		{"nearest on count", gqbd.QdrantCount("c").Nearest([]float32{1}), "only be used with QdrantQuery"},
		{"limit on delete", gqbd.QdrantDelete("c").Match("a", 1).Limit(1), "only be used with QdrantQuery"},
		{"exact on count", gqbd.QdrantCount("c").Exact(), "only be used with QdrantQuery"},
		{"empty using", gqbd.QdrantQuery("c").Using(""), "empty vector name"},
		{"zero limit", gqbd.QdrantQuery("c").Limit(0), "must be positive"},
		{"negative offset", gqbd.QdrantQuery("c").Offset(-1), "must not be negative"},
		{"delete without filter", gqbd.QdrantDelete("c"), "requires a filter"},
		{"build filter", gqbd.QdrantFilter().Match("a", 1), "only for Or and Not"},
		{"unencodable range", gqbd.QdrantQuery("c").Gt("a", make(chan int)), "encoding Qdrant request"},
		{"error kept", gqbd.QdrantQuery("").MatchText("a", "b").IsEmpty("a").Not(gqbd.QdrantFilter()).WithPayload().WithVector().ScoreThreshold(1), "empty Qdrant collection name"},
		{"sql builder", nil, "not a SQL database"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.b == nil {
				_, _, err = gqbd.BuildSelect(gqbd.Qdrant, "c").Build()
			} else {
				_, _, err = tc.b.Build()
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestQdrantConnectionString(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  gqbd.DBConfig
		want string
	}{
		// The API key goes in the api-key header, never the address.
		{"default", gqbd.DBConfig{Host: "localhost", Port: 6333, Password: "key"}, "http://localhost:6333"},
		{"tls", gqbd.DBConfig{Host: "xyz.cloud.qdrant.io", Port: 6333, SSLMode: "require"}, "https://xyz.cloud.qdrant.io:6333"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := gqbd.BuildConnectionString(gqbd.Qdrant, tc.cfg); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
