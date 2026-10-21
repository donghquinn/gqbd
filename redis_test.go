package gqbd_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/donghquinn/gqbd"
)

func TestRedisCommands(t *testing.T) {
	for _, tc := range []struct {
		name     string
		cmd      *gqbd.RedisCommand
		wantName string
		wantArgs []interface{}
	}{
		{"get", gqbd.RedisGet("k"), "GET", []interface{}{"k"}},
		{"set", gqbd.RedisSet("k", "v"), "SET", []interface{}{"k", "v"}},
		{"set ex nx", gqbd.RedisSet("k", "v").TTL(30 * time.Minute).NX(), "SET", []interface{}{"k", "v", "EX", int64(1800), "NX"}},
		{"set px xx", gqbd.RedisSet("k", "v").TTL(1500 * time.Millisecond).XX(), "SET", []interface{}{"k", "v", "PX", int64(1500), "XX"}},
		{"del", gqbd.RedisDel("a", "b"), "DEL", []interface{}{"a", "b"}},
		{"exists", gqbd.RedisExists("a"), "EXISTS", []interface{}{"a"}},
		{"expire", gqbd.RedisExpire("k", time.Minute), "EXPIRE", []interface{}{"k", int64(60)}},
		{"pexpire", gqbd.RedisExpire("k", 250*time.Millisecond), "PEXPIRE", []interface{}{"k", int64(250)}},
		{"incr", gqbd.RedisIncr("c"), "INCR", []interface{}{"c"}},
		{"incrby", gqbd.RedisIncrBy("c", 5), "INCRBY", []interface{}{"c", int64(5)}},
		{"hset", gqbd.RedisHSet("u:1", map[string]interface{}{"name": "Kim", "age": 30}), "HSET", []interface{}{"u:1", "age", 30, "name", "Kim"}},
		{"hget", gqbd.RedisHGet("u:1", "name"), "HGET", []interface{}{"u:1", "name"}},
		{"hmget", gqbd.RedisHMGet("u:1", "name", "age"), "HMGET", []interface{}{"u:1", "name", "age"}},
		{"hgetall", gqbd.RedisHGetAll("u:1"), "HGETALL", []interface{}{"u:1"}},
		{"hdel", gqbd.RedisHDel("u:1", "age"), "HDEL", []interface{}{"u:1", "age"}},
		{"lpush", gqbd.RedisLPush("l", 1, 2), "LPUSH", []interface{}{"l", 1, 2}},
		{"rpush", gqbd.RedisRPush("l", "x"), "RPUSH", []interface{}{"l", "x"}},
		{"lrange", gqbd.RedisLRange("l", 0, -1), "LRANGE", []interface{}{"l", int64(0), int64(-1)}},
		{"sadd", gqbd.RedisSAdd("s", "a", "b"), "SADD", []interface{}{"s", "a", "b"}},
		{"smembers", gqbd.RedisSMembers("s"), "SMEMBERS", []interface{}{"s"}},
		{"zadd", gqbd.RedisZAdd("z", map[string]float64{"bob": 2, "amy": 1.5}), "ZADD", []interface{}{"z", 1.5, "amy", 2.0, "bob"}},
		{"zrange", gqbd.RedisZRange("z", 0, 10), "ZRANGE", []interface{}{"z", int64(0), int64(10)}},
		{"scan", gqbd.RedisScan(0).Match("user:*").Count(100), "SCAN", []interface{}{uint64(0), "MATCH", "user:*", "COUNT", 100}},
		{"generic", gqbd.NewRedisCommand("pfadd", "v", "u1").Arg("u2"), "PFADD", []interface{}{"v", "u1", "u2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name, args, err := tc.cmd.Build()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if name != tc.wantName || !reflect.DeepEqual(args, tc.wantArgs) {
				t.Errorf("got %s %v, want %s %v", name, args, tc.wantName, tc.wantArgs)
			}
		})
	}
}

func TestRedisCommandErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  *gqbd.RedisCommand
	}{
		{"empty name", gqbd.NewRedisCommand("")},
		{"name with space", gqbd.NewRedisCommand("SET k")},
		{"empty key", gqbd.RedisGet("")},
		{"no keys", gqbd.RedisDel()},
		{"empty key in list", gqbd.RedisDel("a", "")},
		{"bad ttl", gqbd.RedisSet("k", "v").TTL(0)},
		{"bad expire", gqbd.RedisExpire("k", -time.Second)},
		{"empty hset", gqbd.RedisHSet("k", nil)},
		{"empty hmget", gqbd.RedisHMGet("k")},
		{"empty lpush", gqbd.RedisLPush("k")},
		{"empty zadd", gqbd.RedisZAdd("k", nil)},
		{"error propagates", gqbd.RedisGet("").Arg("x").NX()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := tc.cmd.Build(); err == nil {
				t.Error("expected error from Build")
			}
			if _, err := tc.cmd.BuildArgs(); err == nil {
				t.Error("expected error from BuildArgs")
			}
		})
	}
}

func TestRedisBuildArgsAndRepeatable(t *testing.T) {
	cmd := gqbd.RedisSet("k", "v").TTL(time.Second)
	all, err := cmd.BuildArgs()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(all, []interface{}{"SET", "k", "v", "EX", int64(1)}) {
		t.Errorf("BuildArgs = %v", all)
	}
	_, a1, _ := cmd.Build()
	a1[0] = "mutated"
	_, a2, _ := cmd.Build()
	if a2[0] != "k" {
		t.Error("Build must return a copy of the args")
	}
}

func TestRedisKey(t *testing.T) {
	if got := gqbd.RedisKey("user", 42, int64(7), uint64(8), "profile", 1.5); got != "user:42:7:8:profile:1.5" {
		t.Errorf("RedisKey = %s", got)
	}
}

func TestRedisRejectedBySQLBuilders(t *testing.T) {
	if _, _, err := gqbd.BuildSelect(gqbd.Redis, "users").Build(); err == nil {
		t.Error("expected error for SQL builder with Redis")
	}
}

func TestRedisConnectionString(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  gqbd.DBConfig
		want string
	}{
		{"default", gqbd.DBConfig{}, "redis://localhost"},
		{"full", gqbd.DBConfig{Host: "cache", Port: 6379, User: "app", Password: "p@ss:w/rd", DBName: "2"}, "redis://app:p%40ss%3Aw%2Frd@cache:6379/2"},
		{"password only", gqbd.DBConfig{Host: "cache", Password: "secret"}, "redis://:secret@cache"},
		{"user only", gqbd.DBConfig{Host: "cache", User: "app"}, "redis://app@cache"},
		{"tls", gqbd.DBConfig{Host: "cache", Port: 6380, SSLMode: "require"}, "rediss://cache:6380"},
		{"ipv6", gqbd.DBConfig{Host: "::1", Port: 6379}, "redis://[::1]:6379"},
		{"ipv6 no port", gqbd.DBConfig{Host: "::1"}, "redis://[::1]"},
		{"unicode password", gqbd.DBConfig{Host: "h", Password: "비밀 번호"}, "redis://:%EB%B9%84%EB%B0%80%20%EB%B2%88%ED%98%B8@h"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := gqbd.BuildConnectionString(gqbd.Redis, tc.cfg); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
