package memory

import (
	"context"
	"strconv"
	"testing"
	"time"
)

func BenchmarkMsgbus(b *testing.B) {
	msgbus := NewMsgBus()
	const (
		topic = "test"
	)

	var testContent []byte
	for i := 0; i < 1024*1024*4; i++ {
		testContent = append(testContent, byte(i%256))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < 1000; i++ {
		channel := topic + strconv.Itoa(i)
		msgbus.AddChannel(ctx, topic, channel)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				_, _, _ = msgbus.Pop(ctx, topic, channel, time.Millisecond)
				time.Sleep(time.Millisecond)
			}
		}()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msgbus.Push(ctx, topic, testContent)
	}

	for i := 0; i < 1000; i++ {
		msgbus.RemoveChannel(ctx, topic, topic+strconv.Itoa(i))
	}
}
