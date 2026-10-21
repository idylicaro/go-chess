package search

import (
	"sync"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// Lazy SMP (lab #45): extra goroutines search the same position with their
// own board and move ordering, sharing only the transposition table and the
// stop conditions. They find nothing the main worker reports; they fill the
// table with results the main worker then reads instead of searching, and
// the different depths they start at make them explore different parts of
// the tree. It is the scheme of most modern engines because it needs no
// work splitting and no locks, which the lockless table (#40) makes safe.

// startHelpers runs threads-1 helpers until stop; wait for them with the
// returned group after stopping it.
func (s *Searcher) startHelpers(board *chess.SearchBoard, stop *stopper) *sync.WaitGroup {
	var helpers sync.WaitGroup
	for i := 1; i < s.threads; i++ {
		helper := s.newWorker(board, stop)
		helper.canStop = true // only the main worker must finish depth 1
		helpers.Add(1)
		go func() {
			defer helpers.Done()
			defer helper.totalNodes() // flush the node count
			helper.helpDeepen(1 + i%2)
		}()
	}
	return &helpers
}

// helpDeepen is iterative deepening that reports nothing and stops only
// when the main worker is done. Odd helpers start one ply deeper, so the
// helpers are not all on the same iteration.
func (w *worker) helpDeepen(firstDepth int) {
	for depth := firstDepth; depth <= maxPly && !w.stop.isStopped(); depth++ {
		w.negamax(depth, 0, -infinity, infinity)
		if w.pv.length[0] > 0 && !w.stop.isStopped() {
			w.rootBest = w.pv.moves[0][0]
		}
	}
}
