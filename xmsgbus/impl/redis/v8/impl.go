package v8

import (
	"context"
	"time"

	"github.com/ccheers/xpkg/xmsgbus/impl/redis/core"
	"github.com/go-redis/redis/v8"
)

type RedisClientImplV8 struct {
	client *redis.Client
}

func NewRedisClientImplV8(client *redis.Client) core.IRedisClient {
	return &RedisClientImplV8{client: client}
}

func (x *RedisClientImplV8) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return x.client.SAdd(ctx, key, members...).Err()
}

func (x *RedisClientImplV8) SMembers(ctx context.Context, key string) ([]string, error) {
	return x.client.SMembers(ctx, key).Result()
}

func (x *RedisClientImplV8) SRem(ctx context.Context, key string, members ...interface{}) error {
	return x.client.SRem(ctx, key, members...).Err()
}

func (x *RedisClientImplV8) Get(ctx context.Context, key string) ([]byte, error) {
	return x.client.Get(ctx, key).Bytes()
}

func (x *RedisClientImplV8) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) (string, error) {
	return x.client.Set(ctx, key, value, expiration).Result()
}

func (x *RedisClientImplV8) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return x.client.SetNX(ctx, key, value, expiration).Result()
}

func (x *RedisClientImplV8) SetEX(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return x.client.SetEX(ctx, key, value, expiration).Err()
}

func (x *RedisClientImplV8) Keys(ctx context.Context, pattern string) ([]string, error) {
	return x.client.Keys(ctx, pattern).Result()
}

func (x *RedisClientImplV8) Del(ctx context.Context, keys ...string) error {
	return x.client.Del(ctx, keys...).Err()
}

func (x *RedisClientImplV8) BLPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	return x.client.BLPop(ctx, timeout, keys...).Result()
}

func (x *RedisClientImplV8) RPushAndExpire(ctx context.Context, key string, value string, ttl time.Duration) error {
	return x.rpushAndExpire(ctx, key, value, ttl)
}

const luaScript = `
local key = KEYS[1]
local value = ARGV[1]
local expiration = tonumber(ARGV[2])

local result = redis.call('RPUSH', key, value)
if result > 0 then
    redis.call('EXPIRE', key, expiration)
    return result
else
    return 0  -- 表示操作失败
end
`

var rpushAndExpireScript = redis.NewScript(luaScript)

func (x *RedisClientImplV8) rpushAndExpire(ctx context.Context, key string, value string, ttl time.Duration) error {
	result, err := rpushAndExpireScript.Run(ctx, x.client, []string{key}, value, int(ttl.Seconds())).Int()
	if err != nil {
		return err
	}
	if result == 0 {
		return core.ErrRPushAndExpire
	}
	return nil
}
