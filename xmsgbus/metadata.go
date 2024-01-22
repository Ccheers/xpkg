package xmsgbus

import (
	"context"

	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

// assert that MetadataSupplier implements the TextMapCarrier interface
var _ propagation.TextMapCarrier = (*MetadataSupplier)(nil)

type MetadataSupplier struct {
	metadata metadata.MD
}

func NewMetadataSupplier(metadata metadata.MD) *MetadataSupplier {
	return &MetadataSupplier{metadata: metadata}
}

func (s *MetadataSupplier) Get(key string) string {
	values := s.metadata.Get(key)
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func (s *MetadataSupplier) Set(key, value string) {
	s.metadata.Set(key, value)
}

func (s *MetadataSupplier) Keys() []string {
	out := make([]string, 0, len(s.metadata))
	for key := range s.metadata {
		out = append(out, key)
	}

	return out
}

// Inject injects cross-cutting concerns from the ctx into the metadata.
func Inject(ctx context.Context, p propagation.TextMapPropagator, metadata metadata.MD) {
	p.Inject(ctx, NewMetadataSupplier(metadata))
}

// Extract extracts the metadata from ctx.
func Extract(ctx context.Context, p propagation.TextMapPropagator, metadata metadata.MD) (
	baggage.Baggage, sdktrace.SpanContext,
) {
	ctx = p.Extract(ctx, NewMetadataSupplier(metadata))

	return baggage.FromContext(ctx), sdktrace.SpanContextFromContext(ctx)
}
