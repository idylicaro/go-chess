package search

import (
	"context"
	"sync/atomic"
)

// pollInterval is how many nodes a worker searches between checks of the
// stop conditions: often enough to stop within a millisecond, rarely enough
// that the check costs nothing.
const pollInterval = 1024

// stopper decides when a search ends: the caller cancels the context (UCI
// "stop"), or a node limit is reached. Workers share it, so it is safe for
// concurrent use.
type stopper struct {
	ctx       context.Context
	nodeLimit int64 // 0 means none
	nodes     atomic.Int64
	stopped   atomic.Bool
}

func newStopper(ctx context.Context, nodeLimit int64) *stopper {
	return &stopper{ctx: ctx, nodeLimit: nodeLimit}
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
	if s.ctx.Err() != nil {
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
