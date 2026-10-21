package search

import (
	"github.com/idylicaro/go-chess/pkg/chess"
)

// quiesce searches captures and promotions until the position is quiet, so
// a leaf is never scored in the middle of an exchange (the horizon effect,
// #39). The side to move may "stand pat" on the static evaluation instead
// of capturing, except in check, where every evasion is searched.
func (w *worker) quiesce(ply, alpha, beta int) int {
	w.pv.clear(ply)
	if w.visit() {
		return drawScore // unwinding; the caller discards the result
	}
	if w.board.InCheck() {
		return w.quiesceEvasions(ply, alpha, beta)
	}
	standPat := w.evaluate(w.board)
	if standPat >= beta || ply >= maxPly {
		return standPat
	}
	var buffer [chess.MaxSearchMoves]chess.SearchMove
	picker := w.order.newMovePicker(w.board, w.board.GenerateCaptures(buffer[:0]), chess.NullSearchMove, ply)
	best, _ := w.quiesceMoves(&picker, ply, max(alpha, standPat), beta)
	return max(best, standPat)
}

// quiesceEvasions searches every move out of check; with none it is mate.
func (w *worker) quiesceEvasions(ply, alpha, beta int) int {
	if ply >= maxPly {
		return w.evaluate(w.board)
	}
	var buffer [chess.MaxSearchMoves]chess.SearchMove
	picker := w.order.newMovePicker(w.board, w.board.GenerateMoves(buffer[:0]), chess.NullSearchMove, ply)
	best, legalMoves := w.quiesceMoves(&picker, ply, alpha, beta)
	if legalMoves == 0 {
		return matedIn(ply)
	}
	return best
}

// quiesceMoves searches the picker's moves and returns the best score and
// how many were legal.
func (w *worker) quiesceMoves(picker *movePicker, ply, alpha, beta int) (best, legalMoves int) {
	best = -infinity
	for move, ok := picker.pick(); ok; move, ok = picker.pick() {
		undo, legal := w.board.MakeMove(move)
		if !legal {
			continue
		}
		legalMoves++
		score := -w.quiesce(ply+1, -beta, -alpha)
		w.board.UnmakeMove(move, undo)
		best = max(best, score)
		alpha = max(alpha, score)
		if alpha >= beta {
			break
		}
	}
	return best, legalMoves
}
