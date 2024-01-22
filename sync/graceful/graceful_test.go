package graceful

import (
	"context"
	"log"
	"math/rand"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestWaitSysExitSignal(t *testing.T) {
	x := NewGraceful()
	_max := time.Duration(0)
	for i := 0; i < 100; i++ {
		randTimeout := time.Duration(rand.Int31n(1000)+200) * time.Millisecond
		_ = x.Add(context.Background(), &dummyExit{
			name:    strconv.Itoa(i),
			timeout: randTimeout,
		})
		if randTimeout > _max {
			_max = randTimeout
		}
	}

	go func() {
		// 在另一个goroutine中模拟信号发送
		time.Sleep(5 * time.Second)
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			log.Fatalf("Failed to find process: %v", err)
		}

		// 向当前进程发送 SIGINT 信号
		if err := p.Signal(syscall.SIGINT); err != nil {
			log.Fatalf("Failed to send signal: %v", err)
		}
	}()

	err := WaitSysExitSignal(x, time.Second)
	if err == nil && _max < time.Second {
		panic("want error")
	}
	t.Logf("err: %v", err)
}

type dummyExit struct {
	name    string
	timeout time.Duration
}

func (d *dummyExit) Name() string {
	return d.name
}

func (d *dummyExit) Stop(ctx context.Context) error {
	time.Sleep(d.timeout)
	return nil
}
