package redis

import (
	"context"
	"strings"
	"time"

	"github.com/ccheers/xpkg/xmsgbus"
	"github.com/go-redis/redis/v8"
)

type SharedStorage struct {
	client *redis.Client
}

func NewSharedStorage(client *redis.Client) xmsgbus.ISharedStorage {
	return &SharedStorage{client: client}
}

func (x *SharedStorage) SetEx(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return x.client.SetEX(ctx, key, value, ttl).Err()
}

func (x *SharedStorage) Keys(ctx context.Context, prefix string) ([]string, error) {
	if !strings.HasSuffix(prefix, "*") {
		prefix += "*"
	}
	return x.client.Keys(ctx, prefix).Result()
}

func (x *SharedStorage) Del(ctx context.Context, key string) error {
	return x.client.Del(ctx, key).Err()
}
