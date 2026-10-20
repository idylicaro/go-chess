package timer

import "time"

// Clock is the time source of a GameTimer. Injecting it lets tests drive time
// explicitly instead of sleeping (#6).
type Clock interface {
	Now() time.Time
}

// SystemClock reads the wall clock.
//
//	gt := NewGameTimerWithClock(Blitz3Plus2, SystemClock{})
type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}
