package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/ccheers/xpkg/xmsgbus/impl/redis/core"
	v8 "github.com/ccheers/xpkg/xmsgbus/impl/redis/v8"
	v9 "github.com/ccheers/xpkg/xmsgbus/impl/redis/v9"
	redisv8 "github.com/go-redis/redis/v8"
)

func TestPublishReceiveAndAcknowledge(t *testing.T) {
	for _, version := range []string{"v8", "v9"} {
		t.Run(version, func(t *testing.T) {
			ctx := context.Background()
			raw := testRedis(t)
			var client core.IRedisClient = v9.NewRedisClientImplV9(raw)
			if version == "v8" {
				r8 := redisv8.NewClient(&redisv8.Options{Addr: raw.Options().Addr})
				t.Cleanup(func() { r8.Close() })
				client = v8.NewRedisClientImplV8(r8)
			}
			bus := core.NewMsgBus(client)
			t.Cleanup(func() { bus.Close() })
			if err := bus.AddChannel(ctx, "topic", "channel"); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 10; i++ {
				if err := bus.Push(ctx, "topic", []byte{0, 255, byte(i)}); err != nil {
					t.Fatal(err)
				}
				data, ack, err := bus.Pop(ctx, "topic", "channel", time.Second)
				if err != nil || len(data) != 3 || data[2] != byte(i) {
					t.Fatalf("data=%v err=%v", data, err)
				}
				ack()
				ack()
			}
		})
	}
}
