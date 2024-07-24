package xmsgbus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ccheers/xpkg/sync/try_lock"
	"github.com/ccheers/xpkg/xlogger"
)

// ITopicManager 用于监听 topic channel 当 channel
type ITopicManager interface {
	// Register 注册
	Register(ctx context.Context, topic string, channel string, uuid string, ttl time.Duration) error
	// Unregister 取消注册
	Unregister(ctx context.Context, topic string, channel string, uuid string)
}

type TopicManager struct {
	msgBus       IMsgBus
	storage      ISharedStorage
	mu           sync.Mutex
	manageTopics map[string]struct{}
	cas          try_lock.CASCommand
}

func NewTopicManager(ctx context.Context, client IMsgBus, cas try_lock.CASCommand, storage ISharedStorage) ITopicManager {
	manager := &TopicManager{
		msgBus:       client,
		storage:      storage,
		manageTopics: make(map[string]struct{}),
		cas:          cas,
	}
	go func() {
		manager.monitor(ctx)
	}()
	return manager
}

func (x *TopicManager) Register(ctx context.Context, topic string, channel string, uuid string, ttl time.Duration) error {
	cancel, err := x.lockTopic(ctx, topic, time.Second*3)
	if err != nil {
		return err
	}
	defer cancel()
	err = x.msgBus.AddChannel(ctx, topic, channel)
	if err != nil {
		return fmt.Errorf("[TopicManager][Register][SAdd] err=%w", err)
	}
	err = x.storage.SetEx(ctx, x.subKey(topic, channel, uuid), 1, ttl)
	if err != nil {
		return fmt.Errorf("[TopicManager][Register][SetEX] err=%w", err)
	}
	x.addTopic(ctx, topic)
	return nil
}

func (x *TopicManager) Unregister(ctx context.Context, topic string, channel string, uuid string) {
	_ = x.storage.Del(ctx, x.subKey(topic, channel, uuid))
}

func (x *TopicManager) subKey(topic string, channel string, uuid string) string {
	return x.subKeyPrefix(topic, channel) + uuid
}

func (x *TopicManager) subKeyPrefix(topic string, channel string) string {
	return "hmsgbus:sub:" + topic + ":" + channel + ":"
}

func (x *TopicManager) addTopic(ctx context.Context, topic string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.manageTopics[topic] = struct{}{}
}

func (x *TopicManager) delTopic(ctx context.Context, topic string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	delete(x.manageTopics, topic)
}

// monitor 监听线程
// 如果 topic 下没有 channel 则删除 topic
// 如果 topic 下某个 channel 失活，则删除 channel 的订阅
func (x *TopicManager) monitor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
		}
		func() {
			defer func() {
				err := recover()
				if err != nil {
					_ = xlogger.DefaultLogger.Log(xlogger.LevelError,
						"err", err,
						"module", "[TopicManager][monitor]")
				}
			}()

			x.mu.Lock()
			clones := make([]string, 0, len(x.manageTopics))
			for topic := range x.manageTopics {
				clones = append(clones, topic)
			}
			x.mu.Unlock()

			for _, topic := range clones {
				x.check(ctx, topic)
			}
		}()
	}
}

func (x *TopicManager) check(ctx context.Context, topic string) {
	cancel, err := x.lockTopic(ctx, topic, time.Second)
	if err != nil {
		return
	}
	defer cancel()
	channels, err := x.msgBus.ListChannel(ctx, topic)
	if err != nil {
		_ = xlogger.DefaultLogger.Log(xlogger.LevelError,
			"err", err,
			"topic", topic,
			"module", "[TopicManager][monitor][SMembers]")
	}

	if len(channels) == 0 && err == nil {
		x.delTopic(ctx, topic)
		return
	}

	for _, channel := range channels {
		results, err := x.storage.Keys(ctx, x.subKeyPrefix(topic, channel))
		if err != nil {
			_ = xlogger.DefaultLogger.Log(xlogger.LevelError,
				"err", err,
				"module", "[TopicManager][check][storage.Keys]")
		}
		if len(results) == 0 {
			_ = x.msgBus.RemoveChannel(ctx, topic, channel)
		}
	}
}

func (x *TopicManager) lockTopic(ctx context.Context, topic string, timeout time.Duration) (func(), error) {
	return try_lock.SimpleDistributedTryLock(x.cas, fmt.Sprintf("hmsgbus:topic_manager:%s", topic), timeout)
}
