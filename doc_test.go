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

func (state State) ExampleWaiters() []Waiter {
	return state.waiters
}

func (state State) ExampleFatalErr() error {
	return state.fatalErr
}
