package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ccheers/xpkg/xmsgbus"
)

type MsgBus struct {
	mu       sync.Mutex
	topicSet map[string]map[string]chan []byte
}

func NewMsgBus() xmsgbus.IMsgBus {
	return &MsgBus{
		mu:       sync.Mutex{},
		topicSet: make(map[string]map[string]chan []byte),
	}
}

func (x *MsgBus) Push(ctx context.Context, topic string, bs []byte) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	for _, ch := range x.topicSet[topic] {
		select {
		case ch <- bs:
			// 满时丢弃
		default:
			return fmt.Errorf("channel is full")
		}
	}
	return nil
}

func (x *MsgBus) Pop(ctx context.Context, topic, channel string, blockTimeout time.Duration) ([]byte, error) {
	x.mu.Lock()
	if x.topicSet[topic] == nil {
		x.topicSet[topic] = make(map[string]chan []byte)
	}
	if x.topicSet[topic][channel] == nil {
		x.topicSet[topic][channel] = make(chan []byte, 32)
	}
	ch := x.topicSet[topic][channel]
	x.mu.Unlock()
	if blockTimeout > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(blockTimeout):
			return nil, xmsgbus.ErrPopTimeout
		case bs := <-ch:
			return bs, nil
		}
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case bs := <-ch:
		return bs, nil
	}
}

func (x *MsgBus) AddChannel(ctx context.Context, topic string, channel string) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	if x.topicSet[topic] == nil {
		x.topicSet[topic] = make(map[string]chan []byte)
	}
	if x.topicSet[topic][channel] == nil {
		x.topicSet[topic][channel] = make(chan []byte, 32)
	}
	return nil
}

func (x *MsgBus) RemoveChannel(ctx context.Context, topic string, channel string) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	if x.topicSet[topic] == nil {
		return nil
	}
	if x.topicSet[topic][channel] == nil {
		return nil
	}
	delete(x.topicSet[topic], channel)
	return nil
}

func (x *MsgBus) ListChannel(ctx context.Context, topic string) ([]string, error) {
	x.mu.Lock()
	defer x.mu.Unlock()

	if x.topicSet[topic] == nil {
		return nil, nil
	}
	channels := make([]string, 0, len(x.topicSet[topic]))
	for c := range x.topicSet[topic] {
		channels = append(channels, c)
	}
	return channels, nil
}
