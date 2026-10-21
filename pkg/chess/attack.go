package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
)

// Attack detection is kept separate from move generation on purpose (#3):
// a pawn attacks diagonals it cannot move to, never the square in front of
// it, and a king's castling destinations are moves, not attacks.

var knightOffsets = []Position{
	{X: -2, Y: -1}, {X: -1, Y: -2}, {X: 1, Y: -2}, {X: 2, Y: -1},
	{X: 2, Y: 1}, {X: 1, Y: 2}, {X: -1, Y: 2}, {X: -2, Y: 1},
}

var orthogonalDirections = []Position{{X: 0, Y: -1}, {X: -1, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}}

var diagonalDirections = []Position{{X: 1, Y: -1}, {X: -1, Y: -1}, {X: 1, Y: 1}, {X: -1, Y: 1}}

var kingOffsets = append(append([]Position{}, orthogonalDirections...), diagonalDirections...)

// IsSquareAttacked reports whether any piece of byColor attacks (x, y),
// regardless of what occupies that square — so it also answers "is this
// piece defended?".
//
//	board.IsSquareAttacked(5, 7, common.Black) // is f1 attacked by Black?
func (b *Board) IsSquareAttacked(x, y int, byColor common.Color) bool {
	return b.isAttacked(squareIndex(x, y), byColor)
}

// isAttacked is IsSquareAttacked on a square index, for the generator.
func (b *Board) isAttacked(target int, byColor common.Color) bool {
	queen := codeOf(common.Queen, byColor)
	return b.isAttackedByPawn(target, byColor) ||
		b.anyCodeOn(moveTables.knight[target], codeOf(common.Knight, byColor)) ||
		b.anyCodeOn(moveTables.king[target], codeOf(common.King, byColor)) ||
		b.isAttackedBySlider(orthogonalRays(target), codeOf(common.Rook, byColor), queen) ||
		b.isAttackedBySlider(diagonalRays(target), codeOf(common.Bishop, byColor), queen)
}

// isAttackedByPawn checks the two squares a byColor pawn would attack target from.
// White pawns advance towards y=0, so they attack from the row below (y+1).
func (b *Board) isAttackedByPawn(target int, byColor common.Color) bool {
	x, attackerY := target%8, target/8-pawnForward(byColor)
	pawn := codeOf(common.Pawn, byColor)
	return (onBoard(x-1, attackerY) && b.squares[squareIndex(x-1, attackerY)] == pawn) ||
		(onBoard(x+1, attackerY) && b.squares[squareIndex(x+1, attackerY)] == pawn)
}

// anyCodeOn reports whether code stands on any of squares.
func (b *Board) anyCodeOn(squares []int8, code pieceCode) bool {
	for _, square := range squares {
		if b.squares[square] == code {
			return true
		}
	}
	return false
}

// isAttackedBySlider looks at the first piece on each ray; it attacks if it
// is the slider or the queen of the attacking side.
func (b *Board) isAttackedBySlider(rays [][]int8, slider, queen pieceCode) bool {
	for _, ray := range rays {
		if blocker := b.firstCodeOnRay(ray); blocker == slider || blocker == queen {
			return true
		}
	}
	return false
}

// firstCodeOnRay returns the nearest piece along ray, or noPiece.
func (b *Board) firstCodeOnRay(ray []int8) pieceCode {
	for _, square := range ray {
		if code := b.squares[square]; code != noPiece {
			return code
		}
	}
	return noPiece
}

// pawnForward is the row step of a pawn of color: White moves towards y=0.
func pawnForward(color common.Color) int {
	if color == common.White {
		return -1
	}
	return 1
}
