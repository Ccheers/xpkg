package aliyun

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type Oss struct {
	client *oss.Client
}

func NewOss(client *oss.Client) *Oss {
	return &Oss{client: client}
}

func (o *Oss) Set(ctx context.Context, bucket string, key string, reader io.Reader) (err error) {
	bkt, err := o.client.Bucket(bucket)
	if err != nil {
		return
	}
	err = bkt.PutObject(key, reader)
	return
}

func (o *Oss) Get(ctx context.Context, bucket string, key string) (resp []byte, err error) {
	bkt, err := o.client.Bucket(bucket)
	if err != nil {
		return
	}
	body, err := bkt.GetObject(key)
	if err != nil {
		return
	}
	defer body.Close()

	resp, err = ioutil.ReadAll(body)
	return
}
