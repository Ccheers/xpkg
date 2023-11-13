package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Call
// 基于otel规范实现一个 call 函数，用于注入任意函数实现链路追踪
func Call[T any](ctx context.Context, spanName string, f func(ctx context.Context) (T, error)) (T, error) {
	// 基于otel规范实现一个 call 函数，用于注入任意函数实现链路追踪
	tr := otel.Tracer("xpkg.trace")
	ctx, span := tr.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	defer span.End()
	reply, err := f(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "OK")
	}
	return reply, err
}
