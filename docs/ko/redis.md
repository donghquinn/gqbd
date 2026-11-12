# Redis

[English](../redis.md) | **한국어**

[← README로 돌아가기](../../README.ko.md)

Redis는 SQL 데이터베이스가 아니므로 `BuildSelect`/`BuildInsert`/... 대신 별도의 명령 빌더를 사용합니다(`gqbd.Redis`에 대해 SQL 빌더는 에러를 반환합니다). SQL 빌더처럼 클라이언트 의존성이 없으며, `Build()`는 어떤 Redis 클라이언트에서도 쓸 수 있는 명령 이름과 인자를 반환합니다.

```go
// TTL과 NX를 사용한 SET
cmd, args, err := gqbd.RedisSet(gqbd.RedisKey("session", 42), token).
    TTL(30 * time.Minute).
    NX().
    Build()
// cmd:  "SET"
// args: [session:42 <token> EX 1800 NX]

conn.Do(cmd, args...) // redigo

// go-redis는 이름과 인자를 하나의 슬라이스로 받습니다
all, err := gqbd.RedisHSet("user:1", map[string]interface{}{
    "name": "Kim",
    "age":  30,
}).BuildArgs()
// all: [HSET user:1 age 30 name Kim]   (fields are sorted for deterministic output)
rdb.Do(ctx, all...)
```

사용 가능한 빌더:

| 빌더 | 명령 |
|---------|---------|
| `RedisGet(key)` / `RedisSet(key, value)` | `GET` / `SET` (`.TTL(d)`, `.NX()`, `.XX()` 체인 가능) |
| `RedisDel(keys...)` / `RedisExists(keys...)` | `DEL` / `EXISTS` |
| `RedisExpire(key, d)` | `EXPIRE` (기간이 초 단위로 나누어떨어지지 않으면 `PEXPIRE`) |
| `RedisIncr(key)` / `RedisIncrBy(key, n)` | `INCR` / `INCRBY` |
| `RedisHSet(key, map)` / `RedisHGet` / `RedisHMGet` / `RedisHGetAll` / `RedisHDel` | 해시 명령 |
| `RedisLPush` / `RedisRPush` / `RedisLRange` | 리스트 명령 |
| `RedisSAdd` / `RedisSMembers` | 셋 명령 |
| `RedisZAdd(key, map[string]float64)` / `RedisZRange` | 정렬된 셋 명령 |
| `RedisScan(cursor)` | `SCAN` (`.Match(pattern)`, `.Count(n)` 체인 가능) |
| `NewRedisCommand(name, args...)` | 그 밖의 모든 명령 (`.Arg(...)` 체인 가능) |

`RedisKey("user", 42, "profile")`는 키 조각을 `:`로 연결합니다 → `user:42:profile`.

## 연결 문자열

```go
// Redis URL (go-redis redis.ParseURL 등에서 사용)
redisConfig := gqbd.DBConfig{
    Host:     "localhost",
    Port:     6379,
    Password: "secret",
    DBName:   "0",       // 논리 데이터베이스 번호
    SSLMode:  "disable", // 다른 값이면 rediss://를 사용합니다
}
redisURL := gqbd.BuildConnectionString(gqbd.Redis, redisConfig)
// Result: "redis://:secret@localhost:6379/0"
```

## 클라이언트

```go
import "github.com/redis/go-redis/v9"     // all, _ := cmd.BuildArgs(); rdb.Do(ctx, all...)
// 또는
import "github.com/gomodule/redigo/redis" // name, args, _ := cmd.Build(); conn.Do(name, args...)
```
