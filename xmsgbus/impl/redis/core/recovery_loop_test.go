package core

import (
	"context"
	"errors"
	"testing"
)

type unavailableRecoveryClient struct {
	IRedisClient
	attempts int
	cancel   context.CancelFunc
}

func (c *unavailableRecoveryClient) Eval(_ context.Context, script string, _ []string, _ ...interface{}) (interface{}, error) {
	if script == pendingCandidatesScript {
		return []interface{}{"receipt"}, nil
	}
	if script == recoverScript {
		c.attempts++
		if c.attempts == 3 {
			c.cancel()
		}
	}
	return nil, errors.New("恢复和退避写入不可用")
}

func TestRecoveryStopsWhenBackoffCannotBeRecorded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &unavailableRecoveryClient{cancel: cancel}
	bus := &MsgBus{client: client}
	bus.drainRecovery(ctx)
	if client.attempts != 1 {
		t.Fatalf("无法推进时应退出本轮，不能紧密重试：%d", client.attempts)
	}
}
