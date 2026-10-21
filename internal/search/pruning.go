package search

import (
	"math"

	"github.com/idylicaro/go-chess/pkg/chess"
)

// Selective search: three ways to spend less time on moves that are very
// likely bad, each switchable so lab #44 can measure it by SPRT.

// Pruning selects the selective-search techniques.
type Pruning struct {
	// NullMove: if passing the turn still fails high at a reduced depth,
	// a real move will too, so the node is cut without searching it.
	NullMove bool
	// LateMoveReductions: quiet moves ordered late are searched at a
	// reduced depth first, and fully only if they beat alpha.
	LateMoveReductions bool
	// Futility: near the horizon, quiet moves cannot lift a static
	// evaluation far below alpha up to it, so they are skipped.
	Futility bool
}

// AllPruning enables every technique; it is the engine's default.
var AllPruning = Pruning{NullMove: true, LateMoveReductions: true, Futility: true}

// node is what the pruning decisions need to know about a position.
type node struct {
	depth, ply int
	inCheck    bool
	staticEval int  // valid only when a pruning technique needs it
	futile     bool // quiet moves that do not give check may be skipped
}

// newNode reads the position for the pruning decisions; alpha is the
// node's window, which futility compares against.
func (w *worker) newNode(depth, ply, alpha int) node {
	n := node{depth: depth, ply: ply, inCheck: w.board.InCheck()}
	if !n.inCheck && (w.pruning.NullMove || w.pruning.Futility) {
		n.staticEval = w.evaluate(w.board)
	}
	n.futile = w.isFutile(n, alpha)
	return n
}

const (
	nullMoveMinDepth = 3
	// The null-move search is reduced by base plus one ply per divisor
	// plies of depth: R = 3 at depth 3, 4 at depth 6.
	nullMoveBaseReduction = 2
	nullMoveDepthDivisor  = 3
)

// tryNullMove gives the opponent a free move; if the side to move still
// scores at least beta it cuts the node. It is skipped in check, without
// pieces (where zugzwang makes passing a real advantage) and right after
// another null move.
func (w *worker) tryNullMove(n node, beta int, allowNull bool) (int, bool) {
	if !w.pruning.NullMove || !allowNull || n.ply == 0 || n.inCheck || n.depth < nullMoveMinDepth ||
		n.staticEval < beta || beta >= mateBound || !w.hasPieces() {
		return 0, false
	}
	reduction := nullMoveBaseReduction + n.depth/nullMoveDepthDivisor
	undo := w.board.MakeNullMove()
	w.afterNull = true
	score := -w.negamax(n.depth-1-reduction, n.ply+1, -beta, -beta+1)
	w.board.UnmakeNullMove(undo)
	if score < beta || w.unwinding() {
		return 0, false
	}
	return min(score, mateBound-1), true // a mate found after passing is not proven
}

// hasPieces reports whether the side to move has more than king and pawns.
func (w *worker) hasPieces() bool {
	side := w.board.SideToMove()
	for square := range 64 {
		pieceType, color, occupied := w.board.PieceOn(square)
		if occupied && color == side && pieceType != chess.Pawn && pieceType != chess.King {
			return true
		}
	}
	return false
}

// futilityMargins, by remaining depth, bound what a quiet move can gain.
var futilityMargins = [...]int{0, 150, 350}

// isFutile reports whether the quiet moves of the node can be skipped.
func (w *worker) isFutile(n node, alpha int) bool {
	return w.pruning.Futility && n.ply > 0 && !n.inCheck && n.depth < len(futilityMargins) &&
		alpha > -mateBound && alpha < mateBound && n.staticEval+futilityMargins[n.depth] <= alpha
}

const (
	lmrMinDepth     = 3
	lmrMinMoveIndex = 3 // the hash move, captures and killers come first
)

// lmrReductions[depth][moveIndex] grows with the log of both, the usual
// shape: 0.75 + ln(depth) ln(index) / 2.25.
var lmrReductions = buildLMRReductions()

func buildLMRReductions() *[maxPly + 1][chess.MaxSearchMoves]int {
	table := &[maxPly + 1][chess.MaxSearchMoves]int{}
	for depth := 1; depth <= maxPly; depth++ {
		for index := 1; index < chess.MaxSearchMoves; index++ {
			table[depth][index] = int(0.75 + math.Log(float64(depth))*math.Log(float64(index))/2.25)
		}
	}
	return table
}

// reduction is how many plies less a late quiet move is searched first.
func (w *worker) reduction(n node, moveIndex int, quietWithoutCheck bool) int {
	if !w.pruning.LateMoveReductions || !quietWithoutCheck || n.inCheck || n.depth < lmrMinDepth || moveIndex < lmrMinMoveIndex {
		return 0
	}
	return min(lmrReductions[n.depth][moveIndex], n.depth-2)
}
