/*
Package status implements a threadsafe status manager.

Intended for use by the sequence package.
Under the hood it combines and wraps functionality from the `sync` package.

Supported actions are:
 * Pausing, continuing, and failing
 * Registering the beginning and end of concurrent tasks
 * Acquiring a read lock after waiting for readiness
 * Releasing the read lock
 * Creating a 'bound copy', which is a new object with:
    - a separate set of bound tasks
    - pause/continue/failure and read-lock state bound to the original's
*/
package status

import (
	"errors"
	"sync"
)

// Interface for managing and refering to a status.
type Interface interface {
	ReadyRLock() bool
	RUnlock()

	Pause() (bool, error)
	Cont() (bool, error)
	Fail() error
	HasFailed() bool

	Add(int)
	Done()

	BoundCopy() Interface
}

// The underlying type for Interface objects returned by this package.
type Status struct {
	sync.WaitGroup

	*state // Propogates
}

type state struct {
	// rw is the same mutex as the contents of the sync.Cond member.
	rw *sync.RWMutex
	sync.Cond

	isPaused  bool
	hasFailed bool
}

// Creates a new status.
//
// A new status object is unlocked, not paused, not waiting,
// and has not failed.
//
// Returns:
//  The new status.
func New() Interface {
	rw := new(sync.RWMutex)
	s := &Status{
		sync.WaitGroup{},
		&state{
			rw,
			*sync.NewCond(rw.RLocker()),
			false,
			false,
		},
	}
	return s
}

// Creates a 'bound copy' of the receiver after acquiring a read lock.
// The read lock will be released before returning.
//
// The bound copy's read lock, pause/continue, and failure statuses
// will be bound to the original's.
// The bound copy's done/add counter will not be bound to the originals;
// it will be set to zero (not waiting).
//
// Returns:
//  The bound copy
func (s *Status) BoundCopy() Interface {
	// RLock
	s.state.L.Lock()
	defer s.state.L.Unlock()

	return &Status{
		sync.WaitGroup{},
		s.state,
	}
}

// Acquires a read lock on the status when it is ready.
//
// 'Ready' means it is not paused, has not failed, and is not waiting.
// RUnlock() should be called to relinquish the read-lock.
//
// Returns:
//   `true` if the RLock is acquired.
//  `false` if a failure has occured.
func (s *Status) ReadyRLock() bool {
	// 1. RLock
	s.state.L.Lock()

	// 2. Check for an early exit during the RLock
	if s.state.hasFailed {
		s.state.L.Unlock()
		return false
	}
	// 3. Wait until waitgroup is done during the RLock
	s.Wait()
	for s.state.isPaused {
		// 4. Wait until unpaused, acquiring L.Lock
		s.state.Wait()

		// 2. Check for an early exit during the RLock.
		if s.state.hasFailed {
			s.state.L.Unlock()
			return false
		}
		// 3. Wait until waitgroup is done during the RLock.
		s.Wait()
	} // 5. Reconfirm the unpaused + no failure in the loop condition.
	return true
}

// Wrapper function for `sync.RWMutex.Unlock()`
func (s *Status) RUnlock() {
	s.state.L.Unlock()
}

// Records a failure.
//
// This cannot be undone.
//
// Returns:
//     `nil` if no failure had yet been recorded.
//  an error if a failure was already recorded.
func (s *Status) Fail() error {
	// Write lock
	s.state.rw.Lock()
	defer s.state.rw.Unlock()

	if s.state.hasFailed {
		return errors.New("A failure already occured.")
	}
	s.state.hasFailed = true
	s.state.Broadcast()
	return nil
}

// Whether a failure has been recorded on this status object.
//
// Blocks until a read lock is acquired.
//
// Returns:
//  `true`  if a failure has been recorded.
//  `false` if a no failure has been recorded.
func (s *Status) HasFailed() bool {
	s.state.rw.RLock()
	defer s.state.rw.RUnlock()
	return s.state.hasFailed
}

// Records a pause, undone by calling `Pause`.
//
// Blocks until a write lock is acquired.
//
// Returns:
//  (unspecified, an error) if there has been a failure.
//       (`true`, `nil`)    if the status was already paused.
//      (`false`, `nil`)    if the status was continuing.
func (s *Status) Pause() (bool, error) {
	// Write lock
	s.state.rw.Lock()
	defer s.state.rw.Unlock()

	if s.state.hasFailed {
		return true, errors.New("Execution is stopped, but because a failure has occured.")
	}
	isPaused := s.state.isPaused
	s.state.isPaused = true
	s.state.Broadcast()
	return isPaused, nil
}

// Records a continuation, undoing a call to `Pause`.
//
// Blocks until a write lock is acquired.
//
// Returns:
//  (unspecified, an error) if there has been a failure.
//       (`true`, `nil`)    if the status was continuing.
//      (`false`, `nil`)    if the status was already paused.
func (s *Status) Cont() (bool, error) {
	// Write lock
	s.state.rw.Lock()
	defer s.state.rw.Unlock()

	if s.state.hasFailed {
		return false, errors.New("Couldn't continue; a failure has occured.")
	}
	isPaused := s.state.isPaused
	s.state.isPaused = false
	s.state.Broadcast()
	return !isPaused, nil
}

// Wrapper function for `sync.WaitGroup.Add(delta)`
func (s *Status) Add(delta int) {
	s.WaitGroup.Add(delta)
}

// Wrapper function for `sync.WaitGroup.Done()`
func (s *Status) Done() {
	s.WaitGroup.Done()
}
