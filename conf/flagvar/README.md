<!-- Code generated by gomarkdoc. DO NOT EDIT -->

# flagvar

```go
import "github.com/ccheers/xpkg/conf/flagvar"
```

## Index

- [type StringVars](<#type-stringvars>)
  - [func (s *StringVars) Set(val string) error](<#func-stringvars-set>)
  - [func (s StringVars) String() string](<#func-stringvars-string>)


## type StringVars

StringVars \[\]string implement flag.Value

```go
type StringVars []string
```

### func \(\*StringVars\) Set

```go
func (s *StringVars) Set(val string) error
```

Set implement flag.Value

### func \(StringVars\) String

```go
func (s StringVars) String() string
```



Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)
