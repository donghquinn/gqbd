# Redis

[← Back to README](../README.md)

Redis is not a SQL database, so it has its own command builders instead of `BuildSelect`/`BuildInsert`/... (those return an error for `gqbd.Redis`). Like the SQL builders, they have no client dependency: `Build()` returns the command name and its arguments for any Redis client.

```go
// SET with TTL and NX
cmd, args, err := gqbd.RedisSet(gqbd.RedisKey("session", 42), token).
    TTL(30 * time.Minute).
    NX().
    Build()
// cmd:  "SET"
// args: [session:42 <token> EX 1800 NX]

conn.Do(cmd, args...) // redigo

// go-redis takes the name and args as one slice
all, err := gqbd.RedisHSet("user:1", map[string]interface{}{
    "name": "Kim",
    "age":  30,
}).BuildArgs()
// all: [HSET user:1 age 30 name Kim]   (fields are sorted for deterministic output)
rdb.Do(ctx, all...)
```

Available builders:

| Builder | Command |
|---------|---------|
| `RedisGet(key)` / `RedisSet(key, value)` | `GET` / `SET` (chain `.TTL(d)`, `.NX()`, `.XX()`) |
| `RedisDel(keys...)` / `RedisExists(keys...)` | `DEL` / `EXISTS` |
| `RedisExpire(key, d)` | `EXPIRE` (or `PEXPIRE` for sub-second durations) |
| `RedisIncr(key)` / `RedisIncrBy(key, n)` | `INCR` / `INCRBY` |
| `RedisHSet(key, map)` / `RedisHGet` / `RedisHMGet` / `RedisHGetAll` / `RedisHDel` | Hash commands |
| `RedisLPush` / `RedisRPush` / `RedisLRange` | List commands |
| `RedisSAdd` / `RedisSMembers` | Set commands |
| `RedisZAdd(key, map[string]float64)` / `RedisZRange` | Sorted set commands |
| `RedisScan(cursor)` | `SCAN` (chain `.Match(pattern)`, `.Count(n)`) |
| `NewRedisCommand(name, args...)` | Any other command (chain `.Arg(...)`) |

`RedisKey("user", 42, "profile")` joins key parts with `:` → `user:42:profile`.

## Connection String

```go
// Redis URL (for go-redis redis.ParseURL, etc.)
redisConfig := gqbd.DBConfig{
    Host:     "localhost",
    Port:     6379,
    Password: "secret",
    DBName:   "0",       // logical database number
    SSLMode:  "disable", // anything else selects rediss://
}
redisURL := gqbd.BuildConnectionString(gqbd.Redis, redisConfig)
// Result: "redis://:secret@localhost:6379/0"
```

## Clients

```go
import "github.com/redis/go-redis/v9"     // all, _ := cmd.BuildArgs(); rdb.Do(ctx, all...)
// OR
import "github.com/gomodule/redigo/redis" // name, args, _ := cmd.Build(); conn.Do(name, args...)
```
