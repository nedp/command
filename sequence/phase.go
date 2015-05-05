package sequence


import (
	"github.com/nedp/command/status"
)

type runAller interface {
	runAll(status status.Interface) status.Interface
}

type phase struct {
	sequences []runAller
	main func() error
}

func (ph phase) runAll(stat status.Interface) status.Interface {
	// Wait for previous computations to end before starting new ones.
	// Don't allow status access during the setup period
	// of this phase's operations.
	if !stat.ReadyRLock() {
		return stat
	}
	stat = ph.runSequences(stat)

	// Setup period over, status is now accessible safely.
	stat.RUnlock()

	// If this operation has an error, return a failed status.
	if ph.main() != nil {
		_ = stat.Fail() // Don't care if a failure already occured.
		return stat
	}

	// Block until "ready" (all child sequences finish).
	if stat.ReadyRLock() {
		stat.RUnlock()
	}
	return stat
}

func (ph phase) runSequences(stat status.Interface) status.Interface {
	stat.Add(len(ph.sequences))
	// Run each child sequence with a new status object.
	// Use a new status object so that different child sequences
	// don't wait on eachother.
	for _, seq := range ph.sequences {
		if stat.HasFailed() {
			break
		}
		go func(boundCopy status.Interface, seq runAller) {
			seq.runAll(boundCopy)

			// Mark this sequence as done.
			// If there was a failure, it propogates automatically.
			stat.Done()
		}(stat.BoundCopy(), seq)
	}
	return stat
}
