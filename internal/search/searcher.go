// Package search is the engine's search: negamax with alpha-beta pruning and
// iterative deepening over a chess.SearchBoard (#37).
package search

import (
	"context"
	"time"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// Evaluator scores a position from the side to move's view, in centipawns.
type Evaluator func(board *chess.SearchBoard) int

// Limits bound a search. With none set it runs until the context ends.
type Limits struct {
	Depth int   // plies; 0 means up to maxPly
	Nodes int64 // 0 means no node limit
}

// Info is reported after each completed iteration, for UCI "info" lines.
type Info struct {
	Depth   int
	Score   int // centipawns, or a mate score (see MateInMoves)
	Nodes   int64
	Elapsed time.Duration
	PV      []chess.SearchMove
}

// Result is the outcome of a search: the move to play and what backs it.
// Best is chess.NullSearchMove when the side to move has no legal move.
type Result struct {
	Best  chess.SearchMove
	Score int
	Depth int
	Nodes int64
	PV    []chess.SearchMove
}

// Searcher runs searches with one evaluation function. Run one search at a
// time; Search may be called again for the next move.
type Searcher struct {
	evaluate Evaluator
	now      func() time.Time
}

// New returns a Searcher that scores leaves with evaluate.
//
//	searcher := search.New(eval.Evaluate)
//	result := searcher.Search(ctx, board, search.Limits{Depth: 6}, nil)
func New(evaluate Evaluator) *Searcher {
	return &Searcher{evaluate: evaluate, now: time.Now}
}

// Search looks for the best move on board, which it leaves as it found it.
// report, if not nil, is called after every completed depth.
func (s *Searcher) Search(ctx context.Context, board *chess.SearchBoard, limits Limits, report func(Info)) Result {
	start := s.now()
	stop := newStopper(ctx, limits.Nodes)
	w := newWorker(board.Clone(), s.evaluate, stop)
	var result Result
	for depth := 1; depth <= maxDepth(limits); depth++ {
		score := w.negamax(depth, 0, -infinity, infinity)
		if stop.isStopped() && depth > 1 {
			break // an interrupted iteration is not trusted
		}
		result = Result{Score: score, Depth: depth, PV: w.pv.line()}
		if len(result.PV) > 0 {
			w.rootBest = result.PV[0]
		}
		reportIteration(report, Info{depth, score, w.totalNodes(), s.now().Sub(start), result.PV})
		w.canStop = true
		if stop.check() || isMateScore(score) {
			break // depth 1 always completes, so there is a move to play
		}
	}
	return finish(result, w.totalNodes())
}

func maxDepth(limits Limits) int {
	if limits.Depth <= 0 || limits.Depth > maxPly {
		return maxPly
	}
	return limits.Depth
}

func reportIteration(report func(Info), info Info) {
	if report != nil {
		report(info)
	}
}

// finish fills in the best move from the principal variation.
func finish(result Result, nodes int64) Result {
	result.Nodes = nodes
	if len(result.PV) > 0 {
		result.Best = result.PV[0]
	}
	return result
}
