package xmsgbus

import (
	"context"
	"encoding/json"
)

type EncodeFunc[T ITopic] func(ctx context.Context, dst T) ([]byte, error)

type DecodeFunc[T ITopic] func(ctx context.Context, bs []byte) (T, error)

func DefaultEncodeFunc[T ITopic](ctx context.Context, dst T) ([]byte, error) {
	return json.Marshal(dst)
}

func DefaultDecodeFunc[T ITopic](ctx context.Context, bs []byte) (T, error) {
	var dst T
	err := json.Unmarshal(bs, &dst)
	return dst, err
}
