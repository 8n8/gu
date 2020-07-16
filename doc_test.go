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
