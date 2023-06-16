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
		mu:     sync.RWMutex{},
		mm:     make(map[string]*node),
	}
}

func (x *T) Set(ctx context.Context, key string, value interface{}, expireAt time.Time) {
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
