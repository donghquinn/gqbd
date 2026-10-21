package gqbd_test

import (
	"strings"
	"testing"

	"github.com/donghquinn/gqbd"
)

// Expected bodies below were checked by running them on Elasticsearch 8.15 with go-elasticsearch.

func TestESSearch(t *testing.T) {
	index, body, err := gqbd.ESSearch("products").
		Match("title", "wireless headphones").
		Term("status", "active").
		Terms("brand", []string{"sony", "bose"}).
		Gte("price", 100).Lt("price", 500).
		Not(gqbd.ESQuery().Exists("discontinued")).
		Sort("price", "DESC").Source("title", "price").Size(10).From(20).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	if index != "products" {
		t.Errorf("index = %s", index)
	}
	want := `{"_source":["title","price"],"from":20,"query":{"bool":{` +
		`"filter":[{"term":{"status":{"value":"active"}}},{"terms":{"brand":["sony","bose"]}},{"range":{"price":{"gte":100,"lt":500}}}],` +
		`"must":[{"match":{"title":"wireless headphones"}}],` +
		`"must_not":[{"bool":{"filter":[{"exists":{"field":"discontinued"}}]}}]}},` +
		`"size":10,"sort":[{"price":{"order":"desc"}}]}`
	if string(body) != want {
		t.Errorf("got  %s\nwant %s", body, want)
	}
}

func TestESOrRequiresOneBranch(t *testing.T) {
	_, body, err := gqbd.ESSearch("products").Term("status", "active").
		Or(gqbd.ESQuery().Prefix("sku", "AC"), gqbd.ESQuery().MatchPhrase("title", "wireless speaker")).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	want := `{"query":{"bool":{"filter":[{"term":{"status":{"value":"active"}}}],` +
		`"must":[{"bool":{"minimum_should_match":1,"should":[` +
		`{"bool":{"filter":[{"prefix":{"sku":{"value":"AC"}}}]}},` +
		`{"bool":{"must":[{"match_phrase":{"title":"wireless speaker"}}]}}]}}]}}}`
	if string(body) != want {
		t.Errorf("got  %s\nwant %s", body, want)
	}
}

func TestESBodies(t *testing.T) {
	for _, tc := range []struct {
		name string
		b    *gqbd.ESBuilder
		want string
	}{
		{"match all search", gqbd.ESSearch("products"), `{}`},
		{"aggregation", gqbd.ESSearch("products").MultiMatch("bose", "title^2", "brand").Size(0).
			Agg("by_status", map[string]interface{}{"terms": map[string]interface{}{"field": "status"}}),
			`{"aggs":{"by_status":{"terms":{"field":"status"}}},"query":{"bool":{"must":[{"multi_match":{"fields":["title^2","brand"],"query":"bose"}}]}},"size":0}`},
		{"count raw", gqbd.ESCount("products").Raw("filter", map[string]interface{}{"wildcard": map[string]interface{}{"sku": "SN-*"}}),
			`{"query":{"bool":{"filter":[{"wildcard":{"sku":"SN-*"}}]}}}`},
		{"delete by query", gqbd.ESDeleteByQuery("products").Term("status", "archived"),
			`{"query":{"bool":{"filter":[{"term":{"status":{"value":"archived"}}}]}}}`},
		{"delete all", gqbd.ESDeleteByQuery("products").MatchAll(), `{"query":{"bool":{"must":[{"match_all":{}}]}}}`},
		{"empty subquery", gqbd.ESSearch("products").Gt("price", 1).Lte("price", 2).Or(gqbd.ESQuery()),
			`{"query":{"bool":{"filter":[{"range":{"price":{"gt":1,"lte":2}}}],"must":[{"bool":{"minimum_should_match":1,"should":[{"match_all":{}}]}}]}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, body, err := tc.b.Build()
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != tc.want {
				t.Errorf("got  %s\nwant %s", body, tc.want)
			}
		})
	}
}

func TestESTermObjectValueStaysValue(t *testing.T) {
	// A map from user input must stay in value position, where Elasticsearch
	// rejects it, rather than becoming term options such as boost.
	_, body, err := gqbd.ESSearch("products").Term("status", map[string]interface{}{"value": "x", "boost": 9}).Build()
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"query":{"bool":{"filter":[{"term":{"status":{"value":{"boost":9,"value":"x"}}}}]}}}`; string(body) != want {
		t.Errorf("got  %s\nwant %s", body, want)
	}
}

func TestESBuildIsRepeatable(t *testing.T) {
	sub := gqbd.ESQuery().Gte("price", 1)
	b := gqbd.ESSearch("products").Or(sub)
	_, first, _ := b.Build()
	sub.Lt("price", 5) // must not leak into b
	b.Gte("rating", 4)
	_, second, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(second), `"lt"`) || !strings.Contains(string(second), `"rating"`) {
		t.Errorf("unexpected body %s", second)
	}
	if strings.Contains(string(first), `"rating"`) {
		t.Errorf("first body changed: %s", first)
	}
}

func TestESErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		b    *gqbd.ESBuilder
		want string
	}{
		{"empty index", gqbd.ESSearch(""), "invalid Elasticsearch index name"},
		{"path in index", gqbd.ESSearch("../_cluster"), "invalid Elasticsearch index name"},
		{"empty field", gqbd.ESSearch("i").Term("", 1), "empty Elasticsearch field name"},
		{"empty range field", gqbd.ESSearch("i").Gt("", 1), "empty Elasticsearch field name"},
		{"duplicate range", gqbd.ESSearch("i").Gte("a", 1).Gte("a", 2), "duplicate gte"},
		{"terms not slice", gqbd.ESSearch("i").Terms("a", 1), "must be a slice"},
		{"terms empty", gqbd.ESSearch("i").Terms("a", []int{}), "at least one value"},
		{"multi match no fields", gqbd.ESSearch("i").MultiMatch("x"), "at least one field"},
		{"multi match empty field", gqbd.ESSearch("i").MultiMatch("x", "a", ""), "empty Elasticsearch field name"},
		{"or empty", gqbd.ESSearch("i").Or(), "at least one query"},
		{"or nil", gqbd.ESSearch("i").Or(nil), "nil query"},
		{"or child error", gqbd.ESSearch("i").Or(gqbd.ESQuery().Term("", 1)), "empty Elasticsearch field name"},
		{"not empty", gqbd.ESSearch("i").Not(), "at least one query"},
		{"not nil", gqbd.ESSearch("i").Not(nil), "nil query"},
		{"raw should", gqbd.ESSearch("i").Raw("should", map[string]interface{}{"a": 1}), "invalid Raw() clause"},
		{"raw empty", gqbd.ESSearch("i").Raw("must", nil), "empty query"},
		{"bad sort", gqbd.ESSearch("i").Sort("a", "up"), "must be asc or desc"},
		{"empty sort field", gqbd.ESSearch("i").Sort("", "asc"), "empty Elasticsearch field name"},
		{"sort on count", gqbd.ESCount("i").Sort("a", "asc"), "only be used with ESSearch"},
		{"size on delete", gqbd.ESDeleteByQuery("i").MatchAll().Size(1), "only be used with ESSearch"},
		{"source on count", gqbd.ESCount("i").Source("a"), "only be used with ESSearch"},
		{"negative size", gqbd.ESSearch("i").Size(-1), "must not be negative"},
		{"negative from", gqbd.ESSearch("i").From(-1), "must not be negative"},
		{"empty agg name", gqbd.ESSearch("i").Agg("", nil), "empty aggregation name"},
		{"duplicate agg", gqbd.ESSearch("i").Agg("a", nil).Agg("a", nil), "duplicate aggregation"},
		{"delete without query", gqbd.ESDeleteByQuery("i"), "requires a query"},
		{"unencodable value", gqbd.ESSearch("i").Term("a", make(chan int)), "encoding Elasticsearch query"},
		{"error kept", gqbd.ESSearch("").MatchAll().Exists("a").Prefix("a", "b"), "invalid Elasticsearch index name"},
		{"sql builder", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.b == nil {
				_, _, err = gqbd.BuildSelect(gqbd.Elasticsearch, "i").Build()
				tc.want = "not a SQL database"
			} else {
				_, _, err = tc.b.Build()
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestESConnectionString(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  gqbd.DBConfig
		want string
	}{
		{"default", gqbd.DBConfig{}, "http://localhost"},
		{"auth", gqbd.DBConfig{Host: "es", Port: 9200, User: "elastic", Password: "p@ss"}, "http://elastic:p%40ss@es:9200"},
		{"tls", gqbd.DBConfig{Host: "es", Port: 9243, SSLMode: "require"}, "https://es:9243"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := gqbd.BuildConnectionString(gqbd.Elasticsearch, tc.cfg); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
