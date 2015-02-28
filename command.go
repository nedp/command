package command

import (
	"time"

	"bitbucket.org/nedp/command/status"
	"bitbucket.org/nedp/command/sequence"
)


type Interface interface {
	Runner
	Pauser
	Stopper
	Output() []string
}

type Runner interface {
	Run(chan<- string) bool
}

type Pauser interface {
	Pause() (bool, error)
	Cont() (bool, error)
}

type Stopper interface {
	Stop() error
	WhenStopped() <-chan time.Time
}

type Command struct {
	status status.Interface
	runAller sequence.RunAller
	logger logger
}

// New creates a new command object, initially allocating
// the default amount of space for output.
// If an estimate of of the number of outputs is available, use
// NewWithNOutputs instead to more efficiently allocate.
//
// Returns
// the new Command.
func New(runAller sequence.RunAller, seqOut <-chan string) *Command {
	lg := newLogger(seqOut)
	return &Command{status.New(), runAller, lg}
}

// NewWithNOutputs creates a new command object, 
// allocating space for the specified number of output strings.
// If space for output runs out, more will be alocated automatically.
//
// Returns
// the new Command.
func NewWithNOutputs(runAller sequence.RunAller, seqOut <-chan string, nOutputs int,
) *Command {
	lg := newLoggerWithCap(seqOut, nOutputs)
	return &Command{status.New(), runAller, lg}
}

// Run calls RunAll on the command's RunAller, having the
// command's logger record and forward output from the
// sequence to outCh.
//
// The logger will stop recording output when the RunAller
// is no longer running.
//
// Returns
// true if the status is fine;
// false if there has been a failure.
func (c *Command) Run(outCh chan<- string) bool {
	go c.logger.listen(outCh)
	go func(done <-chan time.Time, lg logger) {
		<-done
		lg.stop()
	}(c.WhenStopped(), c.logger)

	c.status = c.runAller.RunAll(c.status)
	return !c.status.HasFailed()
}

// A wrapper for status.Interface.Pause
func (c *Command) Pause() (bool, error) {
	return c.status.Pause()
}

// A wrapper for status.Interface.Cont
func (c *Command) Cont() (bool, error) {
	return c.status.Cont()
}

// A wrapper for status.Interface.Fail
func (c *Command) Stop() error {
	return c.status.Fail()
}

// TODO document
func (c *Command) WhenStopped() <-chan time.Time {
	ch := make(chan time.Time)
	go func(ch chan<- time.Time) {
		for c.status.ReadyRLock() && c.runAller.IsRunning() {
			c.status.RUnlock()
		}
		ch <- time.Now()
		close(ch)
	}((chan<- time.Time)(ch))
	return (<-chan time.Time)(ch)
}

// TODO document
func (c *Command) Output() []string {
	var output []string
	copy(output, c.logger.log)
	return output
}
