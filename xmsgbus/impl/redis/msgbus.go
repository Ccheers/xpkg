package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ccheers/xpkg/generic/arrayx"
	"github.com/ccheers/xpkg/xmsgbus"
	"github.com/go-redis/redis/v8"
)

type MsgBus struct {
	client *redis.Client
}

func NewMsgBus(client *redis.Client) xmsgbus.IMsgBus {
	return &MsgBus{client: client}
}

func (x *MsgBus) Push(ctx context.Context, topic string, bs []byte) error {
	channels, err := x.ListChannel(ctx, topic)
	if err != nil {
		return err
	}
	var errList []error
	for _, channel := range channels {
		key := msgBusListKey(topic, channel)
		err = x.client.RPush(ctx, key, bs).Err()
		if err != nil {
			errList = append(errList, err)
		}
		x.client.Expire(ctx, key, tenMinute)
	}
	if len(errList) > 0 {
		err := fmt.Errorf("publish to %s failed: %v", topic, strings.Join(arrayx.Map(errList, func(err error) string {
			return err.Error()
		}), ". "))
		return err
	}
	return nil
}

func (x *MsgBus) Pop(ctx context.Context, topic, channel string, blockTimeout time.Duration) ([]byte, error) {
	strs, err := x.client.BLPop(ctx, blockTimeout, msgBusListKey(topic, channel)).Result()
	if err != nil {
		return nil, err
	}
	if len(strs) < 1 {
		return nil, xmsgbus.ErrNoData
	}
	return []byte(strs[1]), nil
}

func (x *MsgBus) AddChannel(ctx context.Context, topic string, channel string) error {
	return x.client.SAdd(ctx, msgBusSetKey(topic), channel).Err()
}

func (x *MsgBus) RemoveChannel(ctx context.Context, topic string, channel string) error {
	err := x.client.SRem(ctx, msgBusSetKey(topic), channel).Err()
	if err != nil {
		return err
	}
	_ = x.client.Del(ctx, msgBusListKey(topic, channel)).Err()
	return nil
}

func (x *MsgBus) ListChannel(ctx context.Context, topic string) ([]string, error) {
	return x.client.SMembers(ctx, msgBusSetKey(topic)).Result()
}
