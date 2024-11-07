package xaop

import (
	"context"
	"log"
	"testing"
)

func TestHandleChain(t *testing.T) {
	type args[Req any, Resp any] struct {
		mainFn AOPHandleFunc[Req, Resp]
		aopFns []AOPChainFunc[Req, Resp]
	}
	type testCase[Req any, Resp any] struct {
		name string
		args args[Req, Resp]
		want AOPHandleFunc[Req, Resp]
	}
	tests := []testCase[uint32, int32]{
		{
			name: "1",
			args: args[uint32, int32]{
				mainFn: func(ctx context.Context, a uint32) (int32, error) {
					log.Println("main", a)
					return int32(a + 1), nil
				},
				aopFns: []AOPChainFunc[uint32, int32]{
					func(next AOPHandleFunc[uint32, int32]) AOPHandleFunc[uint32, int32] {
						// 前置
						return func(ctx context.Context, u uint32) (int32, error) {
							log.Println("前置", u)
							return next(ctx, u+1)
						}
					},
					func(next AOPHandleFunc[uint32, int32]) AOPHandleFunc[uint32, int32] {
						// 后置
						return func(ctx context.Context, u uint32) (int32, error) {
							reply, err := next(ctx, u)
							log.Println("后置", reply)
							return reply, err
						}
					},
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HandleChain(tt.args.mainFn, tt.args.aopFns...)
			reply, err := got(context.Background(), 1)
			log.Println("reply", reply, "err", err)
		})
	}
}
