package routinepool

import (
	"context"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	prometheusexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const benchmarkTimes = 10000

func DoCopyStack(a, b int) int {
	if b < 100 {
		return DoCopyStack(0, b+1)
	}
	return 0
}

func testFunc() {
	_ = DoCopyStack(0, 0)
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
	closeChan := make(chan struct{})
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
	config := NewConfig(WithScaleThreshold(1))
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

func TestMetrics(t *testing.T) {
	mp, err := newMeterProvider()
	if err != nil {
		panic(err)
	}
	// 注册路由
	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))
	go func() {
		http.ListenAndServe(":23333", handler)
	}()
	p := NewPool("test", 100, NewConfig(WithMeterProvider(mp)))
	var wg sync.WaitGroup
	for i := 0; i < 2000; i++ {
		wg.Add(1)
		p.Go(func(ctx context.Context) {
			time.Sleep(time.Second * time.Duration(rand.Intn(10)))
			defer wg.Done()
		})
	}
	wg.Wait()

}

func newMeterProvider() (metric.MeterProvider, error) {
	exporter, err := prometheusexporter.New()
	if err != nil {
		return nil, err
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("test"),
				attribute.String("environment", "local"),
			),
		),
		sdkmetric.WithReader(exporter),
		sdkmetric.WithView(),
	)
	otel.SetMeterProvider(provider)
	return provider, nil
}
