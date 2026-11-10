package xmsgbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/ccheers/xpkg/xmsgbus"
	"github.com/ccheers/xpkg/xmsgbus/impl/memory"
)

type timeoutBus struct{ xmsgbus.IMsgBus }

func (timeoutBus) Pop(context.Context, string, string, time.Duration) ([]byte, func(), error) {
	return nil, nil, xmsgbus.ErrPopTimeout
}

func TestSubscriberEmptyPollIsNotFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager := xmsgbus.NewTopicManager(ctx, memory.NewMsgBus())
	subscriber := xmsgbus.NewSubscriber[*dummyEvent]("t", "c", timeoutBus{},
		xmsgbus.NewOTELOptions(), manager)
	if err := subscriber.Handle(ctx); err != nil {
		t.Fatalf("空轮询是正常结果：%v", err)
	}
}
