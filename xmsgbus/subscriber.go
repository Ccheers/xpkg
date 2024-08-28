package xmsgbus

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"runtime/debug"
	"time"

	"github.com/ccheers/xpkg/xlogger"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"google.golang.org/grpc/metadata"
)

type ISubscriber[T ITopic] interface {
	Handle(ctx context.Context) error
	// Close 取消订阅，这个操作必须成功
	Close(ctx context.Context)
}

type SubscriberHandleFunc[T ITopic] func(ctx context.Context, dst T) error

func DefaultSubscriberHandleFunc[T ITopic](ctx context.Context, dst T) error {
	xlogger.DefaultLogger.Log(xlogger.LevelDebug, "subscriber handle ...", dst.Topic())
	return nil
}

type SubscriberCheckFunc[T ITopic] func(ctx context.Context, dst T) bool

func DefaultSubscriberCheckFunc[T ITopic](ctx context.Context, dst T) bool {
	xlogger.DefaultLogger.Log(xlogger.LevelDebug, "subscriber check ...", dst.Topic())
	return true
}

type SubscriberOptions[T ITopic] struct {
	HandleEvent SubscriberHandleFunc[T]
	CheckEvent  SubscriberCheckFunc[T]
	Decode      DecodeFunc[T]
}

func defaultSubscriberOptions[T ITopic]() *SubscriberOptions[T] {
	return &SubscriberOptions[T]{
		HandleEvent: DefaultSubscriberHandleFunc[T],
		CheckEvent:  DefaultSubscriberCheckFunc[T],
		Decode:      DefaultDecodeFunc[T],
	}
}

type SubscriberOption[T ITopic] func(o *SubscriberOptions[T])

func WithHandleFunc[T ITopic](f SubscriberHandleFunc[T]) SubscriberOption[T] {
	return func(o *SubscriberOptions[T]) {
		o.HandleEvent = f
	}
}

func WithCheckFunc[T ITopic](f SubscriberCheckFunc[T]) SubscriberOption[T] {
	return func(o *SubscriberOptions[T]) {
		o.CheckEvent = f
	}
}

func WithDecodeFunc[T ITopic](f DecodeFunc[T]) SubscriberOption[T] {
	return func(o *SubscriberOptions[T]) {
		o.Decode = f
	}
}

type Subscriber[T ITopic] struct {
	msgBus      IMsgBus
	otelOptions *OTELOptions

	uuid    string
	topic   string
	channel string

	options *SubscriberOptions[T]

	topicManager ITopicManager
}

func NewSubscriber[T ITopic](topic, channel string, client IMsgBus, otelOptions *OTELOptions, topicManager ITopicManager, opts ...SubscriberOption[T]) ISubscriber[T] {
	options := defaultSubscriberOptions[T]()
	for _, opt := range opts {
		opt(options)
	}
	return &Subscriber[T]{
		msgBus:       client,
		otelOptions:  otelOptions,
		uuid:         uuid.New().String(),
		topic:        topic,
		channel:      channel,
		options:      options,
		topicManager: topicManager,
	}
}

func (x *Subscriber[T]) Handle(ctx context.Context) (err error) {
	defer func() {
		// recover panic
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v\nTrace: %s", r, string(debug.Stack()))
		}
	}()
	err = x.topicManager.Register(ctx, x.topic, x.channel, x.uuid, time.Minute)
	if err != nil {
		return err
	}
	// 一次 handle 监听不超过 [30,45) 秒
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(30+rand.Intn(15)))
	defer cancel()

	bs, ack, err := x.msgBus.Pop(timeoutCtx, x.topic, x.channel, 0)
	if err != nil {
		return err
	}
	defer ack()

	var dst Event
	err = json.Unmarshal(bs, &dst)
	if err != nil {
		return err
	}
	event, err := x.options.Decode(ctx, dst.Payload)
	if err != nil {
		return err
	}

	// 未通过校验则直接返回
	if !x.options.CheckEvent(ctx, event) {
		return ErrCheckFailed
	}

	ctx = metadata.NewIncomingContext(ctx, dst.Metadata)
	ctx, span := x.otelOptions.ConsumerStartSpan(ctx, dst.Topic, semconv.MessagingOperationProcess)
	defer span.End()

	err = x.options.HandleEvent(ctx, event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "ok")
	return nil
}

func (x *Subscriber[T]) Close(ctx context.Context) {
	x.topicManager.Unregister(ctx, x.topic, x.channel, x.uuid)
}
