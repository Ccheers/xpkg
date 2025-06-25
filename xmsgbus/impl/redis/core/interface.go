package core

import (
	"context"
	"fmt"
	"time"
)

var ErrRPushAndExpire = fmt.Errorf("RPushAndExpire failed")

type IRedisClient interface {
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SRem(ctx context.Context, key string, members ...interface{}) error

	Get(ctx context.Context, key string) ([]byte, error)

	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) (string, error)
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	SetEX(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	Keys(ctx context.Context, pattern string) ([]string, error)

	Del(ctx context.Context, keys ...string) error

	BLPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error)

	RPushAndExpire(ctx context.Context, key string, value string, ttl time.Duration) error
}
