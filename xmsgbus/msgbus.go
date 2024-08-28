package xmsgbus

import (
	"context"
	"fmt"
	"time"
)

var (
	ErrNoData      = fmt.Errorf("no data")
	ErrCheckFailed = fmt.Errorf("check failed")
	ErrPopTimeout  = fmt.Errorf("pop timeout")
)

type ITopic interface {
	Topic() string
}

type IMsgBus interface {
	// Push 推入数据
	Push(ctx context.Context, topic string, bs []byte) error
	// Pop 以阻塞的方式获取数据
	// blockTimeout 为 0 则永久阻塞 直到 context 退出 或 数据到达
	Pop(ctx context.Context, topic, channel string, blockTimeout time.Duration) (data []byte, ackFn func(), err error)
	// AddChannel 为 topic 添加 channel
	AddChannel(ctx context.Context, topic string, channel string) error
	// RemoveChannel 删除 Channel, channel 下的数据也应该被删除
	RemoveChannel(ctx context.Context, topic string, channel string) error
	// ListChannel 列出 Topic 下所有 Channel
	ListChannel(ctx context.Context, topic string) ([]string, error)
}

type ISharedStorage interface {
	// SetEx 设置一个 值 ，并且设置它的过期时间
	SetEx(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	// Keys 通过 前缀匹配 列出满足条件的 所有 Key
	Keys(ctx context.Context, prefix string) ([]string, error)
	// Del 删除 key
	Del(ctx context.Context, key string) error
}
