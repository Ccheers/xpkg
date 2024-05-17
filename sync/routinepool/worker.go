package routinepool

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var workerPool sync.Pool

func init() {
	workerPool.New = newWorker
}

type worker struct {
	pool *pool
}

func newWorker() interface{} {
	return &worker{}
}

func (w *worker) run() {
	go func() {
		for {
			var t *task
			w.pool.taskLock.Lock()
			if w.pool.taskHead != nil {
				t = w.pool.taskHead
				w.pool.taskHead = w.pool.taskHead.next
				atomic.AddInt32(&w.pool.taskCount, -1)
			}
			if t == nil {
				// if there's no task to do, exit
				w.close()
				w.pool.taskLock.Unlock()
				w.Recycle()
				return
			}
			w.pool.taskLock.Unlock()

			// check context before doing task
			select {
			case <-t.ctx.Done():
				if w.pool.config.errorHandler != nil {
					w.pool.config.errorHandler(t.ctx, fmt.Errorf("[routinepool] task cancel: %s error: %w", w.pool.name, t.ctx.Err()))
				}
				t.Recycle()
				continue
			default:
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						if w.pool.config.panicHandler != nil {
							w.pool.config.panicHandler(t.ctx, fmt.Errorf("[routinepool] panic in pool: %s: %v: %s", w.pool.name, r, debug.Stack()))
						}
					}
				}()
				t.f(t.ctx)
				w.pool.taskCounter.Add(t.ctx, 1, metric.WithAttributes(attribute.String("pool_name", w.pool.name)))
			}()

			t.Recycle()
		}
	}()
}

func (w *worker) close() {
	w.pool.decWorkerCount()
}

func (w *worker) zero() {
	w.pool = nil
}

func (w *worker) Recycle() {
	w.zero()
	workerPool.Put(w)
}
