package etcdx

import (
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ClientX struct {
	*clientv3.Client
	logger *clientLogger
}

func NewClientX(client *clientv3.Client, logger Logger) *ClientX {
	return &ClientX{Client: client, logger: &clientLogger{logger: logger}}
}

type LogLevel string

const (
	LogLevelERROR LogLevel = "ERROR"
	LogLevelINFO  LogLevel = "INFO"
	LogLevelWARN  LogLevel = "WARN"
	LogLevelDEBUG LogLevel = "DEBUG"
)

type Logger interface {
	Logf(level LogLevel, template string, args ...interface{})
	Logw(level LogLevel, keyPairs ...interface{})
}

type clientLogger struct {
	logger Logger
}

func (x *clientLogger) Errorf(template string, args ...interface{}) {
	x.logger.Logf(LogLevelERROR, template, args...)
}

func (x *clientLogger) Infof(template string, args ...interface{}) {
	x.logger.Logf(LogLevelINFO, template, args...)
}

func (x *clientLogger) Warnf(template string, args ...interface{}) {
	x.logger.Logf(LogLevelWARN, template, args...)
}

func (x *clientLogger) Debugf(template string, args ...interface{}) {
	x.logger.Logf(LogLevelDEBUG, template, args...)
}

func (x *clientLogger) Errorw(kvs ...interface{}) {
	x.logger.Logw(LogLevelERROR, kvs...)
}

func (x *clientLogger) Infow(kvs ...interface{}) {
	x.logger.Logw(LogLevelINFO, kvs...)
}

func (x *clientLogger) Warnw(kvs ...interface{}) {
	x.logger.Logw(LogLevelWARN, kvs...)
}

func (x *clientLogger) Debugw(kvs ...interface{}) {
	x.logger.Logw(LogLevelDEBUG, kvs...)
}
