package try_lock

import (
	"errors"
	"time"
)

type TryMutexLocker interface {
	Unlock()
	TryLock(duration time.Duration) error
}

type chMutex struct {
	ch chan struct{}
}

func NewChMutex() TryMutexLocker {
	return &chMutex{ch: make(chan struct{}, 1)}
}

var errGetLockTimeOut = errors.New("get lock timeout")

func (c *chMutex) TryLock(duration time.Duration) error {
	if duration > 0 {
		timeoutChan := time.After(duration)
		select {
		case <-timeoutChan:
			return errGetLockTimeOut
		case c.ch <- struct{}{}:
		}
	} else {
		select {
		case c.ch <- struct{}{}:
		default:
			return errGetLockTimeOut
		}
	}
	return nil
}

func (c *chMutex) Unlock() {
	<-c.ch
}
