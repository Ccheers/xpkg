package aliyun

import (
	"sync"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var testClient *oss.Client

var once = sync.Once{}

const (
	testBucket = ""
	testKey    = ""
)

func InitTestProvider() {
	once.Do(func() {
		ak := ""
		sk := ""
		endpoint := "http://oss-cn-hangzhou.aliyuncs.com"
		var err error
		testClient, err = oss.New(endpoint, ak, sk)
		if err != nil {
			panic(err)
		}
	})
}

func TestOss_Get(t *testing.T) {
	return

	//InitTestProvider()
	//
	//type fields struct {
	//	client *oss.Client
	//}
	//type args struct {
	//	ctx    context.Context
	//	bucket string
	//	key    string
	//}
	//tests := []struct {
	//	name     string
	//	fields   fields
	//	args     args
	//	wantResp []byte
	//	wantErr  bool
	//}{
	//	{
	//		name: "1",
	//		fields: fields{
	//			client: testClient,
	//		},
	//		args: args{
	//			ctx:    context.Background(),
	//			bucket: testBucket,
	//			key:    testKey,
	//		},
	//		wantResp: []byte("test"),
	//		wantErr:  false,
	//	},
	//}
	//for _, tt := range tests {
	//	t.Run(tt.name, func(t *testing.T) {
	//		o := &Oss{
	//			client: tt.fields.client,
	//		}
	//		gotResp, err := o.Get(tt.args.ctx, tt.args.bucket, tt.args.key)
	//		if (err != nil) != tt.wantErr {
	//			t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
	//			return
	//		}
	//		if !reflect.DeepEqual(gotResp, tt.wantResp) {
	//			t.Errorf("Get() gotResp = %v, want %v", gotResp, tt.wantResp)
	//		}
	//	})
	//}
}

func TestOss_Set(t *testing.T) {
	return
	//
	//InitTestProvider()
	//
	//type fields struct {
	//	client *oss.Client
	//}
	//type args struct {
	//	ctx    context.Context
	//	bucket string
	//	key    string
	//	reader io.Reader
	//}
	//tests := []struct {
	//	name    string
	//	fields  fields
	//	args    args
	//	wantErr bool
	//}{
	//	{
	//		name: "1",
	//		fields: fields{
	//			client: testClient,
	//		},
	//		args: args{
	//			ctx:    context.Background(),
	//			bucket: testBucket,
	//			key:    testKey,
	//			reader: strings.NewReader("test"),
	//		},
	//		wantErr: false,
	//	},
	//}
	//for _, tt := range tests {
	//	t.Run(tt.name, func(t *testing.T) {
	//		o := &Oss{
	//			client: tt.fields.client,
	//		}
	//		if err := o.Set(tt.args.ctx, tt.args.bucket, tt.args.key, tt.args.reader); (err != nil) != tt.wantErr {
	//			t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
	//		}
	//	})
	//}
}
