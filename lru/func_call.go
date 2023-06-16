package lru

import (
	"context"
	"time"
	"unsafe"

	"golang.org/x/sync/singleflight"
)

type Result struct {
	ExpireAt time.Time
	Data     unsafe.Pointer
}

type CacheFunc[T any] func(ctx context.Context) (T, error)

var sf singleflight.Group

func FuncCacheCall[T any](ctx context.Context, cache ILRUCache, key string, cacheFunc CacheFunc[T], expireDuration time.Duration) (T, error) {
	res, ok := cache.Get(ctx, key)
	if ok {
		return res.(T), nil
	}
	res, err, _ := sf.Do(key, func() (interface{}, error) {
		res, err := cacheFunc(ctx)
		if err != nil {
			return res, err
		}
		cache.Set(ctx, key, res, time.Now().Add(expireDuration))
		return res, nil
	})
	return res.(T), err
}
