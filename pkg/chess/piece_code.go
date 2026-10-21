package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
)

// pieceCode packs a piece into one byte so a board is a plain [64] array:
// copying it is a value copy with no allocation, unlike the old [8][8]IPiece
// that held one heap pointer per piece (#12). The low three bits are the
// piece type plus one (so zero is an empty square) and bit 3 is the color.
type pieceCode uint8

const noPiece pieceCode = 0

const pieceCodeColorShift = 3

// codeOf packs a piece type and color.
//
//	codeOf(common.Knight, common.Black).pieceType() // common.Knight
func codeOf(pieceType common.PieceType, color common.Color) pieceCode {
	return pieceCode(color)<<pieceCodeColorShift | pieceCode(pieceType+1)
}

func (p pieceCode) pieceType() common.PieceType {
	return common.PieceType(p&7) - 1
}

func (p pieceCode) color() common.Color {
	return common.Color(p >> pieceCodeColorShift)
}

// isColor is false for an empty square.
func (p pieceCode) isColor(color common.Color) bool {
	return p != noPiece && p.color() == color
}

// pieceViews holds one shared, never-mutated IPiece per code, so reading a
// square through IBoard.GetPiece does not allocate.
var pieceViews = buildPieceViews()

func buildPieceViews() [16]IPiece {
	var views [16]IPiece
	for _, color := range []common.Color{common.White, common.Black} {
		for pieceType := common.Pawn; pieceType <= common.King; pieceType++ {
			views[codeOf(pieceType, color)] = newPiece(pieceType, color)
		}
	}
	return views
}

// codeOfPiece packs an IPiece; nil packs to noPiece.
func codeOfPiece(piece IPiece) pieceCode {
	if piece == nil {
		return noPiece
	}
	return codeOf(piece.GetType(), piece.GetColor())
}

// squareIndex maps internal coordinates to 0..63, row by row from rank 8.
func squareIndex(x, y int) int {
	return y*8 + x
}
