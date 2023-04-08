package etcdx

import (
	"context"
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	defaultTTL = 10
)

type WithLeaseOptions func(*leaseOptions)

func WithLeaseTTL(ttl int64) WithLeaseOptions {
	return func(options *leaseOptions) {
		options.ttl = ttl
	}
}

type leaseOptions struct {
	ttl int64
}

func defaultOptions() *leaseOptions {
	return &leaseOptions{
		ttl: defaultTTL,
	}
}

func (x *ClientX) TryLease(ctx context.Context, key string, value string, opts ...WithLeaseOptions) (func(context.Context), error) {
	var err error
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	// 得到kv和lease的API子集
	lease := clientv3.NewLease(x.Client)
	defer func() {
		if err != nil {
			err := lease.Close()
			if err != nil {
				x.logger.Errorf("lease close error: %v", err)
			}
		}
	}()

	// 1, 创建租约
	leaseGrantResp, err := lease.Grant(ctx, options.ttl)
	if err != nil {
		return nil, err
	}

	// 租约ID
	leaseId := leaseGrantResp.ID

	// 2, 自动续租
	keepRespChan, err := lease.KeepAlive(ctx, leaseId)
	if err != nil {
		return nil, err
	}

	// 3, 处理续租应答的协程
	go func() {
		for {
			select {
			case keepResp := <-keepRespChan: // 自动续租的应答
				if keepResp == nil {
					return
				}
			}
		}
	}()

	// 4, 创建事务txn
	txn := x.Txn(ctx)

	// 5, 事务抢锁
	// etcdv3新引入的多键条件事务，替代了v2中Compare-And-put操作。
	// etcdv3的多键条件事务的语意是先做一个比较（compare）操作，如果比较成立则执行一系列操作，如果比较不成立则执行另外一系列操作。
	// 有类似于C语言中的条件表达式。接下来的这部分实现了如果不存在这个key，则将这个key写入到etcd，如果存在则读取这个key的值这样的功能。
	// 下面这一句，是构建了一个compare的条件，比较的是key的createRevision，如果revision是0，则存入一个key，如果revision不为0，则读取这个key。
	// revision是etcd一个全局的序列号，每一个对etcd存储进行改动都会分配一个这个序号，在v2中叫index，
	// createRevision是表示这个key创建时被分配的这个序号。当key不存在时，createRivision是0。
	txn.If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, value, clientv3.WithLease(leaseId)))

	// 提交事务
	txnResp, err := txn.Commit()
	if err != nil {
		return nil, err
	}

	// 6, 成功返回
	if txnResp.Succeeded { // 锁被占用
		return func(ctx context.Context) {
			_, err := lease.Revoke(ctx, leaseId)
			if err != nil {
				x.logger.Errorf("lease revoke error: %v", err)
			}
			err = lease.Close()
			if err != nil {
				x.logger.Errorf("lease close error: %v", err)
			}
		}, nil
	}
	// 这一步是为了 上面的 lease defer
	err = fmt.Errorf("[lease] txn failed")
	return nil, err
}
