/*
Package gu is a general way of structuring concurrent Go programs
so that most of the logic is contained in pure functions, that is,
functions that don't contain any IO and are easier to test.

The main ideas are:

1. All the program state is kept in one struct.

2. IO functions contain as little logic as possible. Logic is kept
in non-IO functions so it can be tested easily. There is a single
channel for all IO input. When a new message is put in the channel,
the main loop takes it out and uses it and a pure function to update
the global state and trigger new IO actions.

3. Sequential IO is handled by accumulating information in the global
state struct, and updating it when new IO messages come in.
*/
package gu

// State is the global state of the program. So all of the state in
// a gu program is kept in one place.
type State interface {
	// Waiters gets the list of waiters from the state.
	Waiters() []Waiter

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

	// Update contains the logic for updating the state when a new
	// message comes in from the outside world and doesn't affect
	// one of the Waiters. So it shouldn't be used when the message
	// needs to trigger a whole sequence of IO actions that depend
	// on each other. In that case create a Waiter.
	//
	// Each new message that comes in is first offered to the
	// Waiters. If none of them want it then it is processed using
	// this Update function.
	//
	// This function should not contain any IO code at all, not
	// even generating a random number or getting the system time.
	// This makes it easy to test.
	Update(State) (State, []Out)
}

// Out represents a message to the outside world. So it might
// an instruction to read a particular file, or put something into
// a channel, or read the system time, or any other IO action.
type Out interface {
	// Io is used to run IO actions, like reading files or running
	// an HTTP server. If this generates any messages that the
	// main loop needs to know about, these are sent down the
	// chan provided in the argument.
	//
	// Try to make Io implementations as short as they can possibly
	// be and let the pure functions do the logic.
	Io(chan In)

	// Fast determines if the IO action should be run in its own
	// goroutine or not. For example, reading the system time is
	// a quick thing to do and is not worth starting a goroutine for.
	Fast() bool
}

// Waiter represents a stage in a sequential process that is waiting
// for a message from the outside world in order to continue. For
// example, a Waiter might contain the handle of a large file that
// is being read in chunks, with each chunk being processed and sent
// to the server.
type Waiter interface {
	// Expected decides if an input message from the outside world
	// is expected by a waiter. If not, it returns nil, false. Each
	// new message that is received is tested against each waiter
	// to decide what to do with it.
	//
	// So each Expected implementation will contain a type
	// assertion on the In value to see if it is what the Waiter is
	// waiting for.
	Expected(In) (Ready, bool)
}

// Ready is a combination of a waiter and the thing it has been
// waiting for, ready to be processed by the pure Update function.
type Ready interface {
	// Update contains program logic for updating the global state
	// of the program and generating new IO actions to do.
	//
	// This is where all of the logic for sequential processes
	// should be kept.
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
// told to, reads in any new inputs from the outside world, and
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
		ready, relevant := waiter.Expected(in)
		if relevant {
			return ready.Update(state)
		}
	}
	return in.Update(state)
}
