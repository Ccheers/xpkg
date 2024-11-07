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

	cache.Set(ctx, "1", 1, time.Now().Add(time.Minute))
	time.Sleep(time.Second * 10)
	cache.Set(ctx, "1", 1, time.Now().Add(-time.Second))
	_, ok = cache.Get(ctx, "1")
	if ok {
		t.Fatal("should not be ok")
	}

}

func BenchmarkLRUCache(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cache := NewLRUCache(3)
		ctx := context.TODO()
		cache.Set(ctx, "1", 1, time.Now().Add(time.Second))
		cache.Set(ctx, "2", 2, time.Now().Add(time.Second))
		cache.Set(ctx, "3", 3, time.Now().Add(time.Second))
		cache.Set(ctx, "4", 4, time.Now().Add(time.Second))
		cache.Set(ctx, "5", 5, time.Now().Add(time.Second))
		cache.Set(ctx, "6", 6, time.Now().Add(time.Second))
		cache.Set(ctx, "7", 7, time.Now().Add(time.Second))
		cache.Set(ctx, "8", 8, time.Now().Add(time.Second))
		cache.Set(ctx, "9", 9, time.Now().Add(time.Second))
		cache.Set(ctx, "10", 10, time.Now().Add(time.Second))
		cache.Set(ctx, "11", 11, time.Now().Add(time.Second))
		cache.Set(ctx, "12", 12, time.Now().Add(time.Second))
		cache.Set(ctx, "13", 13, time.Now().Add(time.Second))
		cache.Set(ctx, "14", 14, time.Now().Add(time.Second))
		cache.Set(ctx, "15", 15, time.Now().Add(time.Second))
		cache.Set(ctx, "16", 16, time.Now().Add(time.Second))
		cache.Set(ctx, "17", 17, time.Now().Add(time.Second))
		cache.Set(ctx, "18", 18, time.Now().Add(time.Second))
		cache.Set(ctx, "19", 19, time.Now().Add(time.Second))
		cache.Set(ctx, "20", 20, time.Now().Add(time.Second))
		cache.Set(ctx, "21", 21, time.Now().Add(time.Second))
		cache.Set(ctx, "22", 22, time.Now().Add(time.Second))
		cache.Set(ctx, "23", 23, time.Now().Add(time.Second))
		cache.Set(ctx, "24", 24, time.Now().Add(time.Second))
		cache.Set(ctx, "25", 25, time.Now().Add(time.Second))
		cache.Set(ctx, "26", 26, time.Now().Add(time.Second))
	}
}
