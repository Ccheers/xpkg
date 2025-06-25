package xtrace

import (
	"context"
	"fmt"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
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
		psc := trace.SpanContextFromContext(ctx)
		if psc.TraceFlags() != trace.FlagsSampled {
			t.Errorf("trace flags should be sampled")
		}
		return struct{}{}, nil
	})
	if err != nil {
		panic(err)
	}
}

func TestCallWithSampler(t *testing.T) {
	// Setup exporter
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		panic(err)
	}

	// Setup provider with the exporter
	// Set the provider as the global tracer provider
	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	))

	// Call the function with tracing
	_, err = Call(context.TODO(), "测试一下[1]", func(ctx context.Context) (struct{}, error) {
		fmt.Println("Hello, World!")
		psc := trace.SpanContextFromContext(ctx)
		if psc.TraceFlags() != trace.FlagsSampled {
			t.Errorf("trace flags should be sampled")
		}
		return struct{}{}, nil
	})
	if err != nil {
		panic(err)
	}

	// Setup provider with the exporter
	// Set the provider as the global tracer provider
	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.NeverSample()),
	))

	// Call the function with tracing
	_, err = Call(context.TODO(), "测试一下[2]", func(ctx context.Context) (struct{}, error) {
		fmt.Println("Hello, World!")
		psc := trace.SpanContextFromContext(ctx)
		if psc.TraceFlags() == trace.FlagsSampled {
			t.Errorf("trace flags should not be sampled")
		}
		return struct{}{}, nil
	})
	if err != nil {
		panic(err)
	}
}
