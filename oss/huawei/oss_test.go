package huawei

import (
	"context"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

var testClient *obs.ObsClient

var once = sync.Once{}

const (
	testBucket = "hj.test"
	testKey    = "haijun"
)

func InitTestProvider() {
	once.Do(func() {
		ak := "31THLB9DIBM3JF6JLTQA"
		sk := "BdN6yxwjnmFDN4i5rucYVoXZSQq7IAZf5aLJsrnE"
		endpoint := "https://obs.cn-east-3.myhuaweicloud.com"
		var err error
		testClient, err = obs.New(ak, sk, endpoint)
		if err != nil {
			panic(err)
		}
	})
}

func TestOss_Get(t *testing.T) {
	InitTestProvider()

	type fields struct {
		client *obs.ObsClient
	}
	type args struct {
		ctx    context.Context
		bucket string
		key    string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantResp []byte
		wantErr  bool
	}{
		{
			name: "1",
			fields: fields{
				client: testClient,
			},
			args: args{
				ctx:    context.Background(),
				bucket: testBucket,
				key:    testKey,
			},
			wantResp: []byte("test"),
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Oss{
				client: tt.fields.client,
			}
			gotResp, err := h.Get(tt.args.ctx, tt.args.bucket, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResp, tt.wantResp) {
				t.Errorf("Get() gotResp = %v, want %v", gotResp, tt.wantResp)
			}
		})
	}
}

func TestOss_Set(t *testing.T) {
	InitTestProvider()

	type fields struct {
		client *obs.ObsClient
	}
	type args struct {
		ctx    context.Context
		bucket string
		key    string
		reader io.Reader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "1",
			fields: fields{
				client: testClient,
			},
			args: args{
				ctx:    context.Background(),
				bucket: testBucket,
				key:    testKey,
				reader: strings.NewReader("test"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Oss{
				client: tt.fields.client,
			}
			if err := h.Set(tt.args.ctx, tt.args.bucket, tt.args.key, tt.args.reader); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
