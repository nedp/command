package status

import (
	"testing"
	"time"
	//"errors"
	//"fmt"
	"strings"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)


const microDuration = time.Duration(100) * time.Microsecond
const milliDuration = time.Duration(100) * time.Millisecond
const tinyDuration = time.Second / time.Duration(2)
const shortDuration = time.Second
const mediumDuration = time.Duration(3) * time.Second
const longDuration = time.Duration(10) * time.Second
const nSmallTest = 100    // One Hundred
const nMediumTest = 10000 // Ten Thousand
const nBigTest = 1000000  // One million

const BigNumberHack = 1000000 // One million
const testPathDepth = 4

func TestFailReflectFail(t *testing.T) {
	status := New()
	status.Fail()
	boundCopy := status.BoundCopy()
	assert.False(t, status.ReadyRLock())
	assert.False(t, boundCopy.ReadyRLock())
}

type testCallParams struct {
	name string
	fn   func(s Interface) (bool, error)
	nexts []string

	effectsPropogate bool
	shouldPanic      bool
	shouldHalt       bool
	expect           bool

	reading int
	waitCount  int
}

type effect struct {
	shouldPanic *bool
	expect      *bool
	shouldHalt *bool

	reading int
	waitCount  int
}

type callSuite struct {
	suite.Suite
	effects map[string]map[string]effect
	defs    map[string]testCallParams
	hasFailed bool
}

func (c *callSuite) SetupSuite() {
	vtrue := true
	vfalse := false

	c.hasFailed = false

	c.defs = map[string]testCallParams{
		"Pause": {
			name: "Pause",
			fn: func(s Interface) (bool, error) {
				return s.Pause()
			},
			nexts: []string{"Pause", "Cont", "Fail", "ReadyRLock"},
			effectsPropogate: true,
			shouldPanic:      false,
			shouldHalt:       true,
			expect:           false,

			waitCount:  0,
			reading:    1,
		},
		"Cont": {
			name: "Cont",
			fn: func(s Interface) (bool, error) {
				return s.Cont()
			},
			nexts: []string{"Pause", "Cont", "Fail", "ReadyRLock"},
			effectsPropogate: true,
			shouldPanic:      false,
			shouldHalt:       true,
			expect:           true,

			waitCount:  0,
			reading:    1,
		},
		"Fail": {
			name: "Fail",
			fn: func(s Interface) (bool, error) {
				err := s.Fail()
				if err == nil {
					return false, nil
				} else {
					return false, s.Fail()
				}
			},
			nexts: []string{"Pause", "Cont", "Fail", "ReadyRLock"},
			effectsPropogate: true,
			shouldPanic:      false,
			shouldHalt:       true,
			expect:           false,

			waitCount:  0,
			reading:    1,
		},
		"ReadyRLock": {
			name: "ReadyRLock",
			fn: func(s Interface) (bool, error) {
				val := s.ReadyRLock()
				if val == false {
					c.hasFailed = true
				}
				return val, nil
			},
			nexts: []string{"RUnlock", "Pause", "Cont", "Fail"},
			effectsPropogate: true,
			shouldPanic:      false,
			shouldHalt:       true,
			expect:           true,

			waitCount:  0,
			reading:    1,
		},
		"RUnlock": {
			name: "RUnlock",
			fn: func(s Interface) (bool, error) {
				s.RUnlock()
				return false, nil
			},
			nexts: []string{"RUnlock", "Pause", "Cont", "Fail"},
			effectsPropogate: true,
			shouldPanic:      false,
			shouldHalt:       true,
			expect:           false,

			waitCount:  0,
			reading:    0,
		},
		"Add": {
			name: "Add",
			fn: func(s Interface) (bool, error) {
				s.Add(1)
				return false, nil
			},
			nexts: []string{"ReadyRLock", "Done", "BoundCopy"},
			effectsPropogate: false,
			shouldPanic:      false,
			shouldHalt:       true,
			expect:           false,

			waitCount:  0,
			reading:    1,
		},
		"Done": {
			name: "Done",
			fn: func(s Interface) (bool, error) {
				s.Done()
				return false, nil
			},
			nexts: []string{"ReadyRLock", "Add", "BoundCopy"},
			effectsPropogate: false,
			shouldPanic:      false,
			shouldHalt:       true,
			expect:           false,

			waitCount:  0,
			reading:    0,
		},
		"BoundCopy": {
			name:             "BoundCopy",
			fn: func(s Interface) (bool, error) {
				s.BoundCopy()
				return false, nil
			},
			nexts: []string{"ReadyRLock", "Add", "Done"},
			effectsPropogate: false,
			shouldPanic:      false,
			shouldHalt:       true,
			expect:           false,

			waitCount:  0,
			reading:    1,
			// Dummy
		},
	}

	c.effects = map[string]map[string]effect{
		"Pause": {
			"Pause":      {expect: &vtrue},
			"Cont":       {expect: &vfalse},
			"Fail":       {},
			"ReadyRLock": {shouldHalt: &vfalse},
			"RUnlock":    {},
			"Add":        {},
			"Done":       {},
			"BoundCopy": {},
		},
		"Cont": {
			"Pause":      {expect: &vfalse},
			"Cont":       {expect: &vtrue},
			"Fail":       {},
			"ReadyRLock": {shouldHalt: &vtrue},
			"RUnlock":    {},
			"Add":        {},
			"Done":       {},
			"BoundCopy": {},
		},
		"Fail": {
			"Pause": {shouldHalt: &vfalse, expect: &vtrue}, // TODO make this less hacky
			"Cont":  {shouldHalt: &vfalse, expect: &vfalse}, // TODO make this less hacky
			"Fail":  {shouldHalt: &vfalse, expect: &vtrue},
			"ReadyRLock": {
				shouldHalt:      &vtrue,
				waitCount: -BigNumberHack,
				expect:          &vfalse,
			},
			"RUnlock":    {},
			"Add":        {},
			"Done":       {},
			"BoundCopy": {},
		},
		"ReadyRLock": {
			"Pause":      {waitCount: 1},
			"Cont":       {waitCount: 1},
			"Fail":       {waitCount: 1},
			"ReadyRLock": {},
			"RUnlock":    {reading: 1},
			"Add":        {},
			"Done":       {},
			"BoundCopy": {},
		},
		"RUnlock": {
			"Pause":      {waitCount: -1},
			"Cont":       {waitCount: -1},
			"Fail":       {waitCount: -1},
			"ReadyRLock": {},
			"RUnlock":    {reading: -1},
			"Add":        {},
			"Done":       {},
			"BoundCopy": {},
		},
		"Add": {
			"Pause":      {},
			"Cont":       {},
			"Fail":       {},
			"ReadyRLock": {waitCount: 1},
			"RUnlock":    {},
			"Add":        {},
			"Done":       {reading: 1},
			"BoundCopy": {},
		},
		"Done": {
			"Pause":      {},
			"Cont":       {},
			"Fail":       {},
			"ReadyRLock": {waitCount: -1},
			"RUnlock":    {},
			"Add":        {},
			"Done":       {reading: -1},
			"BoundCopy": {},
		},
	}
}

func (c *callSuite) TestPaths() {
	c.testPathsOfLength(testPathDepth)
}

func (c *callSuite) params(name string, path []string) testCallParams {
	p, ok := c.defs[name]
	if !ok {
		c.T().Fatalf("Error in test - bad name: %s", name)
	}

	var iBoundCopy int
	for iBoundCopy = len(path) - 1; iBoundCopy >= 0; iBoundCopy -= 1 {
		if path[iBoundCopy] == "BoundCopy" {
			break
		}
	}
	hasFailed := false
	for i, effectName := range path {
		if effectName == "Fail" {
			hasFailed = true
		}
		if i > iBoundCopy || c.defs[effectName].effectsPropogate {
			p = p.withEffects(c.effects[effectName][name], effectName, hasFailed)
		}
	}
	return p.finalised()
}

func (p testCallParams) finalised() testCallParams {
	if p.reading < 1 {
		p.shouldPanic = true
	}
	if p.waitCount > 0 {
		p.shouldHalt = false
	}
	return p
}

func TestCallSuite(t *testing.T) {
	suite.Run(t, new(callSuite))
}

func (c *callSuite) testPathsOfLength(length int) {
	path := make([]string, length)
	c.testPaths(path, 0, length, c.allFunctions())
}

func (c *callSuite) allFunctions() []string {
	keys := make([]string, len(c.defs))
	i := 0
	for k := range c.defs {
		keys[i] = k
		i += 1
	}
	return keys
}

func (c *callSuite) testPaths(path []string, i int, capacity int, nexts []string) bool {
	if i+1 == capacity {
		nexts = c.allFunctions()
	}
	// For each possible operation at this point...
	for _, name := range nexts {
		// If we're sufficiently far through the path, test what we have so far.
		const thresholdN = 0
		const thresholdD = 100
		threshold := capacity * thresholdN / thresholdD
		if i >= threshold {
			if !c.testFinishedPath(path[:i], name) {
				return false
			}
		} else {
			// Otherwise, if it's early in the path,
			// check if it should panic or fail to halt.
			//
			// If it should do either of those things,
			// test the path up to here, but not child paths.

			p := c.params(name, path[:i])
			if p.shouldPanic || !p.shouldHalt {
				c.testFinishedPath(path[:i], name)
				continue
			}
		}

		// Test all child paths.
		path[i] = name
		if i+1 < capacity {
			c.testPaths(path, i+1, capacity, c.defs[name].nexts)
		}
	}
	return true
}


// Tests all steps in the path.
func (c *callSuite) testFinishedPath(path []string, name string) bool {
	status := New()

	for iPath, pathName := range path {
		p := c.params(pathName, path[:iPath])
		if pathName == "BoundCopy" {
			status = status.BoundCopy()
			continue
		}
		pathString := strings.Join(path[:iPath], "->")
		if !testCall(c.T(), status, p, pathString) || p.shouldPanic || !p.shouldHalt {
			return false
		}
	}
	pathString := strings.Join(path, "->")
	return testCall(c.T(), status, c.params(name, path), pathString)
}

func (p testCallParams) withEffects(effect effect, first string, hasFailed bool,
) testCallParams {
	if first == "ReadyRLock" && hasFailed {
		if p.name == "Pause" || p.name == "Cont" || p.name == "Fail" {
			effect.shouldHalt = nil
		}
		if p.name == "RUnlock" {
			effect.reading = 0
		}
	}
	p.reading += effect.reading
	p.waitCount += effect.waitCount

	if effect.shouldHalt != nil {
		p.shouldHalt = *effect.shouldHalt
	}

	if effect.shouldPanic != nil {
		p.shouldPanic = *effect.shouldPanic
	}

	if effect.expect != nil {
		p.expect = *effect.expect
	}
	return p
}

func testCall(t *testing.T, status Interface, p testCallParams, prefix string) bool {
	var didFail bool

	require.NotEqual(t, "", p.name)
	require.NotNil(t, p.fn, "name: %s", p.name)

	// Use a fuse to detect non-returning calls.
	fuse := time.After(milliDuration)
	out := make(chan bool, 1)
	panicCh := make(chan struct{}, 1)
	errCh := make(chan error, 1)


	// If it should panic, don't test the should-return expectation
	call := func() {
		val, err := p.fn(status)
		if err != nil {
			errCh <- err
		} else {
			out <- val
		}
	}

	switch {
	case p.shouldPanic:
		func() {
			// Use a defer/recover to verify panic expectations.
			defer func() {
				r := recover()
				if p.shouldPanic {
					a := assert.NotNil(t, r,
						"%s: %s didn't panic when expected.", prefix, p.name)
					if a {
						panicCh <- struct{}{}
					} else {
						didFail = true
					}
				} else {
					if !assert.Nil(t, r, "%s: %s panicked unexpectedly.", prefix, p.name) {
						didFail = true
					}
				}
			}()
			call()
		}()
	case p.shouldHalt:
		call()
	default:
		go call()
	}

	select {
	case <-panicCh:
		// Okay - we only sent through here if there was an expected panic.
	case b := <-out:
		a := assert.True(t, p.shouldHalt,
			"%s: %s returned when expected not to", prefix, p.name)
		if a {
			if !assert.Equal(t, p.expect, b, "%s: %s failed expectation", prefix, p.name) {
				didFail = true
			}
		} else {
			didFail = true
		}
	case <-fuse:
		a := assert.False(t, p.shouldHalt,
			"%s: %s didn't return when expected to", prefix, p.name)
		if !a {
			didFail = true
		}
	}
	return !didFail
}

func TestFullThenReflectRUnlock(t *testing.T) {
	// Should take 1 second if successful
	// (One short duration)
	t.Parallel()
	status := New()
	testPauseContFail(t, status)

	ch := make(chan bool)
	boundCopy := status.BoundCopy()
	go func() {
		ch <- boundCopy.ReadyRLock()
	}()
	micro := time.After(microDuration)
	select {
	case <-micro:
		t.Error("BoundCopy didn't instantly report failure")
	case b := <-ch:
		if b {
			t.Error("BoundCopy reported readiness when it should fail")
			boundCopy.RUnlock()
		}
	}
}

func TestReflectThenFullRUnlock(t *testing.T) {
	// Should take 1 second if successful
	// (One short duration)
	t.Parallel()
	status := New()
	boundCopy := status.BoundCopy()
	testPauseContFail(t, boundCopy)

	ch := make(chan bool)
	go func() {
		ch <- status.ReadyRLock()
	}()
	micro := time.After(microDuration)
	select {
	case <-micro:
		t.Error("BoundCopy didn't instantly report failure")
	case b := <-ch:
		if b {
			t.Error("BoundCopy reported readiness when it should fail")
			status.RUnlock()
		}
	}
}

// Desired behaviour confirmed.
// Commenting out to prevent contaminating other tests.
// func TestMegaBroadcast(t *testing.T) {
// 	// This test should result in the ReadyRLock loop being a huge heat sink.
// 	//t.Parallel()
// 	status := New()
// 	status.Pause()
// 	go status.ReadyRLock()
// 	go func() {
// 		for i := 0; i < nMediumTest; i+= 1 {
// 			status.Broadcast()
// 			time.Sleep(microDuration)
// 		}
// 		status.Cont()
// 	}()
// }

func testPauseContFail(t *testing.T, status Interface) {
	// Should take 1 second if successful
	// (One short duration)

	tinyFuse := time.After(tinyDuration)
	cont := time.After(shortDuration)
	fuse := time.After(mediumDuration)

	// Start by pausing it
	status.Pause()

	// Set up timers and a Cont call
	go func() {
		<-cont
		status.Cont()
	}()
	ch := make(chan bool)
	go func() {
		ch <- status.ReadyRLock()
	}()

	// Check it isn't ready too soon.
	select {
	case <-tinyFuse:
		// Ok
	case b := <-ch:
		if b {
			t.Error("Ready before it should be")
			status.RUnlock()
		} else {
			t.Error("Reported failure when it shouldn't.")
		}
	}

	// Check it isn't ready too late.
	select {
	case <-fuse:
		t.Error("Not ready when it should be.")
	case b := <-ch:
		assert.True(t, b, "Reported failure when it shouldn't.")
		if b {
			status.RUnlock()
		}
	}

	// Test failure
	status.Fail()
	go func() {
		ch <- status.ReadyRLock()
	}()
	micro := time.After(microDuration)
	select {
	case b := <-ch:
		assert.False(t, b, "Reported Ready after a failure")
		if b {
			status.RUnlock()
		}
	case <-micro:
		t.Error("Didn't instantly report failure when calling ReadyRLock()")
	}
}
