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
	opts          msgBusOptions
	mu            sync.Mutex
	topicSet      map[string]map[string]chan []byte
	subscriptions map[string]map[string]map[string]time.Time
}

func NewMsgBus(options ...IMsgBusOption) *MsgBus {
	opts := defaultMsgBusOptions()
	for _, opt := range options {
		opt.apply(&opts)
	}
	return &MsgBus{
		opts:          opts,
		mu:            sync.Mutex{},
		topicSet:      make(map[string]map[string]chan []byte),
		subscriptions: make(map[string]map[string]map[string]time.Time),
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
		case bs, ok := <-ch:
			if !ok {
				return nil, nil, xmsgbus.ErrNoData
			}
			return bs, func() {}, nil
		}
	}
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case bs, ok := <-ch:
		if !ok {
			return nil, nil, xmsgbus.ErrNoData
		}
		return bs, func() {}, nil
	}
}

func (x *MsgBus) AddChannel(ctx context.Context, topic string, channel string) error {
	if topic == "" || channel == "" {
		return fmt.Errorf("topic 和 channel 不能为空")
	}
	x.mu.Lock()
	defer x.mu.Unlock()

	x.addChannel(topic, channel)
	return nil
}

func (x *MsgBus) addChannel(topic, channel string) {
	if x.topicSet[topic] == nil {
		x.topicSet[topic] = make(map[string]chan []byte)
	}
	if x.topicSet[topic][channel] == nil {
		x.topicSet[topic][channel] = make(chan []byte, x.opts.maxBuffer)
	}
}

func (x *MsgBus) RegisterSubscription(ctx context.Context, topic, channel, subscriber string, ttl time.Duration) error {
	if topic == "" || channel == "" || subscriber == "" || ttl <= 0 {
		return fmt.Errorf("topic、channel、订阅身份不能为空，TTL 必须为正")
	}
	x.mu.Lock()
	defer x.mu.Unlock()

	x.addChannel(topic, channel)
	if x.subscriptions[topic] == nil {
		x.subscriptions[topic] = make(map[string]map[string]time.Time)
	}
	if x.subscriptions[topic][channel] == nil {
		x.subscriptions[topic][channel] = make(map[string]time.Time)
	}
	x.subscriptions[topic][channel][subscriber] = time.Now().Add(ttl)
	return nil
}

func (x *MsgBus) RemoveChannel(ctx context.Context, topic string, channel string) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	x.removeChannel(topic, channel)
	return nil
}

func (x *MsgBus) removeChannel(topic, channel string) {
	ch := x.topicSet[topic][channel]
	if ch != nil {
		x.drainChain(ch)
	}
	delete(x.topicSet[topic], channel)
	if len(x.topicSet[topic]) == 0 {
		delete(x.topicSet, topic)
	}
	delete(x.subscriptions[topic], channel)
	if len(x.subscriptions[topic]) == 0 {
		delete(x.subscriptions, topic)
	}
}

func (x *MsgBus) UnregisterSubscription(ctx context.Context, topic, channel, subscriber string) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	delete(x.subscriptions[topic][channel], subscriber)
	x.sweepChannel(topic, channel, time.Now())
	return nil
}

func (x *MsgBus) sweepChannel(topic, channel string, now time.Time) {
	subscribers := x.subscriptions[topic][channel]
	for subscriber, expires := range subscribers {
		if !expires.After(now) {
			delete(subscribers, subscriber)
		}
	}
	if len(subscribers) == 0 {
		x.removeChannel(topic, channel)
	}
}

func (x *MsgBus) SweepSubscriptions(ctx context.Context, topic string) (bool, error) {
	x.mu.Lock()
	defer x.mu.Unlock()

	now := time.Now()
	for channel := range x.topicSet[topic] {
		x.sweepChannel(topic, channel, now)
	}
	return len(x.topicSet[topic]) > 0, nil
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
