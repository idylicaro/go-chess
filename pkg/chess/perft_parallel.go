package chess

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// PerftParallel is Perft with the root moves shared among workers
// goroutines, each searching its own copy of the position with make/unmake
// (#35). Workers take the next unsearched root move as they finish, so a
// large subtree does not hold the others back. workers <= 0 means one per
// CPU. The count is always the same as Perft's.
//
//	NewGame().PerftParallel(6, 0) // 119060324, using every CPU
func (g *Game) PerftParallel(depth, workers int) int {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	root := g.clone()
	moves := root.appendLegalMoves(nil)
	if depth <= 1 || workers == 1 {
		return root.perft(depth)
	}
	queue := &rootMoveQueue{moves: moves}
	var total atomic.Int64
	var done sync.WaitGroup
	for range min(workers, len(moves)) {
		search := root.clone()
		done.Add(1)
		go func() {
			defer done.Done()
			total.Add(int64(search.perftFromQueue(queue, depth)))
		}()
	}
	done.Wait()
	return int(total.Load())
}

// rootMoveQueue hands out root moves to workers, each exactly once.
type rootMoveQueue struct {
	moves []genMove
	next  atomic.Int64
}

func (q *rootMoveQueue) take() (genMove, bool) {
	i := q.next.Add(1) - 1
	if i >= int64(len(q.moves)) {
		return genMove{}, false
	}
	return q.moves[i], true
}

// perftFromQueue searches root moves until the queue is empty.
func (g *Game) perftFromQueue(queue *rootMoveQueue, depth int) int {
	nodes := 0
	for move, ok := queue.take(); ok; move, ok = queue.take() {
		undo := g.makeMove(move)
		nodes += g.perft(depth - 1)
		g.unmakeMove(move, undo)
	}
	return nodes
}
