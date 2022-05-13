package routinepool

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const benchmarkTimes = 10000

func DoCopyStack(a, b int) int {
	if b < 100 {
		return DoCopyStack(0, b+1)
	}
	return 0
}

func testFunc() {
	DoCopyStack(0, 0)
}

func testPanicFunc(ctx context.Context) {
	panic("test")
}

func TestPool(t *testing.T) {
	p := NewPool("test", 100, NewConfig())
	var n int32
	var wg sync.WaitGroup
	for i := 0; i < 2000; i++ {
		wg.Add(1)
		p.Go(func(ctx context.Context) {
			defer wg.Done()
			atomic.AddInt32(&n, 1)
		})
	}
	wg.Wait()
	if n != 2000 {
		t.Error(n)
	}
}

func TestPool_CtxGo(t *testing.T) {
	p := NewPool("test", 100, NewConfig())
	var n int32
	var wg sync.WaitGroup
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	for i := 0; i < 2000; i++ {
		wg.Add(1)
		p.CtxGo(timeoutCtx, func(ctx context.Context) {
			defer wg.Done()
			atomic.AddInt32(&n, 1)
		})
	}
	closeChan := make(chan struct{}, 0)
	go func() {
		wg.Wait()
		closeChan <- struct{}{}
	}()
	select {
	case <-timeoutCtx.Done():
		return
	case <-closeChan:
	}
	if n != 2000 {
		t.Error(n)
	}
}

func TestPoolPanic(t *testing.T) {
	p := NewPool("test", 100, NewConfig())
	p.Go(testPanicFunc)
}

func BenchmarkPool(b *testing.B) {
	config := NewConfig()
	config.ScaleThreshold = 1
	p := NewPool("benchmark", int32(runtime.GOMAXPROCS(0)), config)
	var wg sync.WaitGroup
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(benchmarkTimes)
		for j := 0; j < benchmarkTimes; j++ {
			p.Go(func(ctx context.Context) {
				testFunc()
				wg.Done()
			})
		}
		wg.Wait()
	}
}

func BenchmarkGo(b *testing.B) {
	var wg sync.WaitGroup
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(benchmarkTimes)
		for j := 0; j < benchmarkTimes; j++ {
			go func() {
				testFunc()
				wg.Done()
			}()
		}
		wg.Wait()
	}
}
