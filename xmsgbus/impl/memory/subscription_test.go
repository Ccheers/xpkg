package memory_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ccheers/xpkg/xmsgbus"
	"github.com/ccheers/xpkg/xmsgbus/impl/memory"
)

type popContext struct {
	context.Context
	ready chan struct{}
	once  sync.Once
}

func (c *popContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.ready) })
	return c.Context.Done()
}

func TestPopClosedChannelReturnsNoData(t *testing.T) {
	for _, timeout := range []time.Duration{0, time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			popCtx := &popContext{Context: ctx, ready: make(chan struct{})}
			bus := memory.NewMsgBus()
			result := make(chan error, 1)
			go func() {
				_, _, err := bus.Pop(popCtx, "topic", "channel", timeout)
				result <- err
			}()
			select {
			case <-popCtx.ready:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if err := bus.RemoveChannel(ctx, "topic", "channel"); err != nil {
				t.Fatal(err)
			}
			if err := <-result; !errors.Is(err, xmsgbus.ErrNoData) {
				t.Fatalf("Pop() error = %v, want ErrNoData", err)
			}
		})
	}
}

func TestSubscriptionRegistrationCreatesChannel(t *testing.T) {
	ctx := context.Background()
	bus := memory.NewMsgBus()
	if err := bus.RegisterSubscription(ctx, "topic", "channel", "subscriber", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := bus.Push(ctx, "topic", []byte("message")); err != nil {
		t.Fatal(err)
	}
	data, ack, err := bus.Pop(ctx, "topic", "channel", time.Second)
	if err != nil || string(data) != "message" || ack == nil {
		t.Fatalf("Pop() = %q, ack present=%v, %v", data, ack != nil, err)
	}
}

func TestUnregisterPreservesOtherSubscriberQueue(t *testing.T) {
	ctx := context.Background()
	bus := memory.NewMsgBus()
	for _, subscriber := range []string{"old", "new"} {
		if err := bus.RegisterSubscription(ctx, "topic", "channel", subscriber, time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	if err := bus.Push(ctx, "topic", []byte("pending")); err != nil {
		t.Fatal(err)
	}
	if err := bus.UnregisterSubscription(ctx, "topic", "channel", "old"); err != nil {
		t.Fatal(err)
	}
	data, _, err := bus.Pop(ctx, "topic", "channel", time.Second)
	if err != nil || string(data) != "pending" {
		t.Fatalf("remaining subscriber lost queue: %q, %v", data, err)
	}
	if err := bus.UnregisterSubscription(ctx, "topic", "channel", "new"); err != nil {
		t.Fatal(err)
	}
	channels, err := bus.ListChannel(ctx, "topic")
	if err != nil || len(channels) != 0 {
		t.Fatalf("last subscriber left channel: %v, %v", channels, err)
	}
}

func TestSweepSubscriptionsExpiresAndRenews(t *testing.T) {
	ctx := context.Background()
	bus := memory.NewMsgBus()
	for _, channel := range []string{"expired", "renewed"} {
		if err := bus.RegisterSubscription(ctx, "topic", channel, "subscriber", time.Nanosecond); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(time.Millisecond)
	if err := bus.RegisterSubscription(ctx, "topic", "renewed", "subscriber", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := bus.Push(ctx, "topic", []byte("pending")); err != nil {
		t.Fatal(err)
	}
	if alive, err := bus.SweepSubscriptions(ctx, "topic"); err != nil || !alive {
		t.Fatalf("SweepSubscriptions() = %v, %v", alive, err)
	}
	channels, err := bus.ListChannel(ctx, "topic")
	if err != nil || len(channels) != 1 || channels[0] != "renewed" {
		t.Fatalf("channels = %v, %v", channels, err)
	}
	data, _, err := bus.Pop(ctx, "topic", "renewed", time.Second)
	if err != nil || string(data) != "pending" {
		t.Fatalf("renewal lost queue: %q, %v", data, err)
	}
	if err := bus.UnregisterSubscription(ctx, "topic", "renewed", "subscriber"); err != nil {
		t.Fatal(err)
	}
	if alive, err := bus.SweepSubscriptions(ctx, "topic"); err != nil || alive {
		t.Fatalf("empty topic sweep = %v, %v", alive, err)
	}
	if err := bus.RegisterSubscription(ctx, "topic", "expired", "fresh", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, _, err := bus.Pop(ctx, "topic", "expired", time.Millisecond); !errors.Is(err, xmsgbus.ErrPopTimeout) {
		t.Fatalf("expired queue was not discarded: %v", err)
	}
}

func TestSubscriptionRejectsInvalidTTLWithoutCreatingChannel(t *testing.T) {
	for _, ttl := range []time.Duration{0, -time.Second} {
		t.Run(ttl.String(), func(t *testing.T) {
			ctx := context.Background()
			bus := memory.NewMsgBus()
			if err := bus.RegisterSubscription(ctx, "topic", "channel", "subscriber", ttl); err == nil {
				t.Fatal("invalid TTL was accepted")
			}
			channels, err := bus.ListChannel(ctx, "topic")
			if err != nil || len(channels) != 0 {
				t.Fatalf("failed registration created channel: %v, %v", channels, err)
			}
		})
	}
}

func TestSubscriptionKeepsDefaultBufferAndFullQueueBehavior(t *testing.T) {
	ctx := context.Background()
	bus := memory.NewMsgBus()
	if err := bus.RegisterSubscription(ctx, "topic", "channel", "subscriber", time.Minute); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 32; i++ {
		if err := bus.Push(ctx, "topic", []byte{byte(i)}); err != nil {
			t.Fatalf("Push(%d) = %v", i, err)
		}
	}
	if err := bus.Push(ctx, "topic", []byte("dropped")); !errors.Is(err, memory.ErrChanIsFull) {
		t.Fatalf("full queue Push() = %v", err)
	}
	for i := 0; i < 32; i++ {
		data, _, err := bus.Pop(ctx, "topic", "channel", time.Second)
		if err != nil || len(data) != 1 || data[0] != byte(i) {
			t.Fatalf("Pop(%d) = %v, %v", i, data, err)
		}
	}
	if _, _, err := bus.Pop(ctx, "topic", "channel", time.Millisecond); !errors.Is(err, xmsgbus.ErrPopTimeout) {
		t.Fatalf("full queue retained dropped message: %v", err)
	}
}

func TestConcurrentSubscriptionCleanupPreservesNewSubscriber(t *testing.T) {
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		bus := memory.NewMsgBus()
		if err := bus.RegisterSubscription(ctx, "topic", "channel", "old", time.Nanosecond); err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		result := make(chan error, 3)
		go func() {
			<-start
			result <- bus.RegisterSubscription(ctx, "topic", "channel", "new", time.Minute)
		}()
		go func() {
			<-start
			result <- bus.UnregisterSubscription(ctx, "topic", "channel", "old")
		}()
		go func() {
			<-start
			_, err := bus.SweepSubscriptions(ctx, "topic")
			result <- err
		}()
		close(start)
		for j := 0; j < 3; j++ {
			if err := <-result; err != nil {
				t.Fatal(err)
			}
		}
		if err := bus.Push(ctx, "topic", []byte("message")); err != nil {
			t.Fatal(err)
		}
		data, _, err := bus.Pop(ctx, "topic", "channel", time.Second)
		if err != nil || string(data) != "message" {
			t.Fatalf("iteration %d lost new subscriber: %q, %v", i, data, err)
		}
	}
}
