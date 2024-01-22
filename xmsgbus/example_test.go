package xmsgbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/ccheers/xpkg/xlogger"
	"github.com/ccheers/xpkg/xmsgbus"
	"github.com/ccheers/xpkg/xmsgbus/impl/memory"
)

func TestExample(t *testing.T) {
	msgbus := memory.NewMsgBus()
	storage := memory.NewStorage()
	manager := xmsgbus.NewTopicManager(context.TODO(), msgbus, newSimpleCas(), storage)
	publisher := xmsgbus.NewPublisher[*dummyEvent](
		msgbus,
		manager,
		xmsgbus.NewOTELOptions(),
	)
	subscriber := xmsgbus.NewSubscriber[*dummyEvent](
		"test",
		"channel",
		msgbus,
		xmsgbus.NewOTELOptions(),
		manager,
		xmsgbus.WithHandleFunc[*dummyEvent](func(ctx context.Context, dst *dummyEvent) error {
			xlogger.DefaultLogger.Log(xlogger.LevelInfo,
				"message",
				"example test ...",
				"dst", dst,
			)
			return nil
		}),
	)
	go func() {
		i := 0
		for {
			i++
			_ = publisher.Publish(context.TODO(), &dummyEvent{Value: uint32(i)})
			time.Sleep(time.Second)
		}
	}()

	go func() {
		for {
			subscriber.Handle(context.TODO())
		}
	}()

	time.Sleep(time.Second * 5)
}
