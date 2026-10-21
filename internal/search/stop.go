package search

import (
	"context"
	"sync/atomic"
	"time"
)

// pollInterval is how many nodes a worker searches between checks of the
// stop conditions: often enough to stop within a millisecond, rarely enough
// that the check costs nothing.
const pollInterval = 1024

// stopper decides when a search ends: the caller cancels the context (UCI
// "stop"), a node limit is reached or the hard time budget runs out.
// Workers share it, so it is safe for concurrent use.
type stopper struct {
	ctx       context.Context
	nodeLimit int64     // 0 means none
	deadline  time.Time // zero means none
	now       func() time.Time
	nodes     atomic.Int64
	stopped   atomic.Bool
}

// newStopper has no deadline; give it one with withDeadline.
func newStopper(ctx context.Context, nodeLimit int64) *stopper {
	return &stopper{ctx: ctx, nodeLimit: nodeLimit, now: time.Now}
}

// withDeadline stops the search at deadline as read from now.
func (s *stopper) withDeadline(deadline time.Time, now func() time.Time) *stopper {
	s.deadline, s.now = deadline, now
	return s
}

// addNodes counts a worker's nodes and checks whether to stop.
func (s *stopper) addNodes(count int64) {
	s.nodes.Add(count)
	s.check()
}

// check stops the search once a limit is reached, and reports whether it
// is stopped.
func (s *stopper) check() bool {
	if s.nodeLimit > 0 && s.nodes.Load() >= s.nodeLimit {
		s.stop()
	}
	if s.ctx.Err() != nil || s.pastDeadline() {
		s.stop()
	}
	return s.isStopped()
}

func (s *stopper) stop() {
	s.stopped.Store(true)
}

func (s *stopper) isStopped() bool {
	return s.stopped.Load()
}

func (s *stopper) pastDeadline() bool {
	return !s.deadline.IsZero() && !s.now().Before(s.deadline)
}
