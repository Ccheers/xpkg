package xmsgbus_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ccheers/xpkg/xmsgbus"
	"github.com/ccheers/xpkg/xmsgbus/impl/memory"
)

type controlledRegistry struct {
	*memory.MsgBus
	sweep func(context.Context, string) (bool, error)
}

func (r *controlledRegistry) SweepSubscriptions(ctx context.Context, topic string) (bool, error) {
	return r.sweep(ctx, topic)
}

func TestTopicManagerSweepExpiresSubscriptions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus := memory.NewMsgBus()
	manager := xmsgbus.NewTopicManager(ctx, bus)
	if err := manager.Register(ctx, "topic", "channel", "subscriber", time.Nanosecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	if err := manager.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	channels, err := bus.ListChannel(ctx, "topic")
	if err != nil || len(channels) != 0 {
		t.Fatalf("expired subscription retained channel: %v, %v", channels, err)
	}
}

func TestTopicManagerRetriesFailedSweep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus := memory.NewMsgBus()
	failure := errors.New("registry unavailable")
	failed := false
	registry := &controlledRegistry{MsgBus: bus}
	registry.sweep = func(ctx context.Context, topic string) (bool, error) {
		if !failed {
			failed = true
			return false, failure
		}
		return bus.SweepSubscriptions(ctx, topic)
	}
	manager := xmsgbus.NewTopicManager(ctx, registry)
	if err := manager.Register(ctx, "topic", "channel", "subscriber", time.Nanosecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	if err := manager.Sweep(ctx); !errors.Is(err, failure) {
		t.Fatalf("Sweep() error = %v, want backend failure", err)
	}
	channels, err := bus.ListChannel(ctx, "topic")
	if err != nil || len(channels) != 1 {
		t.Fatalf("failed sweep deleted channel: %v, %v", channels, err)
	}
	if err := manager.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	channels, err = bus.ListChannel(ctx, "topic")
	if err != nil || len(channels) != 0 {
		t.Fatalf("failed topic was not retried: %v, %v", channels, err)
	}
}

func TestTopicManagerConcurrentRegistrationRemainsManaged(t *testing.T) {
	for _, recreate := range []bool{false, true} {
		name := "register during sweep"
		if recreate {
			name = "delete and recreate during sweep"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			bus := memory.NewMsgBus()
			started := make(chan struct{})
			release := make(chan struct{})
			var sweeps atomic.Int32
			registry := &controlledRegistry{MsgBus: bus}
			registry.sweep = func(ctx context.Context, topic string) (bool, error) {
				alive, err := bus.SweepSubscriptions(ctx, topic)
				if sweeps.Add(1) == 1 {
					close(started)
					select {
					case <-release:
					case <-ctx.Done():
						return false, ctx.Err()
					}
				}
				return alive, err
			}
			manager := xmsgbus.NewTopicManager(ctx, registry)
			if err := manager.Register(ctx, "topic", "channel", "old", time.Nanosecond); err != nil {
				t.Fatal(err)
			}
			time.Sleep(time.Millisecond)
			result := make(chan error, 1)
			go func() { result <- manager.Sweep(ctx) }()
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if recreate {
				if err := manager.Sweep(ctx); err != nil {
					t.Fatal(err)
				}
			}
			if err := manager.Register(ctx, "topic", "channel", "new", time.Nanosecond); err != nil {
				t.Fatal(err)
			}
			close(release)
			if err := <-result; err != nil {
				t.Fatal(err)
			}
			time.Sleep(time.Millisecond)
			if err := manager.Sweep(ctx); err != nil {
				t.Fatal(err)
			}
			channels, err := bus.ListChannel(ctx, "topic")
			if err != nil || len(channels) != 0 {
				t.Fatalf("concurrent registration was forgotten: %v, %v", channels, err)
			}
		})
	}
}
