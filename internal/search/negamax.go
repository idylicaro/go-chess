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
	pruning Pruning
	// afterNull is set while searching the reply to a null move.
	afterNull bool
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
	allowNull := !w.afterNull // never two null moves in a row
	w.afterNull = false
	w.pv.clear(ply)
	if score, done := w.leafScore(depth, ply, alpha, beta); done {
		return score
	}
	entry, found := w.tt.probe(w.board.Hash())
	if score, settled := entry.cutoffScore(depth, ply, alpha, beta); found && settled && ply > 0 {
		return score
	}
	n := w.newNode(depth, ply, alpha)
	if score, cut := w.tryNullMove(n, beta, allowNull); cut {
		return score
	}
	score, best := w.searchMoves(n, alpha, beta, entry.move)
	w.storeResult(depth, ply, score, best, boundFor(score, alpha, beta))
	return score
}

// leafScore ends the node early: the search is unwinding, the position is
// drawn, too deep, or at the horizon, where quiescence takes over.
func (w *worker) leafScore(depth, ply, alpha, beta int) (int, bool) {
	switch {
	case w.visit():
		return drawScore, true // unwinding; the caller discards the result
	case ply > 0 && w.isDraw():
		return drawScore, true
	case ply >= maxPly:
		return w.evaluate(w.board), true
	case depth <= 0:
		return w.quiesce(ply, alpha, beta), true
	}
	return 0, false
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
func (w *worker) searchMoves(n node, alpha, beta int, ttMove chess.SearchMove) (int, chess.SearchMove) {
	var buffer [chess.MaxSearchMoves]chess.SearchMove
	picker := w.order.newMovePicker(w.board, w.board.GenerateMoves(buffer[:0]), w.hashMove(n.ply, ttMove), n.ply)
	best, bestMove, legalMoves := -infinity, ttMove, 0
	for move, ok := picker.pick(); ok; move, ok = picker.pick() {
		score, outcome := w.searchChild(n, move, legalMoves, alpha, beta)
		if outcome == childIllegal {
			continue
		}
		legalMoves, best = legalMoves+1, max(best, score)
		if outcome == childSearched && score > alpha {
			alpha, bestMove = score, move
			w.pv.update(n.ply, move)
		}
		if alpha >= beta {
			w.recordCutoff(move, n.depth, n.ply)
			break
		}
	}
	if legalMoves == 0 {
		return w.noMoveScore(n.ply), chess.NullSearchMove
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

// childOutcome says what searchChild did with a move.
type childOutcome int

const (
	childSearched childOutcome = iota
	childPruned                // skipped by futility pruning; the score is a bound
	childIllegal               // leaves the own king in check
)

// searchChild plays move, the moveIndex-th legal one at the node, and
// scores the position after it, reduced or pruned when the move is quiet
// and late (#44).
func (w *worker) searchChild(n node, move chess.SearchMove, moveIndex, alpha, beta int) (int, childOutcome) {
	quiet := w.isQuiet(move)
	undo, legal := w.board.MakeMove(move)
	if !legal {
		return 0, childIllegal
	}
	defer w.board.UnmakeMove(move, undo)
	givesCheck := w.board.InCheck()
	if n.futile && quiet && !givesCheck && moveIndex > 0 {
		return n.staticEval + futilityMargins[n.depth], childPruned
	}
	return w.searchAfterMove(n, alpha, beta, w.reduction(n, moveIndex, quiet && !givesCheck)), childSearched
}

// searchAfterMove searches the child, first reduced with a null window when
// reduction > 0, and again at full depth only if that beats alpha.
func (w *worker) searchAfterMove(n node, alpha, beta, reduction int) int {
	if reduction > 0 {
		score := -w.negamax(n.depth-1-reduction, n.ply+1, -alpha-1, -alpha)
		if score <= alpha || w.unwinding() {
			return score
		}
	}
	return -w.negamax(n.depth-1, n.ply+1, -beta, -alpha)
}

// isQuiet is true for a move that neither captures nor promotes.
func (w *worker) isQuiet(move chess.SearchMove) bool {
	return move.Promotion() == chess.Pawn && !w.board.IsCapture(move)
}

// noMoveScore is checkmate or stalemate.
func (w *worker) noMoveScore(ply int) int {
	if w.board.InCheck() {
		return matedIn(ply)
	}
	return drawScore
}
