package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ccheers/xpkg/xmsgbus"
)

var ErrChanIsFull = fmt.Errorf("channel is full")

type msgBusOptions struct {
	maxBuffer int
}

func defaultMsgBusOptions() msgBusOptions {
	return msgBusOptions{maxBuffer: 32}
}

type IMsgBusOption interface {
	apply(*msgBusOptions)
}

type MsgBusOptionFunc func(*msgBusOptions)

func (fn MsgBusOptionFunc) apply(options *msgBusOptions) {
	fn(options)
}

func WithMsgBusMaxBufferOption(maxBuffer int) MsgBusOptionFunc {
	return func(options *msgBusOptions) {
		options.maxBuffer = maxBuffer
	}
}

type MsgBus struct {
	opts     msgBusOptions
	mu       sync.Mutex
	topicSet map[string]map[string]chan []byte
}

func NewMsgBus(options ...IMsgBusOption) xmsgbus.IMsgBus {
	opts := defaultMsgBusOptions()
	for _, opt := range options {
		opt.apply(&opts)
	}
	return &MsgBus{
		opts:     opts,
		mu:       sync.Mutex{},
		topicSet: make(map[string]map[string]chan []byte),
	}
}

func (x *MsgBus) Push(ctx context.Context, topic string, bs []byte) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	var fullChans []string
	for chanName, ch := range x.topicSet[topic] {
		select {
		case ch <- bs:
			// 满时丢弃
		default:
			fullChans = append(fullChans, chanName)
		}
	}
	if len(fullChans) > 0 {
		return fmt.Errorf("%w: channels=%+v", ErrChanIsFull, fullChans)
	}
	return nil
}

func (x *MsgBus) Pop(ctx context.Context, topic, channel string, blockTimeout time.Duration) ([]byte, func(), error) {
	x.mu.Lock()
	if x.topicSet[topic] == nil {
		x.topicSet[topic] = make(map[string]chan []byte)
	}
	if x.topicSet[topic][channel] == nil {
		x.topicSet[topic][channel] = make(chan []byte, x.opts.maxBuffer)
	}
	ch := x.topicSet[topic][channel]
	x.mu.Unlock()
	if blockTimeout > 0 {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(blockTimeout):
			return nil, nil, xmsgbus.ErrPopTimeout
		case bs := <-ch:
			return bs, func() {}, nil
		}
	}
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case bs := <-ch:
		return bs, func() {}, nil
	}
}

func (x *MsgBus) AddChannel(ctx context.Context, topic string, channel string) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	if x.topicSet[topic] == nil {
		x.topicSet[topic] = make(map[string]chan []byte)
	}
	if x.topicSet[topic][channel] == nil {
		x.topicSet[topic][channel] = make(chan []byte, x.opts.maxBuffer)
	}
	return nil
}

func (x *MsgBus) RemoveChannel(ctx context.Context, topic string, channel string) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	if x.topicSet[topic] == nil {
		return nil
	}
	ch := x.topicSet[topic][channel]
	if ch == nil {
		return nil
	}
	x.drainChain(ch)
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

func (x *MsgBus) drainChain(ch chan []byte) {
	close(ch)
	for range ch {
	}
}
