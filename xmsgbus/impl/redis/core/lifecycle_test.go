package core_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ccheers/xpkg/xmsgbus"
	"github.com/ccheers/xpkg/xmsgbus/impl/redis/core"
	v9 "github.com/ccheers/xpkg/xmsgbus/impl/redis/v9"
	"github.com/redis/go-redis/v9"
)

const dueKey = "hmsgbus:v4:pending:due"

func newBus(t *testing.T) (*redis.Client, *core.MsgBus) {
	t.Helper()
	raw := testRedis(t)
	bus := core.NewMsgBus(v9.NewRedisClientImplV9(raw))
	t.Cleanup(func() { bus.Close() })
	return raw, bus
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func take(t *testing.T, bus *core.MsgBus, topic, channel string) ([]byte, func()) {
	t.Helper()
	data, ack, err := bus.Pop(context.Background(), topic, channel, time.Second)
	must(t, err)
	return data, ack
}

func empty(t *testing.T, bus *core.MsgBus, topic, channel string) {
	t.Helper()
	data, _, err := bus.Pop(context.Background(), topic, channel, 15*time.Millisecond)
	if !errors.Is(err, xmsgbus.ErrPopTimeout) {
		t.Fatalf("应为空队列：data=%q err=%v", data, err)
	}
}

func expireReceipts(t *testing.T, raw *redis.Client) {
	t.Helper()
	ctx := context.Background()
	receipts, err := raw.ZRange(ctx, dueKey, 0, -1).Result()
	must(t, err)
	for _, receipt := range receipts {
		must(t, raw.ZAdd(ctx, dueKey, redis.Z{Member: receipt, Score: 0}).Err())
	}
}

func TestIdenticalPayloadHasIndependentReceipts(t *testing.T) {
	ctx := context.Background()
	raw, bus := newBus(t)
	must(t, bus.AddChannel(ctx, "t", "a"))
	must(t, bus.AddChannel(ctx, "t", "b"))
	must(t, bus.Push(ctx, "t", []byte("same")))
	_, ackA := take(t, bus, "t", "a")
	_, oldAckB := take(t, bus, "t", "b")
	ackA()
	expireReceipts(t, raw)
	count, err := bus.Recover(ctx, 100)
	must(t, err)
	if count != 1 {
		t.Fatalf("仅未确认的 b 应恢复：%d", count)
	}
	data, newAckB := take(t, bus, "t", "b")
	if string(data) != "same" {
		t.Fatalf("恢复正文被改变：%q", data)
	}
	oldAckB()
	expireReceipts(t, raw)
	count, err = bus.Recover(ctx, 100)
	must(t, err)
	if count != 1 {
		t.Fatal("旧回执不得确认新投递")
	}
	newAckB()
	_, finalAck := take(t, bus, "t", "b")
	finalAck()
	empty(t, bus, "t", "a")
	empty(t, bus, "t", "b")
}

func TestRecoverIsBoundedAndConcurrent(t *testing.T) {
	ctx := context.Background()
	raw, bus := newBus(t)
	must(t, bus.AddChannel(ctx, "t", "c"))
	for i := 0; i < 20; i++ {
		must(t, bus.Push(ctx, "t", []byte(fmt.Sprintf("message-%d", i))))
		take(t, bus, "t", "c")
	}
	count, err := bus.Recover(ctx, 100)
	must(t, err)
	if count != 0 {
		t.Fatal("未到期不应恢复")
	}
	expireReceipts(t, raw)
	count, err = bus.Recover(ctx, 3)
	must(t, err)
	if count != 3 {
		t.Fatalf("每批上限失效：%d", count)
	}
	var total atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count, err := bus.Recover(ctx, 100)
			if err != nil {
				t.Error(err)
			}
			total.Add(int64(count))
		}()
	}
	wg.Wait()
	if total.Load() != 17 {
		t.Fatalf("竞争恢复重复或遗漏：%d", total.Load())
	}
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		data, ack := take(t, bus, "t", "c")
		if seen[string(data)] {
			t.Fatalf("重复恢复：%q", data)
		}
		seen[string(data)] = true
		ack()
	}
	empty(t, bus, "t", "c")
}

func TestAcknowledgeAndRecoveryRace(t *testing.T) {
	ctx := context.Background()
	raw, bus := newBus(t)
	must(t, bus.AddChannel(ctx, "t", "c"))
	for i := 0; i < 40; i++ {
		body := []byte(fmt.Sprintf("message-%d", i))
		must(t, bus.Push(ctx, "t", body))
		_, ack := take(t, bus, "t", "c")
		expireReceipts(t, raw)
		var count atomic.Int64
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); ack() }()
		go func() {
			defer wg.Done()
			n, err := bus.Recover(ctx, 100)
			if err != nil {
				t.Error(err)
			}
			count.Store(int64(n))
		}()
		wg.Wait()
		if count.Load() == 1 {
			data, newAck := take(t, bus, "t", "c")
			if string(data) != string(body) {
				t.Fatalf("竞争恢复改变消息：%q", data)
			}
			newAck()
		} else if count.Load() != 0 {
			t.Fatalf("同一回执被重复恢复：%d", count.Load())
		}
		empty(t, bus, "t", "c")
	}
}

func TestAcknowledgeAfterContextCancellation(t *testing.T) {
	raw, bus := newBus(t)
	ctx, cancel := context.WithCancel(context.Background())
	must(t, bus.AddChannel(ctx, "t", "c"))
	must(t, bus.Push(ctx, "t", []byte{}))
	data, ack, err := bus.Pop(ctx, "t", "c", time.Second)
	must(t, err)
	if len(data) != 0 {
		t.Fatal("空正文也必须是有效消息")
	}
	cancel()
	ack()
	expireReceipts(t, raw)
	count, err := bus.Recover(context.Background(), 100)
	must(t, err)
	if count != 0 {
		t.Fatal("已确认的空消息不应恢复")
	}
}

func TestOldReceiptCannotReviveRecreatedChannel(t *testing.T) {
	ctx := context.Background()
	raw, bus := newBus(t)
	must(t, bus.AddChannel(ctx, "t", "c"))
	must(t, bus.Push(ctx, "t", []byte("old")))
	_, ack := take(t, bus, "t", "c")
	must(t, bus.RemoveChannel(ctx, "t", "c"))
	must(t, bus.AddChannel(ctx, "t", "c"))
	expireReceipts(t, raw)
	count, err := bus.Recover(ctx, 100)
	must(t, err)
	if count != 0 {
		t.Fatal("旧回执不得恢复到新一代 channel")
	}
	ack()
	empty(t, bus, "t", "c")
	must(t, bus.Push(ctx, "t", []byte("new")))
	data, ack := take(t, bus, "t", "c")
	if string(data) != "new" {
		t.Fatalf("旧消息串入新生命周期：%q", data)
	}
	ack()
}

func TestSubscriptionRenewalCleanupRace(t *testing.T) {
	ctx := context.Background()
	_, bus := newBus(t)
	for i := 0; i < 40; i++ {
		must(t, bus.RegisterSubscription(ctx, "t", "c", "old", time.Minute))
		must(t, bus.UnregisterSubscription(ctx, "t", "c", "old"))
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			if err := bus.RegisterSubscription(ctx, "t", "c", "new", time.Minute); err != nil {
				t.Error(err)
			}
		}()
		go func() {
			defer wg.Done()
			if _, err := bus.SweepSubscriptions(ctx, "t"); err != nil {
				t.Error(err)
			}
		}()
		wg.Wait()
		must(t, bus.UnregisterSubscription(ctx, "t", "c", "old"))
		alive, err := bus.SweepSubscriptions(ctx, "t")
		must(t, err)
		if !alive {
			t.Fatal("清理或旧实例注销误删新订阅")
		}
		must(t, bus.Push(ctx, "t", []byte("live")))
		data, ack := take(t, bus, "t", "c")
		if string(data) != "live" {
			t.Fatalf("活订阅消息丢失：%q", data)
		}
		ack()
		must(t, bus.RemoveChannel(ctx, "t", "c"))
	}
}

func fixtureChannelKey(topic, channel, suffix string) string {
	encode := base64.RawURLEncoding.EncodeToString
	return "hmsgbus:v4:channel:" + encode([]byte(topic)) + ":" + encode([]byte(channel)) + ":" + suffix
}

func TestWrongTypeDoesNotPartiallyRegisterOrClean(t *testing.T) {
	ctx := context.Background()
	raw, bus := newBus(t)
	leaseKey := fixtureChannelKey("t", "c", "leases")
	must(t, raw.Set(ctx, leaseKey, "wrong-type", 0).Err())
	if err := bus.RegisterSubscription(ctx, "t", "c", "id", time.Minute); err == nil {
		t.Fatal("无效类型必须报错")
	}
	channels, err := bus.ListChannel(ctx, "t")
	must(t, err)
	if len(channels) != 0 {
		t.Fatal("注册失败留下了半完成路由")
	}
	must(t, raw.Del(ctx, leaseKey).Err())
	must(t, bus.RegisterSubscription(ctx, "t", "c", "id", time.Minute))
	must(t, bus.Push(ctx, "t", []byte("keep")))
	must(t, raw.Del(ctx, leaseKey).Err())
	must(t, raw.Set(ctx, leaseKey, "wrong-type", 0).Err())
	if _, err := bus.SweepSubscriptions(ctx, "t"); err == nil {
		t.Fatal("损坏索引不能当作失活")
	}
	channels, err = bus.ListChannel(ctx, "t")
	must(t, err)
	if len(channels) != 1 {
		t.Fatal("清理失败删除了路由")
	}
	data, ack := take(t, bus, "t", "c")
	if string(data) != "keep" {
		t.Fatal("清理失败删除了积压")
	}
	ack()
}

func TestWrongTypeDoesNotLoseClaimOrRecovery(t *testing.T) {
	ctx := context.Background()
	raw, bus := newBus(t)
	must(t, bus.AddChannel(ctx, "t", "c"))
	must(t, bus.Push(ctx, "t", []byte("keep")))
	must(t, raw.Set(ctx, dueKey, "wrong-type", 0).Err())
	if _, _, err := bus.Pop(ctx, "t", "c", time.Second); err == nil {
		t.Fatal("损坏到期索引必须阻止领取")
	}
	must(t, raw.Del(ctx, dueKey).Err())
	take(t, bus, "t", "c")
	expireReceipts(t, raw)
	readyKey := fixtureChannelKey("t", "c", "ready")
	must(t, raw.Set(ctx, readyKey, "wrong-type", 0).Err())
	if _, err := bus.Recover(ctx, 100); err == nil {
		t.Fatal("恢复遇到错误队列类型必须保留回执")
	}
	must(t, raw.Del(ctx, readyKey).Err())
	expireReceipts(t, raw)
	count, err := bus.Recover(ctx, 100)
	must(t, err)
	if count != 1 {
		t.Fatal("失败的恢复丢失了回执")
	}
	data, ack := take(t, bus, "t", "c")
	if string(data) != "keep" {
		t.Fatalf("消息被破坏：%q", data)
	}
	ack()
}

func TestBadReceiptsDoNotStarveHealthyChannels(t *testing.T) {
	ctx := context.Background()
	raw, bus := newBus(t)
	must(t, bus.AddChannel(ctx, "t", "bad"))
	must(t, bus.AddChannel(ctx, "t", "good"))
	must(t, bus.Push(ctx, "t", []byte("keep")))
	take(t, bus, "t", "good")
	expireReceipts(t, raw)
	for i := 0; i < 3; i++ {
		receipt := fmt.Sprintf("broken-%d", i)
		must(t, raw.HSet(ctx, "hmsgbus:v4:pending:data", receipt, "not-msgpack").Err())
		must(t, raw.ZAdd(ctx, dueKey, redis.Z{Member: receipt, Score: -1}).Err())
	}
	if _, err := bus.Recover(ctx, 3); err == nil {
		t.Fatal("坏回执必须报告错误")
	}
	count, err := bus.Recover(ctx, 3)
	must(t, err)
	if count != 1 {
		t.Fatalf("一整批坏记录也不能阻止后续健康消息恢复：%d", count)
	}
	for i := 0; i < 3; i++ {
		value, err := raw.HGet(ctx, "hmsgbus:v4:pending:data", fmt.Sprintf("broken-%d", i)).Result()
		must(t, err)
		if value != "not-msgpack" {
			t.Fatal("坏记录应保留以便诊断/重试，不能静默丢弃")
		}
	}
	data, ack := take(t, bus, "t", "good")
	if string(data) != "keep" {
		t.Fatal("健康消息损坏")
	}
	ack()
}

type interceptClient struct {
	core.IRedisClient
	raw    *redis.Client
	before func(string, []string, []interface{})
	after  func(string, []string, []interface{}, interface{}, error) (interface{}, error)
}

func (c *interceptClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	if c.before != nil {
		c.before(script, keys, args)
	}
	result, err := c.raw.Eval(ctx, script, keys, args...).Result()
	if c.after != nil {
		return c.after(script, keys, args, result, err)
	}
	return result, err
}

func TestClaimReplyReplayDoesNotPopTwice(t *testing.T) {
	ctx := context.Background()
	raw := testRedis(t)
	client := &interceptClient{IRedisClient: v9.NewRedisClientImplV9(raw), raw: raw}
	bus := core.NewMsgBus(client)
	t.Cleanup(func() { bus.Close() })
	must(t, bus.AddChannel(ctx, "t", "c"))
	must(t, bus.Push(ctx, "t", []byte("first")))
	must(t, bus.Push(ctx, "t", []byte("second")))
	client.after = func(script string, keys []string, args []interface{}, result interface{}, err error) (interface{}, error) {
		if len(keys) == 4 && err == nil {
			return raw.Eval(ctx, script, keys, args...).Result()
		}
		return result, err
	}
	for _, want := range []string{"first", "second"} {
		data, ack := take(t, bus, "t", "c")
		if string(data) != want {
			t.Fatalf("重复请求多取了一条消息：got=%q want=%q", data, want)
		}
		ack()
	}
	empty(t, bus, "t", "c")
}

func TestLostClaimReplyRemainsRecoverable(t *testing.T) {
	ctx := context.Background()
	raw := testRedis(t)
	client := &interceptClient{IRedisClient: v9.NewRedisClientImplV9(raw), raw: raw}
	bus := core.NewMsgBus(client)
	t.Cleanup(func() { bus.Close() })
	must(t, bus.AddChannel(ctx, "t", "c"))
	must(t, bus.Push(ctx, "t", []byte("keep")))
	client.after = func(script string, keys []string, args []interface{}, result interface{}, err error) (interface{}, error) {
		if len(keys) == 4 && err == nil {
			return nil, io.ErrUnexpectedEOF
		}
		return result, err
	}
	if _, _, err := bus.Pop(ctx, "t", "c", time.Second); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("应报告响应丢失：%v", err)
	}
	client.after = nil
	expireReceipts(t, raw)
	count, err := bus.Recover(ctx, 100)
	must(t, err)
	if count != 1 {
		t.Fatal("领取成功但响应丢失时消息必须可恢复")
	}
	data, ack := take(t, bus, "t", "c")
	if string(data) != "keep" {
		t.Fatal("恢复内容不符")
	}
	ack()
}

func TestPublishSnapshotCannotReviveOldChannel(t *testing.T) {
	ctx := context.Background()
	raw := testRedis(t)
	client := &interceptClient{IRedisClient: v9.NewRedisClientImplV9(raw), raw: raw}
	bus := core.NewMsgBus(client)
	t.Cleanup(func() { bus.Close() })
	must(t, bus.AddChannel(ctx, "t", "c"))
	var once sync.Once
	client.after = func(script string, keys []string, args []interface{}, result interface{}, err error) (interface{}, error) {
		if strings.Contains(script, "'HGETALL'") {
			once.Do(func() {
				must(t, bus.RemoveChannel(ctx, "t", "c"))
				must(t, bus.AddChannel(ctx, "t", "c"))
			})
		}
		return result, err
	}
	must(t, bus.Push(ctx, "t", []byte("stale")))
	client.after = nil
	empty(t, bus, "t", "c")
	must(t, bus.Push(ctx, "t", []byte("new")))
	data, ack := take(t, bus, "t", "c")
	if string(data) != "new" {
		t.Fatal("陈旧发布快照复活旧消息")
	}
	ack()
}

func TestWaitingConsumerCannotCrossChannelGeneration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	raw := testRedis(t)
	client := &interceptClient{IRedisClient: v9.NewRedisClientImplV9(raw), raw: raw}
	bus := core.NewMsgBus(client)
	t.Cleanup(func() { bus.Close() })
	must(t, bus.AddChannel(ctx, "t", "c"))
	waiting := make(chan struct{})
	resume := make(chan struct{})
	var once sync.Once
	client.after = func(script string, keys []string, args []interface{}, result interface{}, err error) (interface{}, error) {
		if len(keys) == 4 {
			once.Do(func() {
				close(waiting)
				select {
				case <-resume:
				case <-ctx.Done():
				}
			})
		}
		return result, err
	}
	result := make(chan error, 1)
	go func() {
		data, _, err := bus.Pop(ctx, "t", "c", time.Second)
		if len(data) != 0 {
			result <- fmt.Errorf("旧消费者领取了新一代消息：%q", data)
			return
		}
		result <- err
	}()
	select {
	case <-waiting:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	must(t, bus.RemoveChannel(ctx, "t", "c"))
	must(t, bus.AddChannel(ctx, "t", "c"))
	must(t, bus.Push(ctx, "t", []byte("new")))
	close(resume)
	if err := <-result; !errors.Is(err, xmsgbus.ErrNoData) {
		t.Fatalf("旧领取请求应因代际变化终止：%v", err)
	}
	data, ack := take(t, bus, "t", "c")
	if string(data) != "new" {
		t.Fatal("新消息应留给新一代消费者")
	}
	ack()
}

func TestBackgroundMaintenanceDoesNotScanKeyspace(t *testing.T) {
	ctx := context.Background()
	raw, bus := newBus(t)
	pipe := raw.Pipeline()
	for i := 0; i < 2000; i++ {
		pipe.Set(ctx, fmt.Sprintf("unrelated:%d", i), "value", 0)
	}
	_, err := pipe.Exec(ctx)
	must(t, err)
	must(t, bus.RegisterSubscription(ctx, "t", "c", "id", time.Minute))
	must(t, bus.Push(ctx, "t", []byte("data")))
	take(t, bus, "t", "c")
	expireReceipts(t, raw)
	_, err = bus.Recover(ctx, 100)
	must(t, err)
	must(t, bus.UnregisterSubscription(ctx, "t", "c", "id"))
	_, err = bus.SweepSubscriptions(ctx, "t")
	must(t, err)
	stats, err := raw.Info(ctx, "commandstats").Result()
	must(t, err)
	for _, command := range []string{"cmdstat_keys:", "cmdstat_scan:"} {
		if strings.Contains(stats, command) {
			t.Fatalf("巡检/恢复不得扫描整个数据库：%s", command)
		}
	}
}
