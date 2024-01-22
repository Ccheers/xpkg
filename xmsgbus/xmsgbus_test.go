package xmsgbus_test

import (
	"context"
	"log"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ccheers/xpkg/xmsgbus"
	"github.com/ccheers/xpkg/xmsgbus/impl/memory"
	"google.golang.org/grpc/metadata"
)

type dummyEvent struct {
	Value uint32
}

func (x *dummyEvent) Topic() string {
	return "test"
}

type simpleCas struct {
	mu sync.Mutex
	mm map[string]string
}

func newSimpleCas() *simpleCas {
	return &simpleCas{
		mu: sync.Mutex{},
		mm: make(map[string]string),
	}
}

func (x *simpleCas) CAS(key, src, dst string) bool {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.mm[key] == src {
		x.mm[key] = dst
		return true
	}
	return false
}

func TestDefaultDecodeFunc(t *testing.T) {
	type args struct {
		ctx context.Context
		bs  []byte
	}
	type testCase[T xmsgbus.ITopic] struct {
		name    string
		args    args
		want    T
		wantErr bool
	}
	tests := []testCase[*dummyEvent]{
		{
			name: "1",
			args: args{
				ctx: context.TODO(),
				bs:  []byte("{\"Value\":123}"),
			},
			want:    &dummyEvent{Value: 123},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := xmsgbus.DefaultDecodeFunc[*dummyEvent](tt.args.ctx, tt.args.bs)
			if (err != nil) != tt.wantErr {
				t.Errorf("DefaultDecodeFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultDecodeFunc() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultEncodeFunc(t *testing.T) {
	type args[T xmsgbus.ITopic] struct {
		ctx context.Context
		dst T
	}
	type testCase[T xmsgbus.ITopic] struct {
		name    string
		args    args[T]
		want    []byte
		wantErr bool
	}
	tests := []testCase[*dummyEvent]{
		{
			name: "1",
			args: args[*dummyEvent]{
				ctx: context.TODO(),
				dst: &dummyEvent{Value: 123},
			},
			want:    []byte("{\"Value\":123}"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := xmsgbus.DefaultEncodeFunc(tt.args.ctx, tt.args.dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("DefaultEncodeFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultEncodeFunc() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultSubscriberCheckFunc(t *testing.T) {
	type args[T xmsgbus.ITopic] struct {
		ctx context.Context
		dst T
	}
	type testCase[T xmsgbus.ITopic] struct {
		name string
		args args[T]
		want bool
	}
	tests := []testCase[*dummyEvent]{
		{
			name: "1",
			args: args[*dummyEvent]{
				ctx: context.TODO(),
				dst: &dummyEvent{Value: 1},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := xmsgbus.DefaultSubscriberCheckFunc[*dummyEvent](tt.args.ctx, tt.args.dst); got != tt.want {
				t.Errorf("DefaultSubscriberCheckFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultSubscriberHandleFunc(t *testing.T) {
	type args[T xmsgbus.ITopic] struct {
		ctx context.Context
		dst T
	}
	type testCase[T xmsgbus.ITopic] struct {
		name    string
		args    args[T]
		wantErr bool
	}
	tests := []testCase[*dummyEvent]{
		{
			name: "1",
			args: args[*dummyEvent]{
				ctx: context.TODO(),
				dst: &dummyEvent{Value: 1},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := xmsgbus.DefaultSubscriberHandleFunc(tt.args.ctx, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("DefaultSubscriberHandleFunc() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetadataSupplier_Get(t *testing.T) {
	type fields struct {
		metadata metadata.MD
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "1",
			fields: fields{
				metadata: metadata.New(map[string]string{
					"test": "123",
				}),
			},
			args: args{
				key: "test",
			},
			want: "123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := xmsgbus.NewMetadataSupplier(
				tt.fields.metadata,
			)
			if got := s.Get(tt.args.key); got != tt.want {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetadataSupplier_Keys(t *testing.T) {
	type fields struct {
		metadata metadata.MD
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "1",
			fields: fields{
				metadata: metadata.New(map[string]string{
					"test": "123",
				}),
			},
			want: []string{"test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := xmsgbus.NewMetadataSupplier(
				tt.fields.metadata,
			)
			if got := s.Keys(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Keys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetadataSupplier_Set(t *testing.T) {
	type fields struct {
		metadata metadata.MD
	}
	type args struct {
		key   string
		value string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "1",
			fields: fields{
				metadata: metadata.New(map[string]string{
					"test": "123",
				}),
			},
			args: args{
				key:   "test",
				value: "456",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := xmsgbus.NewMetadataSupplier(
				tt.fields.metadata,
			)
			s.Set(tt.args.key, tt.args.value)
			if s.Get(tt.args.key) != tt.args.value {
				t.Errorf("need =%s, got=%s", tt.args.value, s.Get(tt.args.key))
			}
		})
	}
}

func TestPublisher_Publish(t *testing.T) {
	type args[T xmsgbus.ITopic] struct {
		ctx   context.Context
		event T
	}
	type testCase[T xmsgbus.ITopic] struct {
		name    string
		x       xmsgbus.IPublisher[T]
		args    args[T]
		wantErr bool
	}
	msgbus := memory.NewMsgBus()
	storage := memory.NewStorage()
	manager := xmsgbus.NewTopicManager(context.TODO(), msgbus, newSimpleCas(), storage)
	tests := []testCase[*dummyEvent]{
		{
			name: "1",
			x: xmsgbus.NewPublisher[*dummyEvent](
				msgbus,
				manager,
				xmsgbus.NewOTELOptions(),
			),
			args: args[*dummyEvent]{
				ctx: context.TODO(),
				event: &dummyEvent{
					Value: 123,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.x.Publish(tt.args.ctx, tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("Publish() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSubscriber_Close(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	type testCase[T xmsgbus.ITopic] struct {
		name string
		x    xmsgbus.ISubscriber[T]
		args args
	}
	msgbus := memory.NewMsgBus()
	storage := memory.NewStorage()
	manager := xmsgbus.NewTopicManager(context.TODO(), msgbus, newSimpleCas(), storage)
	tests := []testCase[*dummyEvent]{
		{
			name: "1",
			x: xmsgbus.NewSubscriber[*dummyEvent](
				"test",
				"channel",
				msgbus,
				xmsgbus.NewOTELOptions(),
				manager,
			),
			args: args{
				ctx: context.TODO(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.x.Close(tt.args.ctx)
		})
	}
}

func TestSubscriber_Handle(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	type testCase[T xmsgbus.ITopic] struct {
		name    string
		x       xmsgbus.ISubscriber[T]
		args    args
		wantErr bool
	}
	msgbus := memory.NewMsgBus()
	storage := memory.NewStorage()
	manager := xmsgbus.NewTopicManager(context.TODO(), msgbus, newSimpleCas(), storage)
	go func() {
		time.Sleep(time.Second)
		err := xmsgbus.NewPublisher[*dummyEvent](
			msgbus,
			manager,
			xmsgbus.NewOTELOptions(),
		).Publish(context.TODO(), &dummyEvent{Value: 123})
		if err != nil {
			panic(err)
		}
	}()
	tests := []testCase[*dummyEvent]{
		{
			name: "1",
			x: xmsgbus.NewSubscriber[*dummyEvent](
				"test",
				"channel",
				msgbus,
				xmsgbus.NewOTELOptions(),
				manager,
				xmsgbus.WithHandleFunc[*dummyEvent](func(ctx context.Context, dst *dummyEvent) error {
					log.Println(dst)
					return nil
				}),
			),
			args: args{
				ctx: context.TODO(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.x.Handle(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
