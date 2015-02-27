package command

import (
	"time"

	"bitbucket.org/nedp/command/status"
	"bitbucket.org/nedp/command/sequence"
)

type Command struct {
	status status.Interface
	runAller sequence.RunAller
	output <-chan string
}

func New(runAller sequence.RunAller, output <-chan string) *Command {
	return &Command{status.New(), runAller, output}
}

// A wrapper for sequence.RunAll
func (c *Command) Run() bool {
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
func (c *Command) Kill() error {
	return c.status.Fail()
}

// TODO document
func (c *Command) WhenTerminated() <-chan time.Time {
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
func (c *Command) Output() <-chan string {
	return c.output
}
