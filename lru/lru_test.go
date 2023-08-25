package lru

import (
	"context"
	"testing"
	"time"
)

func TestNewLRUCache(t *testing.T) {
	cache := NewLRUCache(3)
	ctx := context.TODO()

	cache.Set(ctx, "1", 1, time.Now().Add(time.Second))
	cache.Set(ctx, "2", 2, time.Now().Add(time.Second))

	_, ok := cache.Get(ctx, "1")


	if !ok {
		t.Fatal("should be ok")
	}
	time.Sleep(time.Second * 10)
	_, ok = cache.Get(ctx, "1")
	if ok {
		t.Fatal("should not be ok")
	}

	if len(cache.(*T).mm) > 0 {
		t.Fatal("should be empty")
	}
}
