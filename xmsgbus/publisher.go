package xmsgbus

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"google.golang.org/grpc/metadata"
)

type IPublisher[T ITopic] interface {
	Publish(ctx context.Context, event T) error
}

type PublisherOptions[T ITopic] struct {
	Encode EncodeFunc[T]
}

func defaultPublisherOptions[T ITopic]() *PublisherOptions[T] {
	return &PublisherOptions[T]{
		Encode: DefaultEncodeFunc[T],
	}
}

type PublisherOption[T ITopic] func(o *PublisherOptions[T])

func WithEncodeFunc[T ITopic](f EncodeFunc[T]) PublisherOption[T] {
	return func(o *PublisherOptions[T]) {
		o.Encode = f
	}
}

type Publisher[T ITopic] struct {
	msgBus       IMsgBus
	topicManager ITopicManager
	options      *PublisherOptions[T]
	otelOptions  *OTELOptions
}

func NewPublisher[T ITopic](client IMsgBus, topicManager ITopicManager, otelOptions *OTELOptions, opts ...PublisherOption[T]) IPublisher[T] {
	options := defaultPublisherOptions[T]()
	for _, opt := range opts {
		opt(options)
	}
	return &Publisher[T]{
		msgBus:       client,
		topicManager: topicManager,
		options:      options,
		otelOptions:  otelOptions,
	}
}

func (x *Publisher[T]) Publish(ctx context.Context, event T) error {
	topic := event.Topic()
	channels, err := x.msgBus.ListChannel(ctx, topic)
	if err != nil {
		return err
	}
	if len(channels) == 0 {
		return nil
	}

	ctx, span := x.otelOptions.ProducerStartSpan(ctx, topic, semconv.MessagingOperationPublish)
	defer span.End()

	md, _ := metadata.FromOutgoingContext(ctx)
	bs, err := x.options.Encode(ctx, event)
	if err != nil {
		return err
	}

	bs, _ = json.Marshal(&Event{
		Metadata: md,
		Payload:  bs,
	})

	err = x.msgBus.Push(ctx, topic, bs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "ok")
	return nil
}
