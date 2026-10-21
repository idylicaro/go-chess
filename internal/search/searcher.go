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

// Limits bound a search, as the fields of UCI "go" do. With none set it
// runs until the context ends.
type Limits struct {
	Depth    int           // plies; 0 means up to maxPly
	Nodes    int64         // 0 means no node limit
	MoveTime time.Duration // exact time for this move
	// Clocks: the side to move's remaining time and increment set the
	// budget, assuming MovesToGo moves to the next time control (0 means
	// sudden death).
	WTime, BTime, WInc, BInc time.Duration
	MovesToGo                int
	// Infinite ignores every time field; only the context ends the search.
	Infinite bool
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

// Options configure a Searcher; the zero value uses the defaults.
type Options struct {
	HashMB int // transposition table size; 0 means DefaultHashMB
}

// Searcher runs searches with one evaluation function and keeps its
// transposition table from one search to the next. Run one search at a
// time; Search may be called again for the next move.
type Searcher struct {
	evaluate Evaluator
	tt       *transpositionTable
	now      func() time.Time
}

// New returns a Searcher that scores leaves with evaluate.
//
//	searcher := search.New(eval.Evaluate, search.Options{HashMB: 64})
//	result := searcher.Search(ctx, board, search.Limits{Depth: 6}, nil)
func New(evaluate Evaluator, options Options) *Searcher {
	hashMB := options.HashMB
	if hashMB <= 0 {
		hashMB = DefaultHashMB
	}
	return &Searcher{evaluate: evaluate, tt: newTranspositionTable(hashMB), now: time.Now}
}

// Clear forgets what earlier searches learned, for a new game.
func (s *Searcher) Clear() {
	s.tt.clear()
}

// Search looks for the best move on board, which it leaves as it found it.
// report, if not nil, is called after every completed depth.
func (s *Searcher) Search(ctx context.Context, board *chess.SearchBoard, limits Limits, report func(Info)) Result {
	start := s.now()
	budget := budgetFor(limits, board.SideToMove())
	stop := newStopper(ctx, limits.Nodes)
	if budget.limited {
		stop.withDeadline(start.Add(budget.hard), s.now)
	}
	w := newWorker(board.Clone(), s.evaluate, stop, s.tt)
	result := s.deepen(w, maxDepth(limits), budget, start, report)
	return finish(result, w.totalNodes())
}

// deepen runs iterative deepening and returns the last completed iteration.
func (s *Searcher) deepen(w *worker, depthLimit int, budget timeBudget, start time.Time, report func(Info)) Result {
	var result Result
	for depth := 1; depth <= depthLimit; depth++ {
		score := w.negamax(depth, 0, -infinity, infinity)
		if w.stop.isStopped() && depth > 1 {
			break // an interrupted iteration is not trusted
		}
		result = w.iterationResult(depth, score)
		reportIteration(report, Info{depth, score, w.totalNodes(), s.now().Sub(start), result.PV})
		w.canStop = true
		if w.stop.check() || isMateScore(score) || !budget.startNextIteration(s.now().Sub(start)) {
			break // depth 1 always completes, so there is a move to play
		}
	}
	return result
}

// iterationResult records a completed iteration; its best move is searched
// first in the next one.
func (w *worker) iterationResult(depth, score int) Result {
	result := Result{Score: score, Depth: depth, PV: w.pv.line()}
	if len(result.PV) > 0 {
		w.rootBest = result.PV[0]
	}
	return result
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
