package cmpx

import (
	"golang.org/x/exp/constraints"
)

// EqualsFn is a function that returns whether 'a' and 'b' are equal.
type EqualsFn[T comparable] func(a, b T) bool

// LessFn is a function that returns whether 'a' is less than 'b'.
type LessFn[T comparable] func(a, b T) bool

// HashFn is a function that returns the hash of 't'.
type HashFn[T comparable] func(t T) uint64

// Equals wraps the '==' operator for comparable types.
func Equals[T comparable](a, b T) bool {
	return a == b
}

// Less wraps the '<' operator for ordered types.
func Less[T constraints.Ordered](a, b T) bool {
	return a < b
}

// Compare uses a less function to determine the ordering of 'a' and 'b'. It returns:
//
// * -1 if a < b
//
// * 1 if a > b
//
// * 0 if a == b
func Compare[T comparable](a, b T, less LessFn[T]) int {
	if less(a, b) {
		return -1
	} else if less(b, a) {
		return 1
	}
	return 0
}

// Max returns the max of a and b.
func Max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Min returns the min of a and b.
func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Clamp returns x constrained within [lo:hi] range.
// If x compares less than lo, returns lo; otherwise if hi compares less than x, returns hi; otherwise returns v.
func Clamp[T constraints.Ordered](x, lo, hi T) T {
	return Max(lo, Min(hi, x))
}

// MaxFunc returns the max of a and b using the less func.
func MaxFunc[T comparable](a, b T, less LessFn[T]) T {
	if less(b, a) {
		return a
	}
	return b
}

// MinFunc returns the min of a and b using the less func.
func MinFunc[T comparable](a, b T, less LessFn[T]) T {
	if less(a, b) {
		return a
	}
	return b
}
