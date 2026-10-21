package chess

import (
	"fmt"
	"github.com/idylicaro/go-chess/internal/common"
)

// coordinateConverter handles conversion between different coordinate systems
type coordinateConverter struct{}

// ToAlgebraic converts internal coordinates (0-7, 0-7) to algebraic notation (a1-h8)
// Internal board: (0,0) = top-left, (7,7) = bottom-right
// Algebraic: a1 = bottom-left, h8 = top-right
func (cc *coordinateConverter) ToAlgebraic(x, y int) (string, error) {
	if x < 0 || x > 7 || y < 0 || y > 7 {
		return "", fmt.Errorf("invalid coordinates (%d, %d): expected both between 0 and 7", x, y)
	}

	// Convert x to file (a-h)
	file := string(rune('a' + x))

	// Convert y to rank (1-8) - internal y=0 is rank 8, y=7 is rank 1
	rank := string(rune('0' + (8 - y)))

	return file + rank, nil
}

// FromAlgebraic converts algebraic notation (e.g., "e4") to internal coordinates
func (cc *coordinateConverter) FromAlgebraic(square string) (x, y int, err error) {
	if len(square) != 2 {
		return 0, 0, fmt.Errorf("invalid square %q: expected a file a-h and a rank 1-8, like \"e4\"", square)
	}

	file := square[0]
	rank := square[1]

	// Convert file (a-h) to x coordinate (0-7)
	if file < 'a' || file > 'h' {
		return 0, 0, fmt.Errorf("invalid file %q in square %q: expected a letter from a to h", file, square)
	}
	x = int(file - 'a')

	// Convert rank (1-8) to y coordinate (7-0)
	if rank < '1' || rank > '8' {
		return 0, 0, fmt.Errorf("invalid rank %q in square %q: expected a digit from 1 to 8", rank, square)
	}
	y = 8 - int(rank-'0')

	return x, y, nil
}

// GetPieceSymbol returns the algebraic symbol for a piece type
// Pawn = "" (no symbol), Knight = "N", Bishop = "B", etc.
func (cc *coordinateConverter) GetPieceSymbol(pieceType common.PieceType) string {
	switch pieceType {
	case common.Pawn:
		return ""
	case common.Knight:
		return "N"
	case common.Bishop:
		return "B"
	case common.Rook:
		return "R"
	case common.Queen:
		return "Q"
	case common.King:
		return "K"
	default:
		return "?"
	}
}
