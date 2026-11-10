package core_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ccheers/xpkg/xmsgbus/impl/redis/core"
	v9 "github.com/ccheers/xpkg/xmsgbus/impl/redis/v9"
	"github.com/redis/go-redis/v9"
)

func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	bin, err := exec.LookPath("redis-server")
	if err != nil {
		t.Skip("原子性集成测试需要本机 redis-server")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	cmd := exec.Command(bin, "--bind", "127.0.0.1", "--port", strconv.Itoa(port),
		"--save", "", "--appendonly", "no", "--dir", t.TempDir())
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("127.0.0.1:%d", port), MaxRetries: -1})
	t.Cleanup(func() {
		client.Close()
		cmd.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() { cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			cmd.Process.Kill()
			<-done
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for client.Ping(ctx).Err() != nil {
		if ctx.Err() != nil {
			t.Fatal("隔离 Redis 未能启动")
		}
		time.Sleep(10 * time.Millisecond)
	}
	return client
}

var errInjected = errors.New("注入存储写入失败")

type failingClient struct {
	core.IRedisClient
	raw  *redis.Client
	fail atomic.Bool
}

func (c *failingClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) (string, error) {
	if c.fail.Swap(false) {
		return "", errInjected
	}
	return c.IRedisClient.Set(ctx, key, value, ttl)
}

func (c *failingClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	if c.fail.Swap(false) {
		return nil, errInjected
	}
	return c.raw.Eval(ctx, script, keys, args...).Result()
}

func TestPopStorageFailureDoesNotConsumeMessage(t *testing.T) {
	ctx := context.Background()
	raw := testRedis(t)
	client := &failingClient{IRedisClient: v9.NewRedisClientImplV9(raw), raw: raw}
	bus := core.NewMsgBus(client)
	if closer, ok := interface{}(bus).(interface{ Close() error }); ok {
		t.Cleanup(func() { closer.Close() })
	}
	if err := bus.AddChannel(ctx, "topic", "channel"); err != nil {
		t.Fatal(err)
	}
	if err := bus.Push(ctx, "topic", []byte("不能丢失")); err != nil {
		t.Fatal(err)
	}
	client.fail.Store(true)
	if _, _, err := bus.Pop(ctx, "topic", "channel", time.Second); !errors.Is(err, errInjected) {
		t.Fatalf("登记失败必须返回错误，而不是交付无恢复记录的消息：%v", err)
	}
	data, ack, err := bus.Pop(ctx, "topic", "channel", time.Second)
	if err != nil || string(data) != "不能丢失" {
		t.Fatalf("失败后消息应仍可领取：data=%q err=%v", data, err)
	}
	ack()
}
