package lru

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestFuncCacheCall(t *testing.T) {
	type args[T any] struct {
		ctx            context.Context
		cache          ILRUCache
		key            string
		cacheFunc      CacheFunc[T]
		expireDuration time.Duration
	}
	type testCase[T any] struct {
		name    string
		args    args[T]
		want    T
		wantErr bool
	}
	cache := NewLRUCache(1)
	tests := []testCase[uint]{
		{
			name: "1",
			args: args[uint]{
				ctx:   context.Background(),
				cache: cache,
				key:   "test",
				cacheFunc: func(ctx context.Context) (uint, error) {
					return 1, nil
				},
				expireDuration: time.Minute,
			},
			want: 1,
		},
		{
			name: "2",
			args: args[uint]{
				ctx:   context.Background(),
				cache: cache,
				key:   "test",
				cacheFunc: func(ctx context.Context) (uint, error) {
					return 2, nil
				},
				expireDuration: time.Second,
			},
			want: 1,
		},
		{
			name: "3",
			args: args[uint]{
				ctx:   context.Background(),
				cache: cache,
				key:   "test3",
				cacheFunc: func(ctx context.Context) (uint, error) {
					return 3, nil
				},
				expireDuration: time.Second,
			},
			want: 3,
		},
		{
			name: "4",
			args: args[uint]{
				ctx:   context.Background(),
				cache: cache,
				key:   "test",
				cacheFunc: func(ctx context.Context) (uint, error) {
					return 4, nil
				},
				expireDuration: time.Second,
			},
			want: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FuncCacheCall(tt.args.ctx, tt.args.cache, tt.args.key, tt.args.cacheFunc, tt.args.expireDuration)
			if (err != nil) != tt.wantErr {
				t.Errorf("FuncCacheCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FuncCacheCall() got = %v, want %v", got, tt.want)
			}
		})
	}
}
