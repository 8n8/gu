package gu

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
