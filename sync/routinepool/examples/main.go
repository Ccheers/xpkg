package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ccheers/xpkg/sync/routinepool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	prometheusexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func main() {
	mp, err := newMeterProvider()
	if err != nil {
		panic(err)
	}
	pool := routinepool.NewPool("test", 8,
		routinepool.NewConfig(
			routinepool.WithScaleThreshold(10),
			routinepool.WithMeterProvider(mp),
			routinepool.WithErrorHandler(func(ctx context.Context, err error) {
				// handle panic
				log.Println(err.Error(), "module", "[main][for]")
			}),
		))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		i := i
		pool.CtxGo(ctx, func(ctx context.Context) {
			defer wg.Done()
			// do something
			if i%100 == 0 {
				panic("will panic")
			}
			time.Sleep(time.Millisecond)
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
