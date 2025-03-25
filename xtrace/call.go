package xtrace

import (
	"context"
	"fmt"
	"runtime/debug"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Call
// 基于otel规范实现一个 call 函数，用于注入任意函数实现链路追踪
// 使用这个函数的 trace 一定会被记录, 请自行判断是否调用, 或则使用 CallWithSampler
func Call[T any](ctx context.Context, spanName string, f func(ctx context.Context) (T, error)) (T, error) {
	return CallWithSampler(ctx, spanName, sdktrace.AlwaysSample(), f)
}

// CallWithSampler
// 基于 otel 规范实现一个 call 函数，用于注入任意函数实现链路追踪
// sdktrace.Sampler 用于判断是否采样
func CallWithSampler[T any](ctx context.Context, spanName string, sampler sdktrace.Sampler, f func(ctx context.Context) (T, error)) (T, error) {
	tr := otel.Tracer("xpkg.trace")
	psc := trace.SpanContextFromContext(ctx)
	if !psc.IsValid() {
		tid, sid := idGenerator.NewIDs(ctx)
		result := sampler.ShouldSample(sdktrace.SamplingParameters{
			ParentContext: nil,
			TraceID:       tid,
			Name:          spanName,
			Kind:          trace.SpanKindInternal,
		})
		if result.Decision == sdktrace.RecordAndSample {
			ctx = trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    tid,
				SpanID:     sid,
				TraceFlags: trace.FlagsSampled,
			}))
		}
	}
	ctx, span := tr.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	defer span.End()
	defer func() {
		r := recover()
		if r != nil {
			err := fmt.Errorf("panic: %v, stack: %s", r, debug.Stack())
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}()
	reply, err := f(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "OK")
	}
	return reply, err
}
