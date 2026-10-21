package search

import (
	"github.com/idylicaro/go-chess/pkg/chess"
)

// fiftyMovePlies is the fifty-move rule in plies.
const fiftyMovePlies = 100

// worker searches its own copy of the position.
type worker struct {
	board    *chess.SearchBoard
	evaluate Evaluator
	stop     *stopper
	pv       pvTable
	// unflushed counts nodes not yet added to the stopper.
	unflushed int64
	// canStop is false until depth 1 completes, so there is always a move.
	canStop bool
}

func newWorker(board *chess.SearchBoard, evaluate Evaluator, stop *stopper) *worker {
	return &worker{board: board, evaluate: evaluate, stop: stop}
}

// visit counts a node and reports whether the search must unwind.
func (w *worker) visit() bool {
	w.unflushed++
	if w.unflushed >= pollInterval {
		w.stop.addNodes(w.unflushed)
		w.unflushed = 0
	}
	return w.canStop && w.stop.isStopped()
}

// totalNodes flushes this worker's count and returns the shared total.
func (w *worker) totalNodes() int64 {
	w.stop.nodes.Add(w.unflushed)
	w.unflushed = 0
	return w.stop.nodes.Load()
}

// negamax returns the score of the position within (alpha, beta): a score
// at or below alpha is an upper bound, one at or above beta a lower bound.
func (w *worker) negamax(depth, ply, alpha, beta int) int {
	w.pv.clear(ply)
	if w.visit() {
		return drawScore // unwinding; the caller discards the result
	}
	if ply > 0 && w.isDraw() {
		return drawScore
	}
	if depth <= 0 || ply >= maxPly {
		return w.evaluate(w.board)
	}
	return w.searchMoves(depth, ply, alpha, beta)
}

// isDraw catches repetitions and the fifty-move rule inside the tree.
func (w *worker) isDraw() bool {
	return w.board.IsRepetition() || w.board.HalfMoveClock() >= fiftyMovePlies
}

// searchMoves tries every legal move and scores mate or stalemate when
// there is none.
func (w *worker) searchMoves(depth, ply, alpha, beta int) int {
	var buffer [chess.MaxSearchMoves]chess.SearchMove
	best, legalMoves := -infinity, 0
	for _, move := range w.board.GenerateMoves(buffer[:0]) {
		score, legal := w.searchChild(move, depth, ply, alpha, beta)
		if !legal {
			continue
		}
		legalMoves++
		best = max(best, score)
		if score > alpha {
			alpha = score
			w.pv.update(ply, move)
		}
		if alpha >= beta {
			break
		}
	}
	if legalMoves == 0 {
		return w.noMoveScore(ply)
	}
	return best
}

// searchChild plays move and scores the position after it, or returns
// legal false when move leaves the king in check.
func (w *worker) searchChild(move chess.SearchMove, depth, ply, alpha, beta int) (score int, legal bool) {
	undo, legal := w.board.MakeMove(move)
	if !legal {
		return 0, false
	}
	score = -w.negamax(depth-1, ply+1, -beta, -alpha)
	w.board.UnmakeMove(move, undo)
	return score, true
}

// noMoveScore is checkmate or stalemate.
func (w *worker) noMoveScore(ply int) int {
	if w.board.InCheck() {
		return matedIn(ply)
	}
	return drawScore
}
