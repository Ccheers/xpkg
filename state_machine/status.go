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

func NewStateNode(state uint, desc string) *StateNode {
	return &StateNode{state: state, desc: desc, next: make(map[uint]*StateNode)}
}

// StateMachine 无限状态机
type StateMachine struct {
	stateMap map[uint]*StateNode
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		stateMap: make(map[uint]*StateNode),
	}
}

var ErrChangeState = errors.New("change state error")

func (x *StateMachine) ChangeState(from, to uint) error {
	if _, ok := x.stateMap[from]; !ok {
		return fmt.Errorf("state %d not exist", from)
	}
	_from := x.stateMap[from]
	if _, ok := x.stateMap[to]; !ok {
		return fmt.Errorf("state %d not exist", to)
	}
	_to := x.stateMap[to]
	if _from == nil {
		return fmt.Errorf("%w: state(%d) is not defined", ErrChangeState, from)
	}
	if _, ok := _from.next[to]; !ok {
		return fmt.Errorf("%w: cant change state(%s) to state(%s)", ErrChangeState, _from.desc, _to.desc)
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
