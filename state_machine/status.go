package state_machine

import (
	"errors"
	"fmt"
)

type StateNode struct {
	state uint
	desc  string
	next  map[uint]*StateNode
}

func (s *StateNode) String() string {
	return s.desc
}

func NewStateNode(state uint, desc string) *StateNode {
	return &StateNode{state: state, desc: desc, next: make(map[uint]*StateNode)}
}

type ErrorHandler func(from, to *StateNode) error

type Option func(*StateMachine)

func WithErrorHandler(handler ErrorHandler) Option {
	return func(sm *StateMachine) {
		sm.errorHandler = handler
	}
}

// StateMachine 无限状态机
type StateMachine struct {
	stateMap     map[uint]*StateNode
	errorHandler ErrorHandler
}

func NewStateMachine(options ...Option) *StateMachine {
	machine := &StateMachine{
		stateMap: make(map[uint]*StateNode),
		errorHandler: func(from, to *StateNode) error {
			return fmt.Errorf("%w: %s -> %s", ErrChangeState, from.desc, to.desc)
		},
	}

	for _, option := range options {
		option(machine)
	}
	return machine
}

var ErrChangeState = errors.New("change state error")

func (x *StateMachine) ChangeState(from, to uint) error {
	_from := x.stateMap[from]
	_to := x.stateMap[to]
	if _from == nil {
		return fmt.Errorf("%w: state(%d) is not defined", ErrChangeState, from)
	}
	if _to == nil {
		return fmt.Errorf("%w: state(%d) is not defined", ErrChangeState, to)
	}
	if _, ok := _from.next[to]; !ok {
		return x.errorHandler(_from, _to)
	}
	return nil
}

func (x *StateMachine) Register(from, to *StateNode) error {
	if from == nil || to == nil {
		return fmt.Errorf("from or to is nil")
	}
	if x.stateMap[from.state] == nil {
		x.stateMap[from.state] = from
	}
	if x.stateMap[to.state] == nil {
		x.stateMap[to.state] = to
	}
	x.stateMap[from.state].next[to.state] = x.stateMap[to.state]
	return nil
}
