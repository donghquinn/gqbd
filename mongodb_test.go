package gqbd_test

import (
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/donghquinn/gqbd"
)

type m = map[string]interface{}

func TestMongoFind(t *testing.T) {
	q, err := gqbd.MongoFind("users").
		Eq("status", "active").
		Gte("age", 18).Lt("age", 65).
		In("role", []string{"admin", "editor"}).
		Exists("deleted_at", false).
		Regex("name", "^kim", "i").
		Sort("created_at", -1).Sort("name", 1).
		Project("name", "email").Exclude("_id").
		Limit(10).Skip(20).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	want := &gqbd.MongoQuery{
		Collection: "users",
		Filter: m{
			"status":     "active",
			"age":        m{"$gte": 18, "$lt": 65},
			"role":       m{"$in": []interface{}{"admin", "editor"}},
			"deleted_at": m{"$exists": false},
			"name":       m{"$regex": "^kim", "$options": "i"},
		},
		Sort:       gqbd.D{{Key: "created_at", Value: -1}, {Key: "name", Value: 1}},
		Projection: m{"name": 1, "email": 1, "_id": 0},
		Limit:      10,
		Skip:       20,
	}
	if !reflect.DeepEqual(q, want) {
		t.Errorf("got  %#v\nwant %#v", q, want)
	}
}

func TestMongoEmptyFindMatchesAll(t *testing.T) {
	q, err := gqbd.MongoFind("users").Build()
	if err != nil {
		t.Fatal(err)
	}
	if q.Filter == nil || len(q.Filter) != 0 || q.Sort != nil || q.Projection != nil {
		t.Errorf("unexpected query %#v", q)
	}
}

func TestMongoEqDocumentValueStaysLiteral(t *testing.T) {
	// A map from user input (e.g. decoded JSON) must not become query operators.
	injected := m{"$ne": nil}
	q, _ := gqbd.MongoFind("users").Eq("password", injected).Build()
	if !reflect.DeepEqual(q.Filter, m{"password": m{"$eq": injected}}) {
		t.Errorf("got %#v", q.Filter)
	}
}

func TestMongoOr(t *testing.T) {
	q, err := gqbd.MongoFind("posts").
		Eq("published", true).
		Or(gqbd.MongoFilter().Eq("author", "kim"), gqbd.MongoFilter().Gt("likes", 100)).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	want := m{"published": true, "$or": []interface{}{m{"author": "kim"}, m{"likes": m{"$gt": 100}}}}
	if !reflect.DeepEqual(q.Filter, want) {
		t.Errorf("got %#v", q.Filter)
	}

	q, _ = gqbd.MongoFind("posts").
		Or(gqbd.MongoFilter().Eq("a", 1), gqbd.MongoFilter().Eq("b", 2)).
		Or(gqbd.MongoFilter().Eq("c", 3)).
		Build()
	want = m{"$and": []interface{}{
		m{"$or": []interface{}{m{"a": 1}, m{"b": 2}}},
		m{"$or": []interface{}{m{"c": 3}}},
	}}
	if !reflect.DeepEqual(q.Filter, want) {
		t.Errorf("got %#v", q.Filter)
	}
}

func TestMongoRawAndOtherOps(t *testing.T) {
	q, err := gqbd.MongoDelete("logs").
		Raw("$text", m{"$search": "error"}).
		Ne("level", "debug").Lte("ts", 5).Nin("src", [2]string{"a", "b"}).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	want := m{
		"$text": m{"$search": "error"},
		"level": m{"$ne": "debug"},
		"ts":    m{"$lte": 5},
		"src":   m{"$nin": []interface{}{"a", "b"}},
	}
	if !reflect.DeepEqual(q.Filter, want) || q.Collection != "logs" {
		t.Errorf("got %#v", q)
	}
}

func TestMongoUpdate(t *testing.T) {
	q, err := gqbd.MongoUpdate("users").
		Eq("_id", 1).
		Set(m{"name": "Kim", "email": "kim@example.com"}).
		Unset("temp").
		Inc("logins", 1).
		Push("tags", "new").
		AddToSet("roles", "member").
		Pull("flags", "stale").
		Upsert().
		Build()
	if err != nil {
		t.Fatal(err)
	}
	want := &gqbd.MongoQuery{
		Collection: "users",
		Filter:     m{"_id": 1},
		Update: m{
			"$set":      m{"name": "Kim", "email": "kim@example.com"},
			"$unset":    m{"temp": ""},
			"$inc":      m{"logins": 1},
			"$push":     m{"tags": "new"},
			"$addToSet": m{"roles": "member"},
			"$pull":     m{"flags": "stale"},
		},
		Upsert: true,
	}
	if !reflect.DeepEqual(q, want) {
		t.Errorf("got  %#v\nwant %#v", q, want)
	}
}

func TestMongoBuildIsRepeatable(t *testing.T) {
	b := gqbd.MongoFind("users").Gt("age", 1).Sort("a", 1)
	q1, _ := b.Build()
	q1.Filter["age"].(m)["$gt"] = 999
	q1.Sort[0].Value = -1
	q2, _ := b.Build()
	if q2.Filter["age"].(m)["$gt"] != 1 || q2.Sort[0].Value != 1 {
		t.Errorf("Build result shares state with the builder: %#v", q2)
	}
}

func TestMongoBuilderErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		b    *gqbd.MongoBuilder
		want string
	}{
		{"empty collection", gqbd.MongoFind(""), "collection"},
		{"dollar collection", gqbd.MongoFind("a$b"), "collection"},
		{"operator field", gqbd.MongoFind("c").Eq("$where", "1"), "field"},
		{"empty field", gqbd.MongoFind("c").Gt("", 1), "field"},
		{"duplicate op", gqbd.MongoFind("c").Gt("a", 1).Gt("a", 2), "duplicate"},
		{"in not slice", gqbd.MongoFind("c").In("a", 5), "slice"},
		{"or empty", gqbd.MongoFind("c").Or(), "at least one"},
		{"or nil", gqbd.MongoFind("c").Or(nil), "nil"},
		{"or child error", gqbd.MongoFind("c").Or(gqbd.MongoFilter().Eq("$x", 1)), "field"},
		{"raw or", gqbd.MongoFind("c").Raw("$or", nil), "Raw"},
		{"raw clash", gqbd.MongoFind("c").Eq("a", 1).Raw("a", 2), "Raw"},
		{"bad sort dir", gqbd.MongoFind("c").Sort("a", 2), "direction"},
		{"bad sort field", gqbd.MongoFind("c").Sort("$a", 1), "field"},
		{"dup sort", gqbd.MongoFind("c").Sort("a", 1).Sort("a", -1), "duplicate"},
		{"bad projection", gqbd.MongoFind("c").Project("$a"), "field"},
		{"negative limit", gqbd.MongoFind("c").Limit(-1), "limit"},
		{"negative skip", gqbd.MongoFind("c").Skip(-1), "skip"},
		{"sort on update", gqbd.MongoUpdate("c").Sort("a", 1), "MongoFind"},
		{"limit on delete", gqbd.MongoDelete("c").Limit(1), "MongoFind"},
		{"set on find", gqbd.MongoFind("c").Set(m{"a": 1}), "MongoUpdate"},
		{"upsert on find", gqbd.MongoFind("c").Upsert(), "MongoUpdate"},
		{"no update ops", gqbd.MongoUpdate("c").Eq("a", 1), "no update"},
		{"conflicting update", gqbd.MongoUpdate("c").Set(m{"a": 1}).Inc("a", 1), "already updated"},
		{"bad update field", gqbd.MongoUpdate("c").Inc("$a", 1), "field"},
		{"first error kept", gqbd.MongoFind("").Sort("a", 9), "collection"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.b.Build()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestMongoAggregate(t *testing.T) {
	coll, pipeline, err := gqbd.MongoAggregate("orders").
		Match(gqbd.MongoFilter().Eq("status", "paid")).
		Lookup("users", "user_id", "_id", "user").
		Unwind("user").
		Group("$user.country", m{"total": m{"$sum": "$amount"}}).
		Sort("total", -1).Sort("_id", 1).
		Project(m{"total": 1}).
		Skip(5).
		Limit(10).
		Stage("$sample", m{"size": 3}).
		Count("n").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	want := []interface{}{
		m{"$match": m{"status": "paid"}},
		m{"$lookup": m{"from": "users", "localField": "user_id", "foreignField": "_id", "as": "user"}},
		m{"$unwind": "$user"},
		m{"$group": m{"_id": "$user.country", "total": m{"$sum": "$amount"}}},
		m{"$sort": gqbd.D{{Key: "total", Value: -1}, {Key: "_id", Value: 1}}},
		m{"$project": m{"total": 1}},
		m{"$skip": int64(5)},
		m{"$limit": int64(10)},
		m{"$sample": m{"size": 3}},
		m{"$count": "n"},
	}
	if coll != "orders" || !reflect.DeepEqual(pipeline, want) {
		t.Errorf("got %s %#v", coll, pipeline)
	}

	// Build is repeatable and a later Sort does not alter an earlier result.
	a := gqbd.MongoAggregate("o").Sort("a", 1)
	_, p1, _ := a.Build()
	a.Sort("b", -1)
	_, p2, _ := a.Build()
	if len(p1[0].(m)["$sort"].(gqbd.D)) != 1 || len(p2[0].(m)["$sort"].(gqbd.D)) != 2 {
		t.Errorf("sort stages share state: %v %v", p1, p2)
	}
}

func TestMongoAggregateErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		a    *gqbd.MongoAggregateBuilder
	}{
		{"collection", gqbd.MongoAggregate("")},
		{"stage name", gqbd.MongoAggregate("o").Stage("match", nil)},
		{"nil match", gqbd.MongoAggregate("o").Match(nil)},
		{"match error", gqbd.MongoAggregate("o").Match(gqbd.MongoFilter().Eq("", 1))},
		{"group _id", gqbd.MongoAggregate("o").Group("$a", m{"_id": 1})},
		{"group field", gqbd.MongoAggregate("o").Group("$a", m{"$x": 1})},
		{"sort dir", gqbd.MongoAggregate("o").Sort("a", 0)},
		{"sort field", gqbd.MongoAggregate("o").Sort("", 1)},
		{"limit", gqbd.MongoAggregate("o").Limit(0)},
		{"skip", gqbd.MongoAggregate("o").Skip(-1)},
		{"unwind", gqbd.MongoAggregate("o").Unwind("$")},
		{"lookup field", gqbd.MongoAggregate("o").Lookup("u", "$a", "b", "c")},
		{"lookup from", gqbd.MongoAggregate("o").Lookup("", "a", "b", "c")},
		{"count", gqbd.MongoAggregate("o").Count("$n")},
		{"propagates", gqbd.MongoAggregate("").Limit(1).Count("n").Unwind("a").Lookup("u", "a", "b", "c").Group(1, nil).Sort("a", 1).Match(gqbd.MongoFilter())},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := tc.a.Build(); err == nil {
				t.Error("expected error")
			}
		})
	}
}

// golden bytes produced by go.mongodb.org/mongo-driver/v2 bson.Marshal for the
// equivalent bson.D.
const goldenBSON = "b40000000a6e756c6c00087400010273000700000068c3a96c6c6f001069333200070000001269363400000000000001000012626967000000000000010000127500090000000000000001660000000000000002400974696d65009520755ea10100000562696e000300000000010203076f69640065f0a1b2c3d4e5f601234567046172720015000000103000010000000231000200000061000003646f63001300000010620002000000106100010000000000"

func TestDMarshalBSONMatchesDriver(t *testing.T) {
	oid, _ := hex.DecodeString("65f0a1b2c3d4e5f601234567")
	var objectID [12]byte
	copy(objectID[:], oid)
	d := gqbd.D{
		{Key: "null", Value: nil}, {Key: "t", Value: true}, {Key: "s", Value: "héllo"},
		{Key: "i32", Value: 7}, {Key: "i64", Value: int64(1) << 40}, {Key: "big", Value: 1 << 40},
		{Key: "u", Value: uint64(9)}, {Key: "f", Value: 2.25},
		{Key: "time", Value: time.Date(2026, 10, 21, 12, 34, 56, 789000000, time.UTC)},
		{Key: "bin", Value: []byte{1, 2, 3}}, {Key: "oid", Value: objectID},
		{Key: "arr", Value: []interface{}{1, "a"}}, {Key: "doc", Value: gqbd.D{{Key: "b", Value: 2}, {Key: "a", Value: 1}}},
	}
	got, err := d.MarshalBSON()
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(got) != goldenBSON {
		t.Errorf("got  %x\nwant %s", got, goldenBSON)
	}
}

type named string
type kv struct {
	Key   string
	Value interface{}
}
type marshaled struct{}

func (marshaled) MarshalBSON() ([]byte, error) { return []byte{5, 0, 0, 0, 0}, nil }

type valueMarshaled struct{}

func (valueMarshaled) MarshalBSONValue() (byte, []byte, error) { return 0x08, []byte{1}, nil }

func TestDMarshalBSONTypes(t *testing.T) {
	s := "p"
	var nilPtr *int
	ok := []interface{}{
		false, int8(1), int16(1), int32(1), uint8(1), uint16(1), uint32(1), uint(1), float32(1),
		-1 << 40, named("x"), &s, nilPtr, map[named]int{"a": 1}, m{"a": 1}, []string{"a"}, [2]int{1, 2},
		[]kv{{"a", 1}}, marshaled{}, valueMarshaled{}, struct{ X named }{}.X,
	}
	for _, v := range ok {
		if _, err := (gqbd.D{{Key: "v", Value: v}}).MarshalBSON(); err != nil {
			t.Errorf("%T: unexpected error %v", v, err)
		}
	}
	bad := []interface{}{
		uint64(1) << 63, uint(1) << 63, map[int]int{1: 1}, struct{}{}, make(chan int),
		[]interface{}{make(chan int)}, m{"a": make(chan int)}, []kv{{"a", make(chan int)}},
	}
	for _, v := range bad {
		if _, err := (gqbd.D{{Key: "v", Value: v}}).MarshalBSON(); err == nil {
			t.Errorf("%T: expected error", v)
		}
	}
	if _, err := (gqbd.D{{Key: "a\x00b", Value: 1}}).MarshalBSON(); err == nil {
		t.Error("expected error for NUL in key")
	}
}

func TestMongoDBRejectedBySQLBuilders(t *testing.T) {
	if _, _, err := gqbd.BuildSelect(gqbd.MongoDB, "users").Build(); err == nil {
		t.Error("expected error for SQL builder with MongoDB")
	}
}

func TestMongoDBConnectionString(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  gqbd.DBConfig
		want string
	}{
		{"default", gqbd.DBConfig{}, "mongodb://localhost/"},
		{"full", gqbd.DBConfig{Host: "db", Port: 27017, User: "app", Password: "p@ss", DBName: "shop"}, "mongodb://app:p%40ss@db:27017/shop"},
		{"tls", gqbd.DBConfig{Host: "db", SSLMode: "require"}, "mongodb://db/?tls=true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := gqbd.BuildConnectionString(gqbd.MongoDB, tc.cfg); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
