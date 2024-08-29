package core

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func TestMsgBus_Pop(t *testing.T) {
	ctx := context.TODO()
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   0,
	})
	msg := []byte("test")
	type fields struct {
		client IRedisClient
	}
	type args struct {
		ctx          context.Context
		topic        string
		channel      string
		blockTimeout time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		want1   func()
		wantErr bool
	}{
		{
			name: "1",
			fields: fields{
				client: NewRedisClientImplV8(client),
			},
			args: args{
				ctx:          ctx,
				topic:        "test",
				channel:      "test2",
				blockTimeout: time.Second,
			},
			want:    msg,
			want1:   nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := &MsgBus{
				client: tt.fields.client,
			}
			_ = x.AddChannel(ctx, tt.args.topic, tt.args.channel)
			for i := 0; i < 10; i++ {
				_ = x.Push(ctx, tt.args.topic, msg)
				got, ack, err := x.Pop(tt.args.ctx, tt.args.topic, tt.args.channel, tt.args.blockTimeout)
				if (err != nil) != tt.wantErr {
					t.Errorf("Pop() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Pop() got = %v, want %v", got, tt.want)
				}
				// 如果没有 ack 则 有数据在
				md5Bs := md5.Sum(msg)
				ackKey := msgBusAckKey(time.Now(), hex.EncodeToString(md5Bs[:]))
				bs, _ := x.client.Get(ctx, ackKey)
				t.Logf("ack key: %s, ack value: %s", ackKey, bs)
				ack()
				_, err = x.client.Get(ctx, ackKey)
				if !errors.Is(err, redis.Nil) {
					t.Errorf("ack failed, ack key: %s", ackKey)
				}
			}
		})
	}
}

type RedisClientImplV8 struct {
	client *redis.Client
}

func NewRedisClientImplV8(client *redis.Client) IRedisClient {
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

var (
	rpushAndExpireScript = redis.NewScript(luaScript)
)

func (x *RedisClientImplV8) rpushAndExpire(ctx context.Context, key string, value string, ttl time.Duration) error {
	result, err := rpushAndExpireScript.Run(ctx, x.client, []string{key}, value, int(ttl.Seconds())).Int()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrRPushAndExpire
	}
	return nil
}
