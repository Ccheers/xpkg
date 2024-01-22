package memory

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ccheers/xpkg/xmsgbus"
)

type Storage struct {
	mu sync.Mutex
	mm map[string]*node

	latestGCAt int64
}

func NewStorage() xmsgbus.ISharedStorage {
	return &Storage{
		mu:         sync.Mutex{},
		mm:         make(map[string]*node),
		latestGCAt: 0,
	}
}

type node struct {
	value     interface{}
	expiredAt time.Time
}

func (x *Storage) SetEx(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	defer x.gcTick()
	x.mu.Lock()
	defer x.mu.Unlock()
	x.mm[key] = &node{
		value:     value,
		expiredAt: time.Now().Add(ttl),
	}
	return nil
}

func (x *Storage) Keys(ctx context.Context, prefix string) ([]string, error) {
	defer x.gcTick()
	x.mu.Lock()
	defer x.mu.Unlock()
	now := time.Now()
	var keys []string
	for key, node := range x.mm {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if node.expiredAt.Before(now) {
			continue
		}
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (x *Storage) Del(ctx context.Context, key string) error {
	defer x.gcTick()
	x.mu.Lock()
	defer x.mu.Unlock()
	delete(x.mm, key)
	return nil
}

func (x *Storage) gcTick() {
	latestGCAt := atomic.LoadInt64(&x.latestGCAt)
	now := time.Now()
	if now.Unix() < latestGCAt {
		return
	}
	if !atomic.CompareAndSwapInt64(&x.latestGCAt, latestGCAt, now.Unix()+10) {
		return
	}

	x.mu.Lock()
	defer x.mu.Unlock()
	for key, node := range x.mm {
		if node.expiredAt.Before(now) {
			delete(x.mm, key)
		}
	}
}
