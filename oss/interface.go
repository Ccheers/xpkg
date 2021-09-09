package oss

import (
	"context"
	"errors"
	"github.com/ccheers/xpkg/oss/huawei"
	"io"

	"github.com/ccheers/xpkg/oss/aliyun"

	aliyunoss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	huaweiyunoss "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

//Oss 抽象接口
type Oss interface {
	Set(ctx context.Context, bucket string, key string, reader io.Reader) (err error)
	Get(ctx context.Context, bucket string, key string) (resp []byte, err error)
}

// Type Oss 类型
type Type string

const (
	//OssTypeHuaweiYun Oss 类型是华为云
	OssTypeHuaweiYun Type = "huaweiyun"
	//OssTypeAliYun Oss 类型是阿里云
	OssTypeAliYun Type = "aliyun"
)

var (
	errNoMatchType = errors.New("no match oss type")
)

//Config Oss 配置
type Config struct {
	AccessKey       string
	AccessKeySecret string
	EndPoint        string
}

//Factory 工厂函数
func Factory(tp Type, config Config) (Oss, error) {
	switch tp {
	case OssTypeHuaweiYun:
		return getHuaweiyunOss(config)
	case OssTypeAliYun:
		return getAliyunOss(config)
	}
	return nil, errNoMatchType
}

func getAliyunOss(config Config) (*aliyun.Oss, error) {
	client, err := aliyunoss.New(config.EndPoint, config.AccessKey, config.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	return aliyun.NewOss(client), nil
}
func getHuaweiyunOss(config Config) (*huawei.Oss, error) {
	client, err := huaweiyunoss.New(config.AccessKey, config.AccessKeySecret, config.EndPoint)
	if err != nil {
		return nil, err
	}
	return huawei.NewOss(client), nil
}

//IsErrNoMatchType 错误断言 判断错误是否是： 未匹配 oss 错误
func IsErrNoMatchType(err error) bool {
	return errors.Is(err, errNoMatchType)
}
