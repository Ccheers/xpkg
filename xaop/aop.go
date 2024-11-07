package xaop

import "context"

type AOPHandleFunc[Req any, Resp any] func(context.Context, Req) (Resp, error)

type AOPChainFunc[Req any, Resp any] func(next AOPHandleFunc[Req, Resp]) AOPHandleFunc[Req, Resp]

func HandleChain[Req any, Resp any](mainFn AOPHandleFunc[Req, Resp], aopFns ...AOPChainFunc[Req, Resp]) AOPHandleFunc[Req, Resp] {
	if len(aopFns) == 0 {
		return mainFn
	}

	fn := mainFn

	for i := len(aopFns) - 1; i >= 0; i-- {
		fn = aopFns[i](fn)
	}

	return fn
}
