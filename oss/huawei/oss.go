package huawei

// 引入依赖包
import (
	"context"
	"io"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

type Oss struct {
	client *obs.ObsClient
}

func NewOss(client *obs.ObsClient) *Oss {
	return &Oss{client: client}
}

func (h *Oss) Set(ctx context.Context, bucket string, key string, reader io.Reader) (err error) {
	input := &obs.PutObjectInput{}
	input.Bucket = bucket
	input.Key = key
	input.Body = reader
	_, err = h.client.PutObject(input)
	return
}

func (h *Oss) Get(ctx context.Context, bucket string, key string) (reader io.ReadCloser, err error) {
	input := &obs.GetObjectInput{}
	input.Bucket = bucket
	input.Key = key
	output, err := h.client.GetObject(input)
	if err != nil {
		return
	}

	return output.Body, nil
}
