package ratelimit

import (
	"context"
)

// Op operations type.
type Op int

const (
	// Success opertion type: success
	Success Op = iota
	// Ignore opertion type: ignore
	Ignore
	// Drop opertion type: drop
	Drop
)
// Allow  allow options.
type Allow struct{}

// AllowOption some option of Allow
type AllowOption interface {
	Apply(*Allow)
}

// DoneInfo done info.
type DoneInfo struct {
	Err error
	Op  Op
}

// DefaultAllowOpts returns the default allow options.
func DefaultAllowOpts() Allow {
	return Allow{}
}

// Limiter limit interface.
type Limiter interface {
	Allow(ctx context.Context, opts ...AllowOption) (func(info DoneInfo), error)
}
