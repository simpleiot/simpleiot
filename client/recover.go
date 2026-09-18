package client

import (
	"fmt"
	"log"
	"runtime/debug"
	"time"
)

// clientRestartMax is the longest a crashed client waits before it is started
// again, and how long a client has to run before its crash count resets.
const clientRestartMax = 5 * time.Minute

// panicError is a panic that runRecovered turned into an error
type panicError struct {
	value any
}

func (e *panicError) Error() string {
	return fmt.Sprintf("panic: %v", e.value)
}

// runRecovered calls f and returns a panic in it as a *panicError, after
// logging the value and the stack.
//
// A panic in a client would otherwise stop the whole instance: NATS, the
// store, the API, and every other client. Several panics are reachable from a
// stored point, so the instance would stop again at every restart until the
// store was repaired. Every client Run, the manager loop, and every NATS
// callback the manager registers runs through here.
func runRecovered(what string, f func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("%v: recovered from panic: %v\n%s", what, r, debug.Stack())
			err = &panicError{value: r}
		}
	}()

	return f()
}
