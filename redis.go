package gqbd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RedisCommand builds a Redis command. Like the SQL builders it has no client
// dependency: Build returns the command name and its arguments, ready for any
// Redis client.
//
// Example:
//
//	cmd, args, err := gqbd.RedisSet("session:42", token).TTL(30 * time.Minute).NX().Build()
//	// cmd: "SET", args: [session:42 <token> EX 1800 NX]
//
//	conn.Do(cmd, args...)            // redigo
//	all, _ := gqbd.RedisGet("k").BuildArgs()
//	rdb.Do(ctx, all...)              // go-redis
type RedisCommand struct {
	name string
	args []interface{}
	err  error
}

// NewRedisCommand creates a builder for any Redis command, for commands that
// have no dedicated constructor.
//
// Example:
//
//	gqbd.NewRedisCommand("PFADD", "visitors", "u1", "u2")
func NewRedisCommand(name string, args ...interface{}) *RedisCommand {
	c := &RedisCommand{}
	if name == "" || strings.ContainsAny(name, " \t\r\n") {
		c.err = fmt.Errorf("invalid Redis command name %q", name)
		return c
	}
	c.name = strings.ToUpper(name)
	c.args = append(c.args, args...)
	return c
}

// Arg appends raw arguments to the command.
func (c *RedisCommand) Arg(args ...interface{}) *RedisCommand {
	if c.err != nil {
		return c
	}
	c.args = append(c.args, args...)
	return c
}

// TTL appends an expiry: EX <seconds>, or PX <milliseconds> when d is not a whole
// number of seconds. Valid for SET and GETEX.
func (c *RedisCommand) TTL(d time.Duration) *RedisCommand {
	if c.err != nil {
		return c
	}
	if d <= 0 {
		c.err = fmt.Errorf("TTL must be positive, got %s", d)
		return c
	}
	if d%time.Second == 0 {
		return c.Arg("EX", int64(d/time.Second))
	}
	return c.Arg("PX", int64(d/time.Millisecond))
}

// NX appends the NX flag (only set if the key does not exist).
func (c *RedisCommand) NX() *RedisCommand { return c.Arg("NX") }

// XX appends the XX flag (only set if the key already exists).
func (c *RedisCommand) XX() *RedisCommand { return c.Arg("XX") }

// Match appends MATCH <pattern>, for SCAN, HSCAN, SSCAN and ZSCAN.
func (c *RedisCommand) Match(pattern string) *RedisCommand { return c.Arg("MATCH", pattern) }

// Count appends COUNT <n>, for SCAN, HSCAN, SSCAN and ZSCAN.
func (c *RedisCommand) Count(n int) *RedisCommand { return c.Arg("COUNT", n) }

// Build returns the command name and its arguments.
// It does not modify the builder, so it is safe to call more than once.
func (c *RedisCommand) Build() (string, []interface{}, error) {
	if c.err != nil {
		return "", nil, c.err
	}
	return c.name, append([]interface{}(nil), c.args...), nil
}

// BuildArgs returns the command name followed by its arguments as one slice,
// the form taken by go-redis's Do.
func (c *RedisCommand) BuildArgs() ([]interface{}, error) {
	if c.err != nil {
		return nil, c.err
	}
	all := make([]interface{}, 0, len(c.args)+1)
	all = append(all, c.name)
	return append(all, c.args...), nil
}

// RedisKey joins key parts with ":" (e.g. RedisKey("user", 42, "profile") is
// "user:42:profile").
func RedisKey(parts ...interface{}) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteByte(':')
		}
		switch v := p.(type) {
		case string:
			b.WriteString(v)
		case int:
			b.WriteString(strconv.Itoa(v))
		case int64:
			b.WriteString(strconv.FormatInt(v, 10))
		case uint64:
			b.WriteString(strconv.FormatUint(v, 10))
		default:
			fmt.Fprint(&b, v)
		}
	}
	return b.String()
}

func redisKeyed(name, key string, args ...interface{}) *RedisCommand {
	if key == "" {
		return &RedisCommand{err: fmt.Errorf("%s: empty key not allowed", name)}
	}
	all := make([]interface{}, 0, len(args)+1)
	all = append(all, key)
	return NewRedisCommand(name, append(all, args...)...)
}

func redisKeys(name string, keys []string) *RedisCommand {
	if len(keys) == 0 {
		return &RedisCommand{err: fmt.Errorf("%s: at least one key required", name)}
	}
	args := make([]interface{}, len(keys))
	for i, k := range keys {
		if k == "" {
			return &RedisCommand{err: fmt.Errorf("%s: empty key not allowed", name)}
		}
		args[i] = k
	}
	return NewRedisCommand(name, args...)
}

func redisMembers(name, key string, members []interface{}) *RedisCommand {
	if len(members) == 0 {
		return &RedisCommand{err: fmt.Errorf("%s: at least one value required", name)}
	}
	return redisKeyed(name, key, members...)
}

func redisFields(name, key string, fields []string) *RedisCommand {
	if len(fields) == 0 {
		return &RedisCommand{err: fmt.Errorf("%s: at least one field required", name)}
	}
	args := make([]interface{}, len(fields))
	for i, f := range fields {
		args[i] = f
	}
	return redisKeyed(name, key, args...)
}

// RedisGet builds GET key.
func RedisGet(key string) *RedisCommand { return redisKeyed("GET", key) }

// RedisSet builds SET key value. Chain TTL, NX or XX for options.
func RedisSet(key string, value interface{}) *RedisCommand { return redisKeyed("SET", key, value) }

// RedisDel builds DEL key [key ...].
func RedisDel(keys ...string) *RedisCommand { return redisKeys("DEL", keys) }

// RedisExists builds EXISTS key [key ...].
func RedisExists(keys ...string) *RedisCommand { return redisKeys("EXISTS", keys) }

// RedisExpire builds EXPIRE key seconds, or PEXPIRE key milliseconds when d is
// not a whole number of seconds.
func RedisExpire(key string, d time.Duration) *RedisCommand {
	if d <= 0 {
		return &RedisCommand{err: fmt.Errorf("EXPIRE: duration must be positive, got %s", d)}
	}
	if d%time.Second == 0 {
		return redisKeyed("EXPIRE", key, int64(d/time.Second))
	}
	return redisKeyed("PEXPIRE", key, int64(d/time.Millisecond))
}

// RedisIncr builds INCR key.
func RedisIncr(key string) *RedisCommand { return redisKeyed("INCR", key) }

// RedisIncrBy builds INCRBY key n.
func RedisIncrBy(key string, n int64) *RedisCommand { return redisKeyed("INCRBY", key, n) }

// RedisHSet builds HSET key field value [field value ...], with fields in sorted
// order so the command is deterministic.
func RedisHSet(key string, fields map[string]interface{}) *RedisCommand {
	if len(fields) == 0 {
		return &RedisCommand{err: fmt.Errorf("HSET: no fields provided")}
	}
	names := sortedKeys(fields)
	args := make([]interface{}, 0, 2*len(names))
	for _, f := range names {
		args = append(args, f, fields[f])
	}
	return redisKeyed("HSET", key, args...)
}

// RedisHGet builds HGET key field.
func RedisHGet(key, field string) *RedisCommand { return redisKeyed("HGET", key, field) }

// RedisHMGet builds HMGET key field [field ...].
func RedisHMGet(key string, fields ...string) *RedisCommand {
	return redisFields("HMGET", key, fields)
}

// RedisHGetAll builds HGETALL key.
func RedisHGetAll(key string) *RedisCommand { return redisKeyed("HGETALL", key) }

// RedisHDel builds HDEL key field [field ...].
func RedisHDel(key string, fields ...string) *RedisCommand {
	return redisFields("HDEL", key, fields)
}

// RedisLPush builds LPUSH key value [value ...].
func RedisLPush(key string, values ...interface{}) *RedisCommand {
	return redisMembers("LPUSH", key, values)
}

// RedisRPush builds RPUSH key value [value ...].
func RedisRPush(key string, values ...interface{}) *RedisCommand {
	return redisMembers("RPUSH", key, values)
}

// RedisLRange builds LRANGE key start stop.
func RedisLRange(key string, start, stop int64) *RedisCommand {
	return redisKeyed("LRANGE", key, start, stop)
}

// RedisSAdd builds SADD key member [member ...].
func RedisSAdd(key string, members ...interface{}) *RedisCommand {
	return redisMembers("SADD", key, members)
}

// RedisSMembers builds SMEMBERS key.
func RedisSMembers(key string) *RedisCommand { return redisKeyed("SMEMBERS", key) }

// RedisZAdd builds ZADD key score member [score member ...], with members in
// sorted order so the command is deterministic.
func RedisZAdd(key string, members map[string]float64) *RedisCommand {
	if len(members) == 0 {
		return &RedisCommand{err: fmt.Errorf("ZADD: no members provided")}
	}
	names := make([]string, 0, len(members))
	for m := range members {
		names = append(names, m)
	}
	sort.Strings(names)
	args := make([]interface{}, 0, 2*len(names))
	for _, m := range names {
		args = append(args, members[m], m)
	}
	return redisKeyed("ZADD", key, args...)
}

// RedisZRange builds ZRANGE key start stop.
func RedisZRange(key string, start, stop int64) *RedisCommand {
	return redisKeyed("ZRANGE", key, start, stop)
}

// RedisScan builds SCAN cursor. Chain Match and Count for options.
func RedisScan(cursor uint64) *RedisCommand { return NewRedisCommand("SCAN", cursor) }
