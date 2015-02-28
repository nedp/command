package command

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"bitbucket.org/nedp/command/status"
)

type runAllerMock struct {
	mock.Mock

	duration time.Duration
}

func (ra *runAllerMock) RunAll(stat status.Interface) status.Interface {
	args := ra.Called(stat)
	fuse := time.After(ra.duration)
	<-fuse
	return args.Get(0).(status.Interface)
}

func (ra *runAllerMock) IsRunning() bool {
	args := ra.Called()
	return args.Bool(0)
}

// Test `New` in sequence with `Run`.
// CC = ((1 + 1)total + 1) - 2 nodes
//    = 1
//
// sequence.Interface's black box behaviour varies:
//    success?    2 - true or false
//    time taken? 2 - short or long
// nVars = 2 * 2
//       = 4
//
// nTests = (1 CC) * (4 nVars)
//        = 4

// Command.Run should wait for seq.RunAll to return, then immediately returns its result.
func testRun(t *testing.T, expectSuccess bool, duration time.Duration) {
	// Set up the sequence mock according to parameters.
	runAller := new(runAllerMock)
	runAller.duration = duration
	output := make(chan string)
	c := New(runAller, output)
	runAller.On("RunAll", c.status).Return(c.status).Once()
	runAller.duration = duration

	if !expectSuccess {
		c.status.Fail()
	}
	start := time.Now()
	ch := make(chan bool)
	go func() {
		ch <- c.Run(make(chan string))
	}()
	fuse := time.After(duration * time.Duration(11) / time.Duration(10))

	// Time out if time since start exceeds maxDuration.
	var wasSuccessful bool
	select {
	case wasSuccessful = <-ch:
		// Okay
	case <-fuse:
		wasSuccessful = false
	}
	timeTaken := time.Since(start)

	// Verify the time taken
	assert.InEpsilon(t, int(duration), int(timeTaken), 0.2, "Unexpected delay")

	// Verify result
	assert.Equal(t, wasSuccessful, expectSuccess)

	// Verify expectations
	runAller.AssertExpectations(t)

	// Verify Output
	assert.Equal(t, 0, len(c.Output()),
		"Command produced outputs when expected not to.")
	runAller.AssertExpectations(t)
}

const shortDuration = time.Duration(500) * time.Millisecond
const longDuration = time.Duration(2) * time.Second

func TestNewFail(t *testing.T) {
	t.Parallel()
	testRun(t, false, shortDuration)
}

func TestNewSuccess(t *testing.T) {
	t.Parallel()
	testRun(t, true, shortDuration)
}

func TestNewLongFail(t *testing.T) {
	t.Parallel()
	testRun(t, false, longDuration)
}

func TestNewLongSuccess(t *testing.T) {
	t.Parallel()
	testRun(t, true, longDuration)
}

func TestStop(t *testing.T) {
	runAller := new(runAllerMock)
	runAller.duration = longDuration

	// There will be no output
	seqOut := make(chan string)
	close(seqOut)
	c := New(runAller, seqOut)
	runAller.On("RunAll", c.status).Return(c.status).Once()

	// The command should be externally stopped.
	cmdOut := make(chan string, 1)
	go func() {
		time.After(shortDuration)
		c.Stop()
	}()
	start := time.Now()

	wasSuccessful := c.Run(cmdOut)

	// The command finishes as soon as it's,
	// so it shouldn't finish early.
	assert.InEpsilon(t, int(longDuration), int(time.Since(start)), 0.3,
		"Stop delay was wrong")
	assert.False(t, wasSuccessful, "c.Run didn't return false")

	runAller.AssertExpectations(t)
}
