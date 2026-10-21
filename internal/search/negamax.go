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
	tt       *transpositionTable
	pv       pvTable
	order    moveOrderer
	// rootBest is the previous iteration's best move, searched first.
	rootBest chess.SearchMove
	// unflushed counts nodes not yet added to the stopper.
	unflushed int64
	// canStop is false until depth 1 completes, so there is always a move.
	canStop bool
}

func newWorker(board *chess.SearchBoard, evaluate Evaluator, stop *stopper, tt *transpositionTable) *worker {
	return &worker{board: board, evaluate: evaluate, stop: stop, tt: tt}
}

// visit counts a node and reports whether the search must unwind.
func (w *worker) visit() bool {
	w.unflushed++
	if w.unflushed >= pollInterval {
		w.stop.addNodes(w.unflushed)
		w.unflushed = 0
	}
	return w.unwinding()
}

// unwinding is true once the search was stopped: scores from here on are
// meaningless and must not be stored or used.
func (w *worker) unwinding() bool {
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
	if ply >= maxPly {
		return w.evaluate(w.board)
	}
	if depth <= 0 {
		return w.quiesce(ply, alpha, beta)
	}
	entry, found := w.tt.probe(w.board.Hash())
	if score, settled := entry.cutoffScore(depth, ply, alpha, beta); found && settled && ply > 0 {
		return score
	}
	score, best := w.searchMoves(depth, ply, alpha, beta, entry.move)
	w.storeResult(depth, ply, score, best, boundFor(score, alpha, beta))
	return score
}

// storeResult records a finished node in the transposition table.
func (w *worker) storeResult(depth, ply, score int, best chess.SearchMove, kind bound) {
	if w.unwinding() {
		return
	}
	w.tt.store(w.board.Hash(), ttEntry{move: best, score: scoreToTT(score, ply), depth: depth, bound: kind})
}

// isDraw catches repetitions and the fifty-move rule inside the tree.
func (w *worker) isDraw() bool {
	return w.board.IsRepetition() || w.board.HalfMoveClock() >= fiftyMovePlies
}

// searchMoves tries every legal move, best first, and scores mate or
// stalemate when there is none. It returns the score and the best move:
// the one that raised alpha, or ttMove when none did.
func (w *worker) searchMoves(depth, ply, alpha, beta int, ttMove chess.SearchMove) (int, chess.SearchMove) {
	var buffer [chess.MaxSearchMoves]chess.SearchMove
	picker := w.order.newMovePicker(w.board, w.board.GenerateMoves(buffer[:0]), w.hashMove(ply, ttMove), ply)
	best, bestMove, legalMoves := -infinity, ttMove, 0
	for move, ok := picker.pick(); ok; move, ok = picker.pick() {
		score, legal := w.searchChild(move, depth, ply, alpha, beta)
		if !legal {
			continue
		}
		legalMoves++
		best = max(best, score)
		if score > alpha {
			alpha, bestMove = score, move
			w.pv.update(ply, move)
		}
		if alpha >= beta {
			w.recordCutoff(move, depth, ply)
			break
		}
	}
	if legalMoves == 0 {
		return w.noMoveScore(ply), chess.NullSearchMove
	}
	return best, bestMove
}

// hashMove is the move to try first at ply: the previous iteration's best
// at the root, the transposition table's elsewhere.
func (w *worker) hashMove(ply int, ttMove chess.SearchMove) chess.SearchMove {
	if ply == 0 && !w.rootBest.IsNull() {
		return w.rootBest
	}
	return ttMove
}

// recordCutoff feeds the killers and history; captures are already
// ordered by MVV-LVA, so only quiet moves are learned.
func (w *worker) recordCutoff(move chess.SearchMove, depth, ply int) {
	if !w.board.IsCapture(move) && move.Promotion() == chess.Pawn {
		w.order.recordCutoff(w.board.SideToMove(), move, depth, ply)
	}
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
