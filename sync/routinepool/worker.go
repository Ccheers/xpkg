package routinepool

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
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
				cnt := atomic.AddInt32(&w.pool.taskCount, -1)
				_metricQueueSize.Set(float64(cnt), w.pool.name)
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
				w.pool.panicHandler(t.ctx, fmt.Errorf("[routinepool] task cancel: %s error: %w", w.pool.name, t.ctx.Err()))
				t.Recycle()
				continue
			default:
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						if w.pool.panicHandler != nil {
							w.pool.panicHandler(t.ctx, fmt.Errorf("[routinepool] panic in pool: %s: %v: %s", w.pool.name, r, debug.Stack()))
						}
					}
				}()
				t.f(t.ctx)
				_metricCount.Inc(w.pool.name)
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
