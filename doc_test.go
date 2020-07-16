package gu

import (
	"errors"
)

func ExampleState() {
	type state struct {
		waiters  []Waiter
		fatalErr error
	}
}

func (state State) ExampleState_Waiters() []Waiter {
	return state.waiters
}

func (state State) ExampleState_FatalErr() error {
	return state.fatalErr
}
