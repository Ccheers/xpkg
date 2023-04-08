package etcdx

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

const defaultSessionTTL = 60

type TargetNode struct {
	Key   string
	Value string
}

type WatchCallback interface {
	BatchSet([]TargetNode)
	BatchDelete([]string)
	Reset()
}

type WithWatchOptions func(*watchOptions)

// WithWatchSessionTTL 探活指针探活间隔（秒）
func WithWatchSessionTTL(ttl int) WithWatchOptions {
	return func(options *watchOptions) {
		options.sessionTTL = ttl
	}
}

// WithWatchPrefix 是否监听前缀
func WithWatchPrefix() WithWatchOptions {
	return func(options *watchOptions) {
		options.prefix = true
	}
}

type watchOptions struct {
	key        string
	sessionTTL int
	prefix     bool
}

func defaultWatchOptions() *watchOptions {
	return &watchOptions{
		sessionTTL: defaultSessionTTL,
	}
}

// Watch 监听指定 key 这是一个同步的函数，会阻塞直到 ctx 超时或者出错
func (x *ClientX) Watch(ctx context.Context, key string, callback WatchCallback, opts ...WithWatchOptions) error {
	options := defaultWatchOptions()
	for _, opt := range opts {
		opt(options)
	}
	options.key = key

	// 先探活 session 监听
	session, err := x.checkAlive(options.sessionTTL)
	if err != nil {
		return fmt.Errorf("[discovery] new session error: %s", err)
	}
	defer func() {
		err := session.Close()
		if err != nil {
			x.logger.Errorf("[discovery] session Close error: %s\n", err)
		}
	}()

	x.logger.Infof("[discovery] start batchwatch %s", options.key)
	revision, err := x.setAllKvs(ctx, options, callback)
	if err != nil {
		return fmt.Errorf("[discovery] client get error: %s", err)
	}

	// Network-partition aware health service(https://github.com/etcd-io/etcd/issues/8673)

	// watcher 开启
	watcher := clientv3.NewWatcher(x.Client)
	defer func() {
		watcher.Close()
	}()
	rch := x.getWatchChan(ctx, watcher, options, revision)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("[discovery] watch %s [finished]", options.key)
		case watchResp, ok := <-rch:
			if !ok {
				x.logger.Warnf("[discovery] watch %s [error]: channel close", options.key)
				revision, err = x.setAllKvs(ctx, options, callback)
				if err != nil {
					x.logger.Errorf("[discovery] client get error: %s", err)
					continue
				}
				rch = x.getWatchChan(ctx, watcher, options, revision)
				continue
			}
			err := x.handleBatchWatchResponse(watchResp, callback)
			if err != nil {
				x.logger.Errorf("[discovery] watch %s response error: %s ", options.key, err)
				continue
			}

			x.logger.Debugf("[discovery] watch %s response %+v", options.key, watchResp)
		case <-session.Done():
			x.logger.Warnf("[discovery] watch %s [error]: session close", options.key)

			_session, _watcher, _rch, err := x.batchReconnect(ctx, options, callback, watcher)
			if err != nil {
				x.logger.Errorf("[discovery][batchReconnect] err=%v", err)
				// 沉睡一段时间 防止大量的重连打垮 ETCD
				time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
				continue
			}

			// 确保一切正常以后，完成恢复
			watcher = _watcher
			rch = _rch
			session = _session
			x.logger.Infof("[discovery] watch %s reconnect success", options.key)
		}
	}
}

func (x *ClientX) batchReconnect(ctx context.Context, options *watchOptions, callback WatchCallback, watcher clientv3.Watcher) (*concurrency.Session, clientv3.Watcher, clientv3.WatchChan, error) {
	// 失去连接之后，尝试重启连接，重启探活
	_session, err := x.checkAlive(options.sessionTTL)
	if err != nil {
		return nil, nil, nil, err
	}

	// 探活成功 刷新 watch
	// 关闭之前的 watcher
	err = watcher.Close()
	if err != nil {
		x.logger.Warnf("[batchReconnect] watch %s [error]: watcher close error: %s", options.key, err)
	}

	// 重启 watcher
	revision, err := x.setAllKvs(ctx, options, callback)
	if err != nil {
		return nil, nil, nil, err
	}

	_watcher := clientv3.NewWatcher(x.Client)
	_rch := x.getWatchChan(ctx, watcher, options, revision)
	return _session, _watcher, _rch, nil
}

// getWatchChan 获取监听通道
func (x *ClientX) getWatchChan(ctx context.Context, watcher clientv3.Watcher, options *watchOptions, revision int64) clientv3.WatchChan {
	var opts []clientv3.OpOption
	opts = append(opts, clientv3.WithRev(revision))
	if options.prefix {
		opts = append(opts, clientv3.WithPrefix())
	}
	return watcher.Watch(clientv3.WithRequireLeader(ctx), options.key, opts...)
}

// handleBatchWatchResponse 处理监听事件
func (x *ClientX) handleBatchWatchResponse(watchResp clientv3.WatchResponse, callback WatchCallback) error {
	err := watchResp.Err()
	if err != nil {
		return err
	}

	createVal := make([]TargetNode, 0, len(watchResp.Events))
	modifyVal := make([]TargetNode, 0, len(watchResp.Events))
	deleteKeys := make([]string, 0, len(watchResp.Events))

	for _, ev := range watchResp.Events {
		node := TargetNode{
			Key:   string(ev.Kv.Key),
			Value: string(ev.Kv.Value),
		}
		if ev.IsCreate() {
			createVal = append(createVal, node)
		} else if ev.IsModify() {
			modifyVal = append(modifyVal, node)
		} else if ev.Type == mvccpb.DELETE {
			deleteKeys = append(deleteKeys, node.Key)
		} else {
			x.logger.Warnf("[discovery] no found watch type: %s %q", ev.Type, ev.Kv.Key)
		}
	}

	// create
	if len(createVal) > 0 {
		x.logger.Debugf("BatchCreate size:%v", len(createVal))
		callback.BatchSet(createVal)
	}
	// modify
	if len(modifyVal) > 0 {
		x.logger.Debugf("BatchModify size:%v", len(modifyVal))
		callback.BatchSet(modifyVal)
	}
	// delete
	if len(deleteKeys) > 0 {
		x.logger.Debugf("BatchDelete size:%v", len(deleteKeys))
		callback.BatchDelete(deleteKeys)
	}
	return nil
}

// checkAlive 心跳检测 探活
func (x *ClientX) checkAlive(ttl int) (*concurrency.Session, error) {
	return concurrency.NewSession(x.Client, concurrency.WithTTL(ttl))
}

// setAllKvs 初始化所有的值
func (x *ClientX) setAllKvs(ctx context.Context, options *watchOptions, callback WatchCallback) (revision int64, err error) {
	var opOpts []clientv3.OpOption
	if options.prefix {
		opOpts = append(opOpts, clientv3.WithPrefix())
	}

	resp, err := x.Get(ctx, options.key, opOpts...)
	if err != nil {
		return 0, err
	}

	// init set
	initVal := make([]TargetNode, len(resp.Kvs))
	for i, ev := range resp.Kvs {
		initVal[i].Key = string(ev.Key)
		initVal[i].Value = string(ev.Value)
	}
	x.logger.Debugf("setAllKvs key=%v initval size=%v", options.key, len(initVal))

	callback.Reset()
	callback.BatchSet(initVal)
	return resp.Header.Revision, nil
}
