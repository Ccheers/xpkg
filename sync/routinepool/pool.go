package routinepool

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

var (
	errPoolIsStopped = errors.New("routine pool is stopped")
)

const (
	poolStatusRunning = 1 // 线程池 活跃中
	poolStatusStopped = 0 // 线程池 已经被关闭
)

type Pool interface {
	// Name returns the corresponding pool name.
	Name() string
	// SetCap sets the goroutine capacity of the pool.
	SetCap(cap int32)
	// Go executes f.
	Go(f RoutineFunc) error
	// CtxGo executes f and accepts the context.
	CtxGo(ctx context.Context, f RoutineFunc) error
	// SetPanicHandler sets the panic handler.
	SetPanicHandler(f func(context.Context, error))
	// Stop the Pool graceful
	Stop(ctx context.Context) error
}

type RoutineFunc func(context.Context)

var taskPool sync.Pool

func init() {
	taskPool.New = newTask
}

type task struct {
	ctx context.Context
	f   func(context.Context)

	next *task
}

func (t *task) zero() {
	t.ctx = nil
	t.f = nil
	t.next = nil
}

func (t *task) Recycle() {
	t.zero()
	taskPool.Put(t)
}

func newTask() interface{} {
	return &task{}
}

type pool struct {
	// The name of the pool
	name string

	// capacity of the pool, the maximum number of goroutines that are actually working
	cap int32
	// Configuration information
	config *Config
	// linked list of tasks
	taskHead  *task
	taskTail  *task
	taskLock  sync.Mutex
	taskCount int32

	// Record the number of running workers
	workerCount int32

	// This method will be called when the worker panic
	panicHandler func(context.Context, error)

	// sign for the pool is stopped
	isStop uint32
}

// NewPool creates a new pool with the given name, cap and config.
func NewPool(name string, cap int32, config *Config) Pool {
	p := &pool{
		name:   name,
		cap:    cap,
		config: config,
		isStop: poolStatusRunning,
	}
	p.SetPanicHandler(func(ctx context.Context, err error) {
		log.Println(err.Error())
	})
	return p
}

func (p *pool) Name() string {
	return p.name
}

func (p *pool) SetCap(cap int32) {
	atomic.StoreInt32(&p.cap, cap)
}

func (p *pool) Go(f RoutineFunc) error {
	return p.CtxGo(context.Background(), f)
}

func (p *pool) CtxGo(ctx context.Context, f RoutineFunc) error {
	if p.isStopped() {
		return errPoolIsStopped
	}

	t := taskPool.Get().(*task)
	t.ctx = ctx
	t.f = f
	p.taskLock.Lock()
	if p.taskHead == nil {
		p.taskHead = t
		p.taskTail = t
	} else {
		p.taskTail.next = t
		p.taskTail = t
	}
	p.taskLock.Unlock()
	cnt := atomic.AddInt32(&p.taskCount, 1)
	_metricQueueSize.Set(float64(cnt), p.name)
	// The following two conditions are met:
	// 1. the number of tasks is greater than the threshold.
	// 2. The current number of workers is less than the upper limit p.cap.
	// or there are currently no workers.
	if (atomic.LoadInt32(&p.taskCount) >= p.config.ScaleThreshold && p.WorkerCount() < atomic.LoadInt32(&p.cap)) || p.WorkerCount() == 0 {
		p.incWorkerCount()
		w := workerPool.Get().(*worker)
		w.pool = p
		w.run()
	}
	return nil
}

// SetPanicHandler the func here will be called after the panic has been recovered.
func (p *pool) SetPanicHandler(f func(context.Context, error)) {
	p.panicHandler = f
}

func (p *pool) WorkerCount() int32 {
	return atomic.LoadInt32(&p.workerCount)
}

func (p *pool) Stop(ctx context.Context) error {
	p.setStopped()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if p.WorkerCount() == 0 {
				return nil
			}
			time.Sleep(time.Millisecond * 300)
		}
	}
}

func (p *pool) isStopped() bool {
	return atomic.LoadUint32(&p.isStop) == poolStatusStopped
}

func (p *pool) setStopped() {
	atomic.StoreUint32(&p.isStop, poolStatusStopped)
}

func (p *pool) incWorkerCount() {
	atomic.AddInt32(&p.workerCount, 1)
}

func (p *pool) decWorkerCount() {
	atomic.AddInt32(&p.workerCount, -1)
}
