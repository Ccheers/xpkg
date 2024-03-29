<!-- Code generated by gomarkdoc. DO NOT EDIT -->

# cmpx

```go
import "github.com/ccheers/xpkg/generic/cmpx"
```

## Index

- [func Clamp[T constraints.Ordered](x, lo, hi T) T](<#func-clamp>)
- [func Compare[T comparable](a, b T, less LessFn[T]) int](<#func-compare>)
- [func Equals[T comparable](a, b T) bool](<#func-equals>)
- [func Less[T constraints.Ordered](a, b T) bool](<#func-less>)
- [func Max[T constraints.Ordered](a, b T) T](<#func-max>)
- [func MaxFunc[T comparable](a, b T, less LessFn[T]) T](<#func-maxfunc>)
- [func Min[T constraints.Ordered](a, b T) T](<#func-min>)
- [func MinFunc[T comparable](a, b T, less LessFn[T]) T](<#func-minfunc>)
- [type EqualsFn](<#type-equalsfn>)
- [type HashFn](<#type-hashfn>)
- [type LessFn](<#type-lessfn>)


## func Clamp

```go
func Clamp[T constraints.Ordered](x, lo, hi T) T
```

Clamp returns x constrained within \[lo:hi\] range. If x compares less than lo, returns lo; otherwise if hi compares less than x, returns hi; otherwise returns v.

## func Compare

```go
func Compare[T comparable](a, b T, less LessFn[T]) int
```

Compare uses a less function to determine the ordering of 'a' and 'b'. It returns:

\* \-1 if a \< b

\* 1 if a \> b

\* 0 if a == b

## func Equals

```go
func Equals[T comparable](a, b T) bool
```

Equals wraps the '==' operator for comparable types.

## func Less

```go
func Less[T constraints.Ordered](a, b T) bool
```

Less wraps the '\<' operator for ordered types.

## func Max

```go
func Max[T constraints.Ordered](a, b T) T
```

Max returns the max of a and b.

## func MaxFunc

```go
func MaxFunc[T comparable](a, b T, less LessFn[T]) T
```

MaxFunc returns the max of a and b using the less func.

## func Min

```go
func Min[T constraints.Ordered](a, b T) T
```

Min returns the min of a and b.

## func MinFunc

```go
func MinFunc[T comparable](a, b T, less LessFn[T]) T
```

MinFunc returns the min of a and b using the less func.

## type EqualsFn

EqualsFn is a function that returns whether 'a' and 'b' are equal.

```go
type EqualsFn[T comparable] func(a, b T) bool
```

## type HashFn

HashFn is a function that returns the hash of 't'.

```go
type HashFn[T comparable] func(t T) uint64
```

## type LessFn

LessFn is a function that returns whether 'a' is less than 'b'.

```go
type LessFn[T comparable] func(a, b T) bool
```



Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)
