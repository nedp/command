package command

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	t.Parallel()
	testStrings := []string{
		"test string 1",
		"test string 2",
	}

	in := make(chan string)
	done := make(chan struct{})
	go func(out chan<- string, done chan<- struct{}) {
		for _, s := range testStrings {
			out <- s
		}
		done <- struct{}{}
	}(in, done)

	out := make(chan string)
	lg := newLogger(in)
	go lg.listen(out)

	for i := range testStrings {
		assert.Equal(t, testStrings[i], <-out, "Test string %d didn't match out", i)
	}
	const timeout = time.Duration(1) * time.Second
	fuse := time.After(timeout)
	select {
	case <-fuse:
		t.Error("Test timed out")
	case <-done:
		// Okay
	}

	println(len(lg.log))
	for i, s := range testStrings {
		assert.Equal(t, s, lg.log[i], "Test string %d didn't match the log", i)
	}
}
