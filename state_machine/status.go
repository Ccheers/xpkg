package state_machine

import (
	"errors"
	"fmt"
)

type State map[uint]State

// StateMachine 无限状态机
type StateMachine struct {
	stateMap map[uint]State
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		stateMap: make(map[uint]State),
	}
}

var ErrChangeState = errors.New("change state error")

func (x *StateMachine) ChangeState(from, to uint) error {
	if x.stateMap[from] == nil {
		return fmt.Errorf("%w: state(%d) is not defined", ErrChangeState, from)
	}
	if x.stateMap[from][to] == nil {
		return fmt.Errorf("%w: cant change state(%d) to state(%d)", ErrChangeState, from, to)
	}
	return nil
}

func (x *StateMachine) Register(from, to uint) {
	if x.stateMap[from] == nil {
		x.stateMap[from] = make(State)
	}
	if x.stateMap[to] == nil {
		x.stateMap[to] = make(State)
	}
	x.stateMap[from][to] = x.stateMap[to]
}
