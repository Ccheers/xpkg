<!-- Code generated by gomarkdoc. DO NOT EDIT -->

# state\_machine

```go
import "github.com/ccheers/xpkg/state_machine"
```

## Index

- [Variables](<#variables>)
- [type ErrorHandler](<#type-errorhandler>)
- [type Option](<#type-option>)
  - [func WithErrorHandler(handler ErrorHandler) Option](<#func-witherrorhandler>)
- [type StateMachine](<#type-statemachine>)
  - [func NewStateMachine(options ...Option) *StateMachine](<#func-newstatemachine>)
  - [func (x *StateMachine) ChangeState(from, to uint) error](<#func-statemachine-changestate>)
  - [func (x *StateMachine) Register(from, to *StateNode) error](<#func-statemachine-register>)
- [type StateNode](<#type-statenode>)
  - [func NewStateNode(state uint, desc string) *StateNode](<#func-newstatenode>)
  - [func (s *StateNode) String() string](<#func-statenode-string>)


## Variables

```go
var ErrChangeState = errors.New("change state error")
```

## type ErrorHandler

```go
type ErrorHandler func(from, to *StateNode) error
```

## type Option

```go
type Option func(*StateMachine)
```

### func WithErrorHandler

```go
func WithErrorHandler(handler ErrorHandler) Option
```

## type StateMachine

StateMachine 无限状态机

```go
type StateMachine struct {
    // contains filtered or unexported fields
}
```

### func NewStateMachine

```go
func NewStateMachine(options ...Option) *StateMachine
```

### func \(\*StateMachine\) ChangeState

```go
func (x *StateMachine) ChangeState(from, to uint) error
```

### func \(\*StateMachine\) Register

```go
func (x *StateMachine) Register(from, to *StateNode) error
```

## type StateNode

```go
type StateNode struct {
    // contains filtered or unexported fields
}
```

### func NewStateNode

```go
func NewStateNode(state uint, desc string) *StateNode
```

### func \(\*StateNode\) String

```go
func (s *StateNode) String() string
```



Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)
