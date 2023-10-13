package lru

import (
	"context"
	"sync"
	"time"

	"github.com/ccheers/xpkg/generic/containerx/heap"
)

type ILRUCache interface {
	Set(ctx context.Context, key string, value interface{}, expireAt time.Time)
	Get(ctx context.Context, key string) (interface{}, bool)
}

type node struct {
	expireAt time.Time
	key      string
	value    interface{}
}

type T struct {
	heap    *heap.Heap[*node]
	objPool sync.Pool

	latestGCAt time.Time

	gcLock sync.Mutex

	maxLen int

	mu sync.RWMutex
	mm map[string]*node
}

func NewLRUCache(maxLen int) ILRUCache {
	return &T{
		heap: heap.New[*node](func(a, b *node) bool {
			return a.expireAt.After(b.expireAt)
		}),
		objPool: sync.Pool{
			New: func() any {
				return &node{}
			},
		},
		maxLen: maxLen,
		mm:     make(map[string]*node, maxLen),
	}
}

func (x *T) Set(ctx context.Context, key string, value interface{}, expireAt time.Time) {
	defer x.gcTick()

	x.mu.Lock()
	node := x.objPool.Get().(*node)
	node.expireAt = expireAt
	node.key = key
	node.value = value

	x.heap.Push(node)
	x.mm[key] = node

	if len(x.mm) > x.maxLen {
		node, ok := x.heap.Pop()
		if ok {
			delete(x.mm, node.key)
		}
		x.objPool.Put(node)
	}
	x.mu.Unlock()
}

func (x *T) Get(ctx context.Context, key string) (interface{}, bool) {
	defer x.gcTick()

	x.mu.RLock()
	node, ok := x.mm[key]
	x.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if node.expireAt.Before(time.Now()) {
		return nil, false
	}
	return node.value, ok
}

func (x *T) gcTick() {
	const calmDuration = time.Second * 10
	if !x.gcLock.TryLock() {
		return
	}
	defer x.gcLock.Unlock()

	now := time.Now()
	if now.Sub(x.latestGCAt) < calmDuration {
		return
	}
	x.mu.Lock()
	for {
		node, ok := x.heap.Pop()
		if !ok {
			break
		}
		if node.expireAt.Before(now) {
			delete(x.mm, node.key)
			node.value = nil
			x.objPool.Put(node)
		} else {
			x.heap.Push(node)
			break
		}
	}
	x.latestGCAt = now
	mm := make(map[string]*node, len(x.mm))
	// 缩小 bucket 空隙
	for k, v := range x.mm {
		mm[k] = v
	}
	x.mm = mm
	x.mu.Unlock()
}
