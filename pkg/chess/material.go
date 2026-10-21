package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
)

// canPossiblyMate reports whether color has enough material for some legal
// sequence of moves to end in checkmate against the opponent. A lone minor
// piece qualifies only if the opponent has something besides a king, which
// could block the king's escape squares.
func (g *Game) canPossiblyMate(color common.Color) bool {
	minors, opponentHasPieces := 0, false
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			piece, _ := g.board.GetPiece(x, y)
			if piece == nil || piece.GetType() == common.King {
				continue
			}
			if piece.GetColor() != color {
				opponentHasPieces = true
				continue
			}
			switch piece.GetType() {
			case common.Bishop, common.Knight:
				minors++
			default:
				return true
			}
		}
	}
	return minors >= 2 || (minors == 1 && opponentHasPieces)
}

// placedPiece is a non-king piece together with the square it stands on.
type placedPiece struct {
	pieceType common.PieceType
	position  Position
}

// isInsufficientMaterial reports whether neither side can ever deliver mate.
// Covers: bare kings, a single knight, and any number of bishops (either side)
// all standing on squares of the same color — which includes K+B vs K and
// K+B vs K+B with same-colored bishops (#2).
//
//	game.isInsufficientMaterial() // true for K+B(c1) vs K+B(f8)
func (g *Game) isInsufficientMaterial() bool {
	material := g.collectNonKingPieces()
	if len(material) == 0 {
		return true
	}
	if len(material) == 1 && material[0].pieceType == common.Knight {
		return true
	}
	return allBishopsOnSameSquareColor(material)
}

// collectNonKingPieces lists every piece on the board except the kings.
func (g *Game) collectNonKingPieces() []placedPiece {
	var material []placedPiece
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			piece, _ := g.board.GetPiece(x, y)
			if piece == nil || piece.GetType() == common.King {
				continue
			}
			material = append(material, placedPiece{piece.GetType(), Position{x, y}})
		}
	}
	return material
}

// allBishopsOnSameSquareColor is true when material is non-empty, made only of
// bishops, and every bishop stands on the same square color.
func allBishopsOnSameSquareColor(material []placedPiece) bool {
	if len(material) == 0 {
		return false
	}
	firstSquareColor := squareColor(material[0].position)
	for _, placed := range material {
		if placed.pieceType != common.Bishop || squareColor(placed.position) != firstSquareColor {
			return false
		}
	}
	return true
}

// squareColor returns 0 for light squares and 1 for dark squares.
func squareColor(position Position) int {
	return (position.X + position.Y) % 2
}
