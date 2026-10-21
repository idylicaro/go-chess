package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
)

// pawnCaptureSides are the column steps of a pawn's two capture diagonals.
var pawnCaptureSides = [2]int{-1, 1}

// appendPawnMoves adds pushes, the double push from the home rank, captures,
// en passant, and the four promotions of any move to the last rank.
func (g *Game) appendPawnMoves(moves []SearchMove, from int) []SearchMove {
	x, y := from%8, from/8
	forward := pawnForward(g.currentTurn)
	if !onBoard(x, y+forward) {
		return moves // a pawn on its last rank (possible from FEN) has no moves
	}
	moves = g.appendPawnPushes(moves, from, x, y, forward)
	for _, side := range pawnCaptureSides {
		if to, ok := g.pawnCaptureTarget(x+side, y+forward); ok {
			moves = appendPawnMove(moves, from, to)
		}
	}
	return moves
}

func (g *Game) appendPawnPushes(moves []SearchMove, from, x, y, forward int) []SearchMove {
	one := squareIndex(x, y+forward)
	if g.board.squares[one] != noPiece {
		return moves
	}
	moves = appendPawnMove(moves, from, one)
	// A pawn still on its home rank has never moved, so it may advance two
	// squares. Derived from the rank (not a flag) so FEN positions work (#7).
	if y != pawnHomeRow(g.currentTurn) {
		return moves
	}
	if two := squareIndex(x, y+2*forward); g.board.squares[two] == noPiece {
		moves = append(moves, SearchMove{from: int8(from), to: int8(two)})
	}
	return moves
}

// pawnCaptureTarget reports whether a pawn of the side to move may capture
// on (x, y): an enemy piece stands there, or it is a valid en passant square.
func (g *Game) pawnCaptureTarget(x, y int) (int, bool) {
	if !onBoard(x, y) {
		return 0, false
	}
	to := squareIndex(x, y)
	return to, g.board.squares[to].isColor(opposite(g.currentTurn)) || g.isEnPassantSquare(to)
}

// isEnPassantSquare also requires the enemy pawn behind the target, since a
// FEN may name an en passant square with no pawn to capture.
func (g *Game) isEnPassantSquare(to int) bool {
	target := g.enPassantTarget
	if target == nil || squareIndex(target.X, target.Y) != to {
		return false
	}
	victim := to - 8*pawnForward(g.currentTurn)
	return g.board.squares[victim] == codeOf(common.Pawn, opposite(g.currentTurn))
}

// appendPawnMove adds the move, or one move per piece when it promotes.
func appendPawnMove(moves []SearchMove, from, to int) []SearchMove {
	if row := to / 8; row != 0 && row != 7 {
		return append(moves, SearchMove{from: int8(from), to: int8(to)})
	}
	for _, promotion := range promotionChoices {
		moves = append(moves, SearchMove{from: int8(from), to: int8(to), promotion: promotion})
	}
	return moves
}
