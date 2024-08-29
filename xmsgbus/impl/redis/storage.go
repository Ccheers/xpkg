package redis

import (
	"github.com/ccheers/xpkg/xmsgbus"
	"github.com/ccheers/xpkg/xmsgbus/impl/redis/core"
	v8 "github.com/ccheers/xpkg/xmsgbus/impl/redis/v8"
	v9 "github.com/ccheers/xpkg/xmsgbus/impl/redis/v9"
	redisv8 "github.com/go-redis/redis/v8"
	redisv9 "github.com/redis/go-redis/v9"
)

func NewSharedStorage(client *redisv8.Client) xmsgbus.ISharedStorage {
	return core.NewSharedStorage(v8.NewRedisClientImplV8(client))
}

func NewSharedStorageV9(client *redisv9.Client) xmsgbus.ISharedStorage {
	return core.NewSharedStorage(v9.NewRedisClientImplV9(client))
}
