package xmsgbus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

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
	registry     SubscriptionRegistry
	mu           sync.Mutex
	manageTopics map[string]uint64
	revision     uint64
}

func NewTopicManager(ctx context.Context, registry SubscriptionRegistry) *TopicManager {
	manager := &TopicManager{
		registry:     registry,
		manageTopics: make(map[string]uint64),
	}
	go func() {
		manager.monitor(ctx)
	}()
	return manager
}

func (x *TopicManager) Register(ctx context.Context, topic string, channel string, uuid string, ttl time.Duration) error {
	if err := x.registry.RegisterSubscription(ctx, topic, channel, uuid, ttl); err != nil {
		return fmt.Errorf("[TopicManager][Register] err=%w", err)
	}
	x.mu.Lock()
	x.revision++
	x.manageTopics[topic] = x.revision
	x.mu.Unlock()
	return nil
}

func (x *TopicManager) Unregister(ctx context.Context, topic string, channel string, uuid string) {
	if err := x.registry.UnregisterSubscription(ctx, topic, channel, uuid); err != nil {
		_ = xlogger.DefaultLogger.Log(xlogger.LevelError,
			"err", err,
			"topic", topic,
			"channel", channel,
			"module", "[TopicManager][Unregister]")
	}
}

// monitor 监听线程
// 如果 topic 下没有 channel 则删除 topic
// 如果 topic 下某个 channel 失活，则删除 channel 的订阅
func (x *TopicManager) monitor(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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

			if err := x.Sweep(ctx); err != nil {
				_ = xlogger.DefaultLogger.Log(xlogger.LevelError,
					"err", err,
					"module", "[TopicManager][monitor]")
			}
		}()
	}
}

// Sweep 对本进程注册过的 topic 执行一次订阅清理，失败的 topic 留待下次重试。
func (x *TopicManager) Sweep(ctx context.Context) error {
	x.mu.Lock()
	topics := make(map[string]uint64, len(x.manageTopics))
	for topic, revision := range x.manageTopics {
		topics[topic] = revision
	}
	x.mu.Unlock()

	var errs []error
	for topic, revision := range topics {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(errs, err)...)
		}
		alive, err := x.registry.SweepSubscriptions(ctx, topic)
		if err != nil {
			errs = append(errs, fmt.Errorf("[TopicManager][Sweep] topic=%q: %w", topic, err))
			continue
		}
		if !alive {
			x.mu.Lock()
			// 清理期间的新注册必须继续巡检；全局递增版本也避免删除后重建的 ABA。
			if x.manageTopics[topic] == revision {
				delete(x.manageTopics, topic)
			}
			x.mu.Unlock()
		}
	}
	return errors.Join(errs...)
}
