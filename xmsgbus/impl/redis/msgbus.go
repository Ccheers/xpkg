package redis

import (
	"github.com/ccheers/xpkg/xmsgbus"
	"github.com/ccheers/xpkg/xmsgbus/impl/redis/core"
	v8 "github.com/ccheers/xpkg/xmsgbus/impl/redis/v8"
	v9 "github.com/ccheers/xpkg/xmsgbus/impl/redis/v9"
	redisv8 "github.com/go-redis/redis/v8"
	redisv9 "github.com/redis/go-redis/v9"
)

func NewMsgBus(client *redisv8.Client) xmsgbus.IMsgBus {
	return core.NewMsgBus(v8.NewRedisClientImplV8(client))
}

func NewMsgBusV9(client *redisv9.Client) xmsgbus.IMsgBus {
	return core.NewMsgBus(v9.NewRedisClientImplV9(client))
}
