package redis

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
		client *redis.Client
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
				client: client,
			},
			args: args{
				ctx:          ctx,
				topic:        "test",
				channel:      "test2",
				blockTimeout: 0,
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
			bs := x.client.Get(ctx, ackKey).Val()
			t.Logf("ack key: %s, ack value: %s", ackKey, bs)
			ack()
			if !errors.Is(x.client.Get(ctx, ackKey).Err(), redis.Nil) {
				t.Errorf("ack failed, ack key: %s", ackKey)
			}
		})
	}
}
