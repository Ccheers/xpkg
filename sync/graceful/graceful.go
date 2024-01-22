package graceful

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ccheers/xpkg/sync/routinepool"
)

var ErrGracefulExitTimeout = fmt.Errorf("graceful exit timeout")

type ILogger interface {
	Warn(ctx context.Context, message string)
}

type IGraceful interface {
	Add(ctx context.Context, exitings ...IExiting) error
	// Wait 等待终止进程的终止信号，并监听所有的 IExiting 退出之后才推出
	Wait(ctx context.Context) error
}

type IExiting interface {
	Name() string
	Stop(ctx context.Context) error
}

type stdLogger struct{}

func (x *stdLogger) Warn(ctx context.Context, message string) {
	fmt.Printf("[%s][WARN]: %s\n", time.Now().String(), message)
}

type Graceful struct {
	logger ILogger
	pool   routinepool.Pool

	mu       sync.Mutex
	exitings []IExiting

	wg sync.WaitGroup
}

type Options func(x *Graceful)

func WithLogger(logger ILogger) Options {
	return func(x *Graceful) {
		x.logger = logger
	}
}

func WithRoutinepool(pool routinepool.Pool) Options {
	return func(x *Graceful) {
		x.pool = pool
	}
}

func NewGraceful(opts ...Options) *Graceful {
	x := &Graceful{
		logger: &stdLogger{},
		pool:   routinepool.NewPool("graceful_exit", math.MaxInt32, routinepool.NewConfig()),
	}
	for _, opt := range opts {
		opt(x)
	}
	return x
}

func WaitSysExitSignal(x IGraceful, timeout time.Duration) error {
	sig := make(chan os.Signal, 1)
	// 监听系统信号
	// SIGINT: ctrl+c
	// SIGTERM: kill
	// SIGQUIT: ctrl+\
	// SIGKILL: kill -9
	// SIGUSR1: kill -USR1
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return x.Wait(ctx)
}

func (x *Graceful) Add(_ context.Context, exitings ...IExiting) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.exitings = append(x.exitings, exitings...)
	x.wg.Add(len(exitings))
	return nil
}

func (x *Graceful) Wait(ctx context.Context) error {
	var mm sync.Map
	// 等待所有的退出信号
	for _, exiting := range x.exitings {
		exiting := exiting
		mm.Store(exiting.Name(), struct{}{})
		_ = x.pool.CtxGo(ctx, func(ctx context.Context) {
			defer x.wg.Done()
			defer mm.Delete(exiting.Name())
			if err := exiting.Stop(ctx); err != nil {
				x.logger.Warn(ctx, fmt.Sprintf("[%s] stop error: %s", exiting.Name(), err.Error()))
			}
		})
	}

	ch := make(chan struct{}, 1)
	_ = x.pool.CtxGo(ctx, func(_ context.Context) {
		// 等待所有的退出信号
		x.wg.Wait()
		close(ch)
	})
	select {
	case <-ctx.Done():
		mm.Range(func(key, _ any) bool {
			name := key.(string)
			x.logger.Warn(ctx, fmt.Sprintf("[%s] stop timeout", name))
			return true
		})
		return ErrGracefulExitTimeout
	case <-ch:
	}
	return nil
}
