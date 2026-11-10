package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ccheers/xpkg/xmsgbus"
	"github.com/google/uuid"
)

type MsgBus struct {
	client IRedisClient
	cancel context.CancelFunc
	done   chan struct{}
}

func NewMsgBus(client IRedisClient) *MsgBus {
	ctx, cancel := context.WithCancel(context.Background())
	x := &MsgBus{client: client, cancel: cancel, done: make(chan struct{})}
	go x.recoverLoop(ctx)
	return x
}

func (x *MsgBus) Close() error {
	x.cancel()
	<-x.done
	return nil
}

func (x *MsgBus) recoverLoop(ctx context.Context) {
	defer close(x.done)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			budget, cancel := context.WithTimeout(ctx, 10*time.Second)
			x.drainRecovery(budget)
			cancel()
		}
	}
}

func (x *MsgBus) drainRecovery(ctx context.Context) {
	for ctx.Err() == nil {
		if _, err := x.Recover(ctx, 100); err != nil {
			slog.Error("消息回执恢复失败，保留待恢复记录", "err", err)
			if errors.Is(err, errRecoveryUnavailable) {
				return
			}
		}
		remaining, err := x.pendingCandidates(ctx, 1)
		if err != nil || len(remaining) == 0 {
			return
		}
	}
}

type channelEntry struct {
	name       string
	generation string
}

func (x *MsgBus) channels(ctx context.Context, topic string) ([]channelEntry, error) {
	raw, err := x.client.Eval(ctx, channelsScript, []string{catalogKey(topic)})
	if err != nil {
		return nil, err
	}
	values, ok := raw.([]interface{})
	if !ok || len(values)%2 != 0 {
		return nil, fmt.Errorf("无效 channel 列表响应: %T", raw)
	}
	entries := make([]channelEntry, 0, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		name, nameOK := values[i].(string)
		generation, generationOK := values[i+1].(string)
		if !nameOK || !generationOK {
			return nil, errors.New("无效 channel 身份")
		}
		entries = append(entries, channelEntry{name, generation})
	}
	return entries, nil
}

func (x *MsgBus) Push(ctx context.Context, topic string, bs []byte) error {
	channels, err := x.channels(ctx, topic)
	if err != nil {
		return err
	}
	var errs []error
	for _, channel := range channels {
		keys := channelKeysFor(topic, channel.name)
		_, err := x.client.Eval(ctx, pushScript, []string{keys.catalog, keys.ready},
			channel.name, channel.generation, bs)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (x *MsgBus) Pop(ctx context.Context, topic, channel string, blockTimeout time.Duration) ([]byte, func(), error) {
	if blockTimeout < 0 {
		return nil, nil, errors.New("等待时间不能为负")
	}
	if topic == "" || channel == "" {
		return nil, nil, errors.New("topic 和 channel 不能为空")
	}
	var deadline time.Time
	if blockTimeout > 0 {
		deadline = time.Now().Add(blockTimeout)
	}
	receipt := uuid.NewString()
	rawGeneration, err := x.client.Eval(ctx, generationScript, []string{catalogKey(topic)}, channel)
	if err != nil {
		return nil, nil, err
	}
	generation, err := ackResultString(rawGeneration)
	if err != nil {
		return nil, nil, err
	}
	if generation == "" {
		return nil, nil, xmsgbus.ErrNoData
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		data, found, err := x.claim(ctx, topic, channel, receipt, generation)
		if err != nil {
			return nil, nil, err
		}
		if found {
			var mu sync.Mutex
			acked := false
			return data, func() {
				mu.Lock()
				defer mu.Unlock()
				if acked {
					return
				}
				cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer cancel()
				if err := x.acknowledge(cleanup, receipt); err != nil {
					slog.Error("确认消息失败，保留待恢复记录", "err", err)
					return
				}
				acked = true
			}, nil
		}
		wait := 100 * time.Millisecond
		if !deadline.IsZero() {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return nil, nil, xmsgbus.ErrPopTimeout
			}
			wait = min(wait, remaining)
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (x *MsgBus) AddChannel(ctx context.Context, topic, channel string) error {
	if topic == "" || channel == "" {
		return errors.New("topic 和 channel 不能为空")
	}
	keys := channelKeysFor(topic, channel)
	_, err := x.client.Eval(ctx, addChannelScript, []string{keys.catalog, keys.leases, keys.ready}, channel, uuid.NewString())
	return err
}

func (x *MsgBus) RemoveChannel(ctx context.Context, topic, channel string) error {
	keys := channelKeysFor(topic, channel)
	_, err := x.client.Eval(ctx, removeChannelScript, []string{keys.catalog, keys.leases, keys.ready}, channel)
	return err
}

func (x *MsgBus) ListChannel(ctx context.Context, topic string) ([]string, error) {
	entries, err := x.channels(ctx, topic)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.name)
	}
	return names, nil
}

func (x *MsgBus) RegisterSubscription(ctx context.Context, topic, channel, subscriber string, ttl time.Duration) error {
	if topic == "" || channel == "" || subscriber == "" || ttl <= 0 {
		return errors.New("topic、channel、订阅身份不能为空，TTL 必须为正")
	}
	ttlMillis := ttl.Milliseconds()
	if ttl%time.Millisecond != 0 {
		ttlMillis++
	}
	keys := channelKeysFor(topic, channel)
	_, err := x.client.Eval(ctx, registerScript, []string{keys.catalog, keys.leases, keys.ready},
		channel, subscriber, uuid.NewString(), ttlMillis)
	return err
}

func (x *MsgBus) UnregisterSubscription(ctx context.Context, topic, channel, subscriber string) error {
	keys := channelKeysFor(topic, channel)
	_, err := x.client.Eval(ctx, unregisterScript, []string{keys.catalog, keys.leases, keys.ready}, channel, subscriber)
	return err
}

func (x *MsgBus) SweepSubscriptions(ctx context.Context, topic string) (bool, error) {
	entries, err := x.channels(ctx, topic)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		keys := channelKeysFor(topic, entry.name)
		if _, err := x.client.Eval(ctx, sweepScript, []string{keys.catalog, keys.leases, keys.ready},
			entry.name, entry.generation); err != nil {
			return false, err
		}
	}
	remaining, err := x.channels(ctx, topic)
	return len(remaining) > 0, err
}
