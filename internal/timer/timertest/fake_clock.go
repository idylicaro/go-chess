// Package timertest provides test doubles for the timer package.
package timertest

import (
	"sync"
	"time"
)

// FakeClock is a manually advanced timer.Clock for deterministic tests.
//
//	clock := timertest.NewFakeClock()
//	gt := timer.NewGameTimerWithClock(config, clock)
//	clock.Advance(2 * time.Second)
type FakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewFakeClock returns a clock frozen at a fixed instant.
func NewFakeClock() *FakeClock {
	return &FakeClock{now: time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock forward by d.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
