package sequence

import (
	"testing"
	"time"

	//"bitbucket.org/nedp/command/status"

	"github.com/stretchr/testify/assert"
)

const usualDuration = milliDuration // phase_test.go

func TestRunAllPrivateSuccess(t *testing.T) {
	testRunAllSuccess(t, false, 5, Sequence{})
}

func TestRunAllPrivateFailure(t *testing.T) {
	testRunAllFailure(t, false, 5, Sequence{})
}

func TestRunAllPublicSuccess(t *testing.T) {
	testRunAllSuccess(t, true, 5, Sequence{isRunning: make(chan bool, 1)})
}

func TestRunAllPublicFailure(t *testing.T) {
	testRunAllFailure(t, true, 5, Sequence{isRunning: make(chan bool, 1)})
}

func TestIsRunning(t *testing.T) {
	const testDuration = shortDuration
	stat := new(statusMock)
	stat.On("HasFailed").Return(false).Once()

	seq := Sequence{}
	seq.isRunning = make(chan bool, 1)

	ph := new(runAllerMock)
	ph.duration = testDuration
	ph.On("runAll", stat).Return(stat).Once()
	seq.phases = []runAller{ph}

	go seq.RunAll(stat)

	time.Sleep(microDuration)
	assert.True(t, seq.IsRunning(), "First IsRunning call returned false.")

	const fullDuration = testDuration + microDuration
	fuse := time.After(time.Duration(2) * fullDuration)

	for seq.IsRunning() {
		select {
		case <-fuse:
			t.Error("Waiting for `IsRunning == false` timed out")
			break
		default:
			// Do nothing
		}
	}
}

func testRunAllSuccess(t *testing.T, shouldUsePublic bool, nPhases int, seq Sequence) {
	const testDuration = usualDuration
	seq.phases = make([]runAller, nPhases)

	// Mock a status
	stat := new(statusMock)
	stat.On("HasFailed").Return(false).Times(nPhases)
	stat.ch = make(chan int, 1)
	stat.ch <- 0

	// Mock a list of phases to run through
	for i := 0; i < nPhases; i += 1 {
		ph := new(runAllerMock)
		ph.duration = testDuration
		ph.On("runAll", stat).Return(stat).Once()
		seq.phases[i] = ph
	}

	if shouldUsePublic {
		stat = seq.RunAll(stat).(*statusMock)
	} else {
		stat = seq.runAll(stat).(*statusMock)
	}

	// Verify success.
	assert.False(t, stat.hasFailed)

	// Verify expectations
	stat.AssertExpectations(t)
	for _, ph := range seq.phases {
		ph.(*runAllerMock).AssertExpectations(t)
	}
}

func testRunAllFailure(t *testing.T, shouldUsePublic bool, nPhases int, seq Sequence) {
	const testDuration = usualDuration
	iFailure := int(nPhases) / 2

	seq.phases = make([]runAller, nPhases)

	// Mock a status
	stat := new(statusMock)
	stat.On("HasFailed").Return(false).Times(iFailure)
	stat.On("HasFailed").Return(true).Once()

	// Exits without unlocking
	stat.ch = make(chan int, 1)
	stat.ch <- 0

	// Mock a set of phases to run through, one of which should fail.
	for i := 0; i < iFailure; i += 1 {
		ph := new(runAllerMock)
		ph.duration = testDuration
		ph.On("runAll", stat).Return(stat).Once()
		seq.phases[i] = ph
	}

	ph := new(runAllerMock)
	ph.duration = testDuration
	ph.On("runAll", stat).Return(stat).Once()
	seq.phases[iFailure] = ph

	for i := iFailure + 1; i < nPhases; i += 1 {
		ph := new(runAllerMock)
		ph.duration = testDuration
		seq.phases[i] = ph
	}

	if shouldUsePublic {
		stat = seq.RunAll(stat).(*statusMock)
	} else {
		stat = seq.runAll(stat).(*statusMock)
	}

	// Verify that the failure was noticed.
	assert.True(t, stat.hasFailed)

	stat.AssertExpectations(t)
	for _, ph := range seq.phases {
		ph.(*runAllerMock).AssertExpectations(t)
	}
}
