package redis

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ccheers/xpkg/generic/arrayx"
	"github.com/ccheers/xpkg/xmsgbus"
	"github.com/go-redis/redis/v8"
)

type AckData struct {
	ListKey string
	Data    string
}

type MsgBus struct {
	client *redis.Client
}

func NewMsgBus(client *redis.Client) xmsgbus.IMsgBus {
	x := &MsgBus{client: client}
	go func() {
		for {
			x.monitor(context.Background())
			time.Sleep(time.Minute)
		}
	}()
	return x
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

func (x *MsgBus) Pop(ctx context.Context, topic, channel string, blockTimeout time.Duration) ([]byte, func(), error) {
	listKey := msgBusListKey(topic, channel)
	strs, err := x.client.BLPop(ctx, blockTimeout, listKey).Result()
	if err != nil {
		return nil, nil, err
	}
	if len(strs) < 1 {
		return nil, nil, xmsgbus.ErrNoData
	}
	md5Bs := md5.Sum([]byte(strs[1]))
	ackKey := msgBusAckKey(time.Now(), hex.EncodeToString(md5Bs[:]))
	bs, _ := json.Marshal(AckData{
		ListKey: listKey,
		Data:    strs[1],
	})
	x.client.Set(ctx, ackKey, bs, time.Minute*3)
	return []byte(strs[1]), func() {
		x.client.Del(ctx, ackKey)
	}, nil
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

func (x *MsgBus) monitor(ctx context.Context) {
	defer func() {
		r := recover()
		if r != nil {
			fmt.Printf("[MsgBus][redis] monitor panic: %v, stack:\n%s\n", r, debug.Stack())
		}
	}()
	ok := x.client.SetNX(ctx, msgBusMonitorKey(), 1, time.Second*55).Val()
	if !ok {
		return
	}

	ackKeyPrefix := msgBusAckKeyPrefix(time.Now().Add(-time.Minute * 2))
	keys, err := x.client.Keys(ctx, ackKeyPrefix+"*").Result()
	if err != nil {
		return
	}
	for _, key := range keys {
		bs, err := x.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var ackData AckData
		_ = json.Unmarshal(bs, &ackData)
		x.client.Del(ctx, key)
		x.client.RPush(ctx, ackData.ListKey, ackData.Data)
	}
}
