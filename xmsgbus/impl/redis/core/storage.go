package core

import (
	"context"
	"strings"
	"time"

	"github.com/ccheers/xpkg/xmsgbus"
)

type SharedStorage struct {
	client IRedisClient
}

func NewSharedStorage(client IRedisClient) xmsgbus.ISharedStorage {
	return &SharedStorage{client: client}
}

func (x *SharedStorage) SetEx(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return x.client.SetEX(ctx, key, value, ttl)
}

func (x *SharedStorage) Keys(ctx context.Context, prefix string) ([]string, error) {
	if !strings.HasSuffix(prefix, "*") {
		prefix += "*"
	}
	return x.client.Keys(ctx, prefix)
}

func (x *SharedStorage) Del(ctx context.Context, key string) error {
	return x.client.Del(ctx, key)
}
