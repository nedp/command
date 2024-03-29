package command

import (
	"github.com/nedp/command/status"
	"github.com/nedp/command/sequence"
)


type Interface interface {
	Runner
	Pauser
	Stopper
	State() State
	Output() []string

	// Name return's the command's assigned name.
	Name() string
}

type Runner interface {
	Run(chan<- string) bool
	IsRunning() bool
}

type Pauser interface {
	Pause() (bool, error)
	Cont() (bool, error)
	IsPaused() bool
}

type Stopper interface {
	Stop() error
	HasStopped() bool
}

type Command struct {
	name string
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
func New(runAller sequence.RunAller, name string) *Command {
	lg := newLogger(runAller.OutputChannel())
	return &Command{name, status.New(), runAller, lg}
}

// NewForOutLength creates a new command object, initially allocating
// the specified number of strings for output.
//
// If running the command causes it to run out of output space,
// more will be allocated.
//
// Returns
// the new Command.
func NewForOutLength(runAller sequence.RunAller, name string, outLen int) *Command {
	lg := newLoggerWithCap(runAller.OutputChannel(), outLen)
	return &Command{name, status.New(), runAller, lg}
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

	c.status = c.runAller.RunAll(c.status)
	c.logger.stop()
	return !c.status.HasFailed()
}

// A wrapper for status.Interface.Pause
func (c *Command) Pause() (bool, error) {
	return c.status.Pause()
}

// A wrapper for status.Interface.IsPaused
func (c *Command) IsPaused() bool {
	return c.status.IsPaused()
}

// A wrapper for status.Interface.Cont
func (c *Command) Cont() (bool, error) {
	return c.status.Cont()
}

// A wrapper for status.Interface.Fail
func (c *Command) Stop() error {
	return c.status.Fail()
}

// A wrapper for status.Interface.HasFailed
func (c *Command) HasStopped() bool {
	return c.status.HasFailed()
}

// TODO document
func (c *Command) Output() []string {
	var output []string
	copy(output, c.logger.log)
	return output
}

// A wrapper for sequence.IsRunnig
func (c *Command) IsRunning() bool {
	return c.runAller.IsRunning()
}

type State struct {
	IsPaused bool
	IsRunning bool
	HasStopped bool
	Output []string
}

// State returns a threadsafe view of all externally visible
// parts of the command's state.
func (c *Command) State() State {
	c.status.RLock()
	defer c.status.RUnlock()
	return State{
		c.IsPaused(),
		c.IsRunning(),
		c.HasStopped(),
		c.Output(),
	}
}

func (c *Command) Name() string {
	return c.name
}
