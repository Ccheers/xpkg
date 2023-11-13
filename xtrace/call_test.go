package xtrace

import (
	"context"
	"fmt"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestCall(t *testing.T) {
	// Setup exporter
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		panic(err)
	}

	// Setup provider with the exporter
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))

	// Set the provider as the global tracer provider
	otel.SetTracerProvider(provider)

	// Call the function with tracing
	_, err = Call(context.TODO(), "测试一下", func(ctx context.Context) (struct{}, error) {
		fmt.Println("Hello, World!")
		return struct{}{}, nil
	})
	if err != nil {
		panic(err)
	}
}
