package sequence

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"bitbucket.org/nedp/command/status"
)

const (
	microDuration = time.Duration(50) * time.Microsecond
	milliDuration = time.Duration(10) * time.Millisecond
	tinyDuration = time.Duration(500) * time.Millisecond
	shortDuration = time.Duration(1) * time.Second
)

const nSequences = 100

type runAllerMock struct {
	mock.Mock
	duration time.Duration
}

type statusMock struct {
	ch chan int
	isReflection bool
	boundCopy status.Interface
	hasFailed bool

	mock.Mock
}

func (s *statusMock) BoundCopy() status.Interface {
	s.Called()
	return s.boundCopy
}

func (s *statusMock) Add(delta int) {
	s.Called(delta)
	s.ch <- (<-s.ch + delta)
}

func (s *statusMock) Done() {
	s.Called()
	s.ch <- (<-s.ch - 1)
}

func (s *statusMock) ReadyRLock() bool {
	args := s.Called()
	if s.hasFailed {
		return false
	}

	n := <-s.ch
	for n > 0 {
		s.ch <- n
		time.Sleep(microDuration)
		n = <-s.ch

		if s.hasFailed {
			return false
		}
	}
	s.ch <- n
	return args.Bool(0)
}

func (s *statusMock) RUnlock() {
	s.Called()
}

func (s *statusMock) Pause() (bool, error) {
	args := s.Called()
	return args.Bool(0), args.Error(1)
}

func (s *statusMock) Cont() (bool, error) {
	args := s.Called()
	return args.Bool(0), args.Error(1)
}

func (s *statusMock) Fail() error {
	args := s.Called()
	return args.Error(0)
}

func (s *statusMock) HasFailed() bool {
	s.hasFailed = s.Called().Bool(0)
	return s.hasFailed
}

func (ra *runAllerMock) runAll(stat status.Interface) status.Interface {
	args := ra.Called(stat)
	fuse := time.After(ra.duration)
	<-fuse
	return args.Get(0).(status.Interface)
}

func TestRunSequencesSuccess(t *testing.T) {
	stat := new(statusMock)
	stat.On("BoundCopy").Return().Times(nSequences)
	boundCopy := new(statusMock)
	boundCopy.isReflection = true
	stat.boundCopy = status.Interface(boundCopy)
	stat.On("Add", nSequences).Return().Once()
	stat.ch = make(chan int, 1)
	stat.ch <- 0
	stat.On("Done").Return().Times(nSequences)
	stat.On("ReadyRLock").Return(true).Once()
	stat.On("HasFailed").Return(false).Times(nSequences)

	const testDuration = shortDuration
	phase := new(phase)
	phase.main = func() error { return nil }
	phase.sequences = make([]runAller, nSequences)
	const iFailure = nSequences / 2
	for i := 0; i < nSequences; i += 1 {
		seq := &runAllerMock{duration: testDuration}
		seq.On("runAll", boundCopy).Return(stat).Once()
		phase.sequences[i] = seq
	}

	// Verify that the sequences were started in separate goroutines
	start := time.Now()
	phase.runSequences(stat)
	assert.WithinDuration(t, time.Now(), start, milliDuration,
		"runSequences took too long")

	// Verify that the sequences ran concurrently
	stat.ReadyRLock()
	assert.InEpsilon(t, int(testDuration), int(time.Since(start)), 0.5,
		"Test duration outside acceptable valuestat.")

	// Verify expectations
	boundCopy.AssertExpectations(t)
	stat.AssertExpectations(t)
	for i := 0; i < nSequences; i += 1 {
		phase.sequences[i].(*runAllerMock).AssertExpectations(t)
	}
}

func TestRunSequencesFailure(t *testing.T) {
	const iFailure = nSequences / 2
	stat := new(statusMock)
	// No Times because the failure occurs at unknown exact time.
	stat.On("BoundCopy").Return()
	boundCopy := new(statusMock)
	boundCopy.isReflection = true
	stat.boundCopy = status.Interface(boundCopy)
	stat.On("Add", nSequences).Return().Once()
	stat.ch = make(chan int, 1)
	stat.ch <- 0
	stat.On("Done").Return()
	stat.On("ReadyRLock").Return(true).Once()
	stat.On("HasFailed").Return(false).Times(iFailure + 1)
	stat.On("HasFailed").Return(true).Once()
	//stat.On("Fail").Return(true).Once()

	const testDuration = shortDuration
	phase := new(phase)
	nCalls := make(chan int)
	phase.main = func() error {
		nCalls <- (<-nCalls) - 1
		return nil
	}
	phase.sequences = make([]runAller, nSequences)
	for i := 0; i < nSequences; i += 1 {
		seq := &runAllerMock{duration: testDuration}
		seq.On("runAll", boundCopy).Return(boundCopy).Once()
		phase.sequences[i] = seq
	}
	seq := &runAllerMock{duration: testDuration}
	seq.On("runAll", boundCopy).Return(boundCopy).Once()
	phase.sequences[iFailure] = seq

	// Verify that the sequences were started in separate goroutines
	start := time.Now()
	phase.runSequences(stat)
	assert.WithinDuration(t, time.Now(), start, milliDuration,
		"runSequences took too long")

	// Verify that the sequences ran concurrently, and failed
	// (making ReadyRLock end early)
	assert.False(t, stat.ReadyRLock(), "Didn't fail")
	assert.WithinDuration(t, time.Now(), start, milliDuration,
		"ReadyRLong took too long")

	// Wait for sufficient completions
	for n := <-stat.ch; n > nSequences - iFailure; n = <-stat.ch {
		stat.ch <- n
	}

	// Verify expectations
	boundCopy.AssertExpectations(t)
	stat.AssertExpectations(t)

	// Don't check after the failure.
	for i := 0; i <= iFailure; i += 1 {
		phase.sequences[i].(*runAllerMock).AssertExpectations(t)
	}
	phase.sequences[nSequences - 1].(*runAllerMock).
		AssertNotCalled(t, "runAll", boundCopy)
}

func TestRunAllSimple(t *testing.T) {
	stat := new(statusMock)
	stat.ch = make(chan int, 1)
	stat.ch <- 0
	stat.On("ReadyRLock").Return(true).Times(2)
	stat.On("RUnlock").Return().Times(2)
	stat.On("Add", 0).Return().Once()

	ph := phase{}
	ph.main = func() error {
		return nil
	}
	ph.sequences = []runAller{}

	stat = ph.runAll(stat).(*statusMock)

	// Validate expectations
	assert.False(t, stat.hasFailed, "RunAll reported unexpected failure")
	stat.AssertExpectations(t)

	// Validate that we didn't enter the loop in runSequences
	stat.AssertNotCalled(t, "HasFailed")
}

func TestRunAllFailure(t *testing.T) {
	stat := new(statusMock)
	stat.ch = make(chan int, 1)
	stat.ch <- 0
	stat.On("ReadyRLock").Return(true).Times(1)
	stat.On("RUnlock").Return().Times(1)
	stat.On("Add", 1).Return().Once()
	stat.On("Fail").Return(nil).Once()
	stat.On("HasFailed").Return(true).Once()

	seq := new(runAllerMock)

	ph := phase{}
	ph.main = func() error {
		return errors.New("Failure")
	}
	ph.sequences = []runAller{seq}

	stat = ph.runAll(stat).(*statusMock)

	// Validate expectations
	assert.True(t, stat.hasFailed, "RunAll reported unexpected success")
	stat.AssertExpectations(t)

	// Validate that runSequences returned after HasFailed was true.
	seq.AssertNotCalled(t, "runAll")
}
