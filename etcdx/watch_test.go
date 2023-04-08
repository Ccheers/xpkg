package etcdx

import (
	"context"
	"testing"
	"time"
)

type dummyWatchCallback struct {
	m map[string]*TargetNode
}

func (x *dummyWatchCallback) BatchSet(nodes []TargetNode) {
	for _, node := range nodes {
		node := node
		x.m[node.Key] = &node
	}
}

func (x *dummyWatchCallback) BatchDelete(strings []string) {
	for _, key := range strings {
		delete(x.m, key)
	}
}

func (x *dummyWatchCallback) Reset() {
	x.m = make(map[string]*TargetNode)
}

func TestClientX_BatchWatch(t *testing.T) {
	client := newETCDClient("127.0.0.1:2379")

	x := NewClientX(client, newClientLog())
	cb := &dummyWatchCallback{m: make(map[string]*TargetNode)}
	ctx := context.Background()

	go func() {
		err := x.Watch(ctx, "/test/123", cb, WithWatchPrefix(), WithWatchSessionTTL(5))
		if err != nil {
			t.Fatalf("Watch failed: %v", err)
		}
	}()

	x.Put(ctx, "/test/123/1", "1")
	x.Put(ctx, "/test/123/2", "2")
	x.Put(ctx, "/test/123/3", "3")
	time.Sleep(time.Second)
	if cb.m["/test/123/1"].Value != "1" {
		t.Fatalf("except 1 got %s", cb.m["/test/123/1"].Value)
	}
	if cb.m["/test/123/2"].Value != "2" {
		t.Fatalf("except 2 got %s", cb.m["/test/123/2"].Value)
	}
	if cb.m["/test/123/3"].Value != "3" {
		t.Fatalf("except 3 got %s", cb.m["/test/123/3"].Value)
	}

	x.Put(ctx, "/test/123/3", "4")
	time.Sleep(time.Second)
	if cb.m["/test/123/3"].Value != "4" {
		t.Fatalf("except 4 got %s", cb.m["/test/123/3"].Value)
	}

	x.Delete(ctx, "/test/123/3")
	time.Sleep(time.Second)
	if val, ok := cb.m["/test/123/3"]; ok {
		t.Fatalf("except nil got %s", val.Value)
	}

	x.Close()
	time.Sleep(time.Second)

}
