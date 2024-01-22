package xmsgbus

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

type OTELOptions struct {
	propagator propagation.TextMapPropagator
}

type OTELOption func(o *OTELOptions)

func WithOTELOptionPropagator(propagator propagation.TextMapPropagator) OTELOption {
	return func(o *OTELOptions) {
		o.propagator = propagator
	}
}

func NewOTELOptions(opts ...OTELOption) *OTELOptions {
	_default := &OTELOptions{propagator: propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})}
	for _, opt := range opts {
		opt(_default)
	}
	return _default
}

func (x *OTELOptions) ProducerStartSpan(ctx context.Context, topic string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	tr := otel.Tracer("hmsgbus")
	ctx, span := tr.Start(ctx, topic, trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(attributes...))
	Inject(ctx, x.propagator, md)
	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx, span
}

func (x *OTELOptions) ConsumerStartSpan(ctx context.Context, topic string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	bags, spanCtx := Extract(ctx, x.propagator, md)
	ctx = baggage.ContextWithBaggage(ctx, bags)
	tr := otel.Tracer("hmsgbus")

	ctx, span := tr.Start(trace.ContextWithRemoteSpanContext(ctx, spanCtx), topic,
		trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attributes...))
	return ctx, span
}
