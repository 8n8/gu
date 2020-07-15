/*
Package gu is a general way of structuring concurrent Go programs
so that most of the logic is contained in pure functions, that is,
functions that don't contain any IO and are therefore easier to
test.

The main ideas are:

1. All the program state is kept in one struct.

2. IO functions contain as little logic as possible. Logic is
kept in non-IO functions so it can be tested easily. There is
a single channel for IO input. When a new message is put in the
channel, the main loop takes it out and uses it and a pure
function to update the global state and trigger new IO actions.

3. Sequential IO is handled by accumulating the status of a process
in the global state struct, and updating it when new IO
messages come in.
*/
package gu

// State is the global state of the program. So all of the state in
// a gu program is kept in a single struct.
type State interface {
	// Waiters gets the list of waiters from the state.
	Waiters() []Waiter
	// SetWaitable is for adding a new waiter to the state.
	SetWaitable(Waiter) State
	// FatalErr is used to signal that the program has encountered
	// an unrecoverable error. Setting this to a non-nil value
	// will cause the main loop to end and the program to crash.
	FatalErr() error
}

// In represents messages from the outside world, as a result of IO
// actions, for example from the file system or an HTTP server.
type In interface {
	// Router is used to decide which processes a new input should
	// be applied to.
	Router(Waiter) Ready
	// Update contains most of the logic of the program.  In a
	// functional programming language like Haskell, the State
	// input variable would not be altered, because the compiler
	// is efficient at making new copies of large structs when a
	// small part of it is changed.
	//
	// This function should not contain any IO code at all, not
	// even generating a random number or getting the system time.
	// This makes it easy to test.
	//
	// In Go, it is often necessary to modify the fields of the
	// State struct and then return it, rather than making a deep
	// copy.
	Update(State) (State, []Out)
}

// Out represents a message to the outside world. So it might
// an instruction to read a particular file, or put something into
// a channel, or read the system time, or any other IO action.
type Out interface {
	// Io is used to run IO actions, like reading files or running
	// an HTTP server. If this generates any messages that the
	// main loop needs to know about, these are send down the
	// chan provided in the argument.
	Io(chan In)
	// Fast determines if the IO action should be run in its own
	// goroutine or not. For example, reading the system time is
	// a quick thing to do and is not worth starting a goroutine for.
	Fast() bool
}

// Waiter represents a stage in a sequential process that is waiting
// for a message from the outside world to continue. For example
// it might be a message to be sent to the server, but it is waiting
// for the system time to be read before appending it to the message
// and sending it out to a server.
type Waiter interface {
	// Expected decides if an input message from the outside world
	// is expected by a waiter. If not, it returns false, nil. Each
	// new message that is received is tested against each waiter
	// to decide what to do with it.
	Expected(In) (bool, Ready)
}

// Ready is a combination of a waiter and the thing it has been
// waiting for, ready to be incorporated into the global state,
// and new actions to be generated.
type Ready interface {
	// Update contains program logic for updating the global state
	// of the program and generating new IO actions to do.
	Update(State) (State, []Out)
}

// Init has methods for initialising the state and the outputs.
// It should be implemented for a type alias of an empty struct.
type Init interface {
	// InitState returns the initial value of the global state struct.
	// It should be a pure function, that is, it should not do any
	// IO.
	InitState() State
	// InitOutputs returns all the initial IO actions. In general,
	// it will be a list of type aliases for empty structs, which
	// implement Out. For example it might contain a type alias
	// around an empty struct for initialising an HTTP server.
	InitOutputs() []Out
}

// Run is the main loop of the whole program. It initialises the
// global state and runs any initial IO actions. It then runs until
// it is told to crash on an unrecoverable error.
//
// On each pass of the loop it runs all the IO actions it has been
// told to, it reads in any new inputs from the outside world, and
// updates the global state.
func Run(init Init) error {
	state := init.InitState()
	outputs := init.InitOutputs()

	inChan := make(chan In, 1)

	for state.FatalErr() == nil {
		for _, output := range outputs {
			if output.Fast() {
				output.Io(inChan)
			} else {
				go output.Io(inChan)
			}
		}

		in := <-inChan

		state, outputs = update(state, in)
	}

	return state.FatalErr()
}

func update(state State, in In) (State, []Out) {
	for _, waiter := range state.Waiters() {
		relevant, ready := waiter.Expected(in)
		if relevant {
			return ready.Update(state)
		}
	}
	return in.Update(state)
}
