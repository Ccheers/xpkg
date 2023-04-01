package try_lock

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"

	uuid "github.com/satori/go.uuid"
)

type distributedTryLocker struct {
	cmd CASCommand

	key   string
	value string

	option *options
}

type CASCommand interface {
	CAS(key, src, dst string) bool
}

type Option func(*options)

func WithScheduleFunc(f func()) Option {
	return func(o *options) {
		o.schedule = f
	}
}

type options struct {
	schedule Schedule
}

type Schedule func()

func defaultSchedule() {
	time.Sleep(time.Millisecond * time.Duration((rand.Intn(20)+1)*10))
	runtime.Gosched()
}

func NewDistributedTryLocker(cmd CASCommand, key, value string, opts ...Option) TryMutexLocker {
	option := &options{
		schedule: defaultSchedule,
	}
	for _, opt := range opts {
		opt(option)
	}
	return &distributedTryLocker{cmd: cmd, key: key, value: value, option: option}
}

func (r *distributedTryLocker) Unlock() {
	r.cmd.CAS(r.key, r.value, "")
}

func (r *distributedTryLocker) TryLock(duration time.Duration) error {
	if r.cmd.CAS(r.key, "", r.value) {
		return nil
	}
	if duration > 0 {
		timeoutChan := time.After(duration)
		for {
			select {
			case <-timeoutChan:
				return fmt.Errorf("%w: key=%s value=%s", errGetLockTimeOut, r.key, r.value)
			default:
				if r.cmd.CAS(r.key, "", r.value) {
					return nil
				}
				// 执行一次切换调度
				r.option.schedule()
			}
		}
	}
	return fmt.Errorf("%w: key=%s value=%s", errGetLockTimeOut, r.key, r.value)
}

func SimpleDistributedTryLock(command CASCommand, key string, duration time.Duration, opts ...Option) (func(), error) {
	locker := NewDistributedTryLocker(command, key, uuid.NewV1().String(), opts...)
	err := locker.TryLock(duration)
	if err != nil {
		return nil, err
	}
	return func() {
		locker.Unlock()
	}, nil
}
