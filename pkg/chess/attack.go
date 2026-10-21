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
	target := Position{X: x, Y: y}
	return b.isAttackedByPawn(target, byColor) ||
		b.isAttackedByStepper(target, byColor, knightOffsets, common.Knight) ||
		b.isAttackedByStepper(target, byColor, kingOffsets, common.King) ||
		b.isAttackedBySlider(target, byColor, orthogonalDirections, common.Rook) ||
		b.isAttackedBySlider(target, byColor, diagonalDirections, common.Bishop)
}

// isAttackedByPawn checks the two squares a byColor pawn would attack target from.
// White pawns advance towards y=0, so they attack from the row below (y+1).
func (b *Board) isAttackedByPawn(target Position, byColor common.Color) bool {
	attackerY := target.Y - 1
	if byColor == common.White {
		attackerY = target.Y + 1
	}
	return b.hasPieceAt(Position{X: target.X - 1, Y: attackerY}, byColor, common.Pawn) ||
		b.hasPieceAt(Position{X: target.X + 1, Y: attackerY}, byColor, common.Pawn)
}

// isAttackedByStepper checks single-jump attackers (knight, king) at each offset.
func (b *Board) isAttackedByStepper(target Position, byColor common.Color, offsets []Position, stepper common.PieceType) bool {
	for _, offset := range offsets {
		if b.hasPieceAt(Position{X: target.X + offset.X, Y: target.Y + offset.Y}, byColor, stepper) {
			return true
		}
	}
	return false
}

// isAttackedBySlider walks each direction from target until the first piece;
// it attacks if it is a byColor slider of the given type or a queen.
func (b *Board) isAttackedBySlider(target Position, byColor common.Color, directions []Position, slider common.PieceType) bool {
	for _, direction := range directions {
		blocker, found := b.firstPieceInDirection(target, direction)
		if found && b.hasPieceAt(blocker, byColor, slider, common.Queen) {
			return true
		}
	}
	return false
}

// firstPieceInDirection returns the position of the nearest piece from start
// (exclusive) along direction, or found=false if the ray leaves the board.
func (b *Board) firstPieceInDirection(start, direction Position) (Position, bool) {
	current := Position{X: start.X + direction.X, Y: start.Y + direction.Y}
	for b.IsValidCoordinates(current.X, current.Y) == nil {
		if piece, _ := b.GetPiece(current.X, current.Y); piece != nil {
			return current, true
		}
		current = Position{X: current.X + direction.X, Y: current.Y + direction.Y}
	}
	return Position{}, false
}

// hasPieceAt is true when position is on the board and holds a piece of color
// whose type is one of pieceTypes.
func (b *Board) hasPieceAt(position Position, color common.Color, pieceTypes ...common.PieceType) bool {
	piece, err := b.GetPiece(position.X, position.Y)
	if err != nil || piece == nil || piece.GetColor() != color {
		return false
	}
	for _, pieceType := range pieceTypes {
		if piece.GetType() == pieceType {
			return true
		}
	}
	return false
}
