package xlogger

import (
	"context"
	"testing"
)

func TestInfo(_ *testing.T) {
	logger := DefaultLogger
	logger = With(logger, "ts", DefaultTimestamp)
	logger = With(logger, "caller", DefaultCaller)
	_ = logger.Log(LevelInfo, "key1", "value1")
}

func TestWithContext(_ *testing.T) {
	WithContext(context.Background(), nil)
}
