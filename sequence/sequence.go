/*
Package sequence implements an interface for building and running
many computations in one object.

Sequences of computations are built using the builder pattern with
function arguments.
The functions are then run with a caller supplied status object
used to track the status of the computation.
If any function returns an error value, a failure is recorded in
the status, any already-running units of computation are
completed, and no more are run.

The status object may be used by the caller to pause the sequence,
continue a paused sequence, or trigger an early failure.
In any of these cases, already-running functions will be completed,
but no new functions in the sequence will be called.
*/
package sequence

import (
	"bitbucket.org/nedp/command/status"
)

// An interface for running many computations 
// contained in a single object.
type RunAller interface {
	RunAll(status.Interface) status.Interface
	IsRunning() bool
}

// An object containing a series of computations to
// be performed sequentially.
type Sequence struct {
	isRunning chan bool // buffered
	sequence
}

type sequence struct {
	phases []runAller
}

// Runs all computations in the sequence.
//
// If the sequence is already running concurrently, this function blocks
// until the other run finishes.
//
// Returns
// `true`  if all computations ran successfully,
// `false` if there was a failure.
func (seq Sequence) RunAll(stat status.Interface) status.Interface {
	seq.isRunning <- true
	defer func(){
		<-seq.isRunning
	}()
	return seq.runAll(stat)
}

func (seq sequence) runAll(stat status.Interface) status.Interface {
	// Run each phase with the same status.
	for _, phase := range seq.phases {
		// If there is a failure, stop running phases.
		stat = phase.runAll(stat)
		if stat.HasFailed() {
			break
		}
	}
	return stat
}

// Returns
// whether the sequence is currently being run in another goroutine.
func (seq Sequence) IsRunning() bool {
	select {
	case seq.isRunning <- false:
		<-seq.isRunning
		return false
	case isRunning := <-seq.isRunning:
		seq.isRunning <- isRunning
		return isRunning
	}
}
