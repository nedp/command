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
	WhenStopped() <-chan Time.time
}

type Command struct {
	status status.Interface
	runAller sequence.RunAller
	outputLog logger
}

// TODO document
func New(runAller sequence.RunAller, seqOut <-chan string, nOutputs int) Interface {
	var lg logger
	if nOutputs < 0 {
		lg = newLogger(seqOut)
	} else {
		lg = newLoggerWithCap(seqOut, nOutputs)
	}
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
	go c.lg.listen(outCh)
	go func() {
		<-c.WhenStopped()
		lg.stop()
	}(lg)

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
