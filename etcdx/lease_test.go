package etcdx

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type emptyLog struct{}

func (x *emptyLog) Logf(level LogLevel, template string, args ...interface{}) {
	log.Printf(fmt.Sprintf("[%s] %s\n", level, template), args...)
}

func (x *emptyLog) Logw(level LogLevel, keyPairs ...interface{}) {
	args := []interface{}{"level", level}
	args = append(args, keyPairs...)
	log.Println(args...)
}

func newETCDClient(addr string) *clientv3.Client {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:            []string{addr},
		DialTimeout:          time.Second * 5,
		DialKeepAliveTime:    time.Second * 5,
		DialKeepAliveTimeout: time.Second * 5,
		Username:             "",
		Password:             "",
	})
	if err != nil {
		panic(err)
	}
	return cli
}

func newClientLog() *clientLogger {
	return &clientLogger{logger: &emptyLog{}}
}

func TestClientX_TryLease(t *testing.T) {
	client := newETCDClient("127.0.0.1:2379")
	type fields struct {
		Client *clientv3.Client
	}
	type args struct {
		ctx   context.Context
		key   string
		value string
		opts  []WithLeaseOptions
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    func(context.Context)
		wantErr bool
	}{
		{
			name: "1",
			fields: fields{
				Client: client,
			},
			args: args{
				ctx:   context.Background(),
				key:   "/test/watch1",
				value: "123",
				opts:  []WithLeaseOptions{WithLeaseTTL(5)},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "2",
			fields: fields{
				Client: client,
			},
			args: args{
				ctx:   context.Background(),
				key:   "/test/watch1",
				value: "123",
				opts:  []WithLeaseOptions{WithLeaseTTL(5)},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := &ClientX{
				Client: tt.fields.Client,
				logger: newClientLog(),
			}
			_, err := x.TryLease(tt.args.ctx, tt.args.key, tt.args.value, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("TryLease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
