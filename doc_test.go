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

func (state State) Waiters() []Waiter {
	return state.waiters
}

func (state State) FatalErr() error {
	return state.fatalErr
}
