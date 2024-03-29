<!-- Code generated by gomarkdoc. DO NOT EDIT -->

# cpu

```go
import "github.com/ccheers/xpkg/stat/sys/cpu"
```

## Index

- [Variables](<#variables>)
- [func ParseUintList(val string) (map[int]bool, error)](<#func-parseuintlist>)
- [func ReadStat(stat *Stat)](<#func-readstat>)
- [type CPU](<#type-cpu>)
- [type Info](<#type-info>)
  - [func GetInfo() Info](<#func-getinfo>)
- [type Stat](<#type-stat>)


## Variables

ErrNoCFSLimit is no quota limit

```go
var ErrNoCFSLimit = errors.Errorf("no quota limit")
```

## func ParseUintList

```go
func ParseUintList(val string) (map[int]bool, error)
```

ParseUintList parses and validates the specified string as the value found in some cgroup file \(e.g. cpuset.cpus, cpuset.mems\), which could be one of the formats below. Note that duplicates are actually allowed in the input string. It returns a map\[int\]bool with available elements from val set to true. Supported formats: 7 1\-6 0,3\-4,7,8\-10 0\-0,0,1\-7 03,1\-3 \<\- this is gonna get parsed as \[1,2,3\] 3,2,1 0\-2,3,1

## func ReadStat

```go
func ReadStat(stat *Stat)
```

ReadStat read cpu stat.

## type CPU

CPU is cpu stat usage.

```go
type CPU interface {
    Usage() (u uint64, e error)
    Info() Info
}
```

## type Info

Info cpu info.

```go
type Info struct {
    Frequency uint64
    Quota     float64
}
```

### func GetInfo

```go
func GetInfo() Info
```

GetInfo get cpu info.

## type Stat

Stat cpu stat.

```go
type Stat struct {
    Usage uint64 // cpu use ratio.
}
```



Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)
