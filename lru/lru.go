package lru

import (
	"context"
	"sync"
	"time"

	v2 "github.com/ccheers/xpkg/lru/v2"
)

type ILRUCache interface {
	Set(ctx context.Context, key string, value interface{}, expireAt time.Time)
	Get(ctx context.Context, key string) (interface{}, bool)
}

type T struct {
	arcCache   *v2.ARCCache
	latestGCAt time.Time
	mu         sync.Mutex
	mm         map[string]time.Time
}

func NewLRUCache(maxLen int) ILRUCache {
	cache, _ := v2.NewARC(int(uint32(maxLen)))
	return &T{
		arcCache:   cache,
		latestGCAt: time.Unix(0, 0),
		mm:         make(map[string]time.Time, maxLen),
	}
}

func (x *T) Set(ctx context.Context, key string, value interface{}, expireAt time.Time) {
	defer x.gcTick()
	x.mu.Lock()
	x.mm[key] = expireAt
	x.mu.Unlock()
	x.arcCache.Add(key, value)
}

func (x *T) Get(ctx context.Context, key string) (interface{}, bool) {
	x.gcTick()
	return x.arcCache.Get(key)
}

func (x *T) gcTick() {
	const calmDuration = time.Second * 10
	if !x.mu.TryLock() {
		return
	}
	defer x.mu.Unlock()

	now := time.Now()
	if now.Sub(x.latestGCAt) < calmDuration {
		return
	}
	for key, t := range x.mm {
		if t.Before(now) {
			x.arcCache.Remove(key)
			delete(x.mm, key)
		}
	}
	x.latestGCAt = now
}
