package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
	"testing"
)

// newGameFromSquares builds a game holding only the given pieces, keyed by
// algebraic square (e.g. "e1"). Both kings must be present.
func newGameFromSquares(t *testing.T, pieces map[string]IPiece) *Game {
	t.Helper()
	board := newBoard()
	converter := &coordinateConverter{}
	for square, piece := range pieces {
		x, y, err := converter.FromAlgebraic(square)
		if err != nil {
			t.Fatalf("invalid square %q: %v", square, err)
		}
		if err := board.SetPiece(piece, x, y); err != nil {
			t.Fatalf("could not place piece on %q: %v", square, err)
		}
		if piece.GetType() == common.King {
			board.SetKingPosition(piece.GetColor(), x, y)
		}
	}
	return &Game{board: board, currentTurn: common.White}
}

// Regression for #2: K+B+B vs K panicked with index out of range, and the
// same/opposite square-color check for K+B vs K+B was inverted.
func TestIsInsufficientMaterial(t *testing.T) {
	tests := []struct {
		name   string
		pieces map[string]IPiece
		want   bool
	}{
		{
			name:   "only kings",
			pieces: map[string]IPiece{"e1": newKing(common.White), "e8": newKing(common.Black)},
			want:   true,
		},
		{
			name: "king and bishop vs king",
			pieces: map[string]IPiece{
				"e1": newKing(common.White), "e8": newKing(common.Black), "c1": newBishop(common.White),
			},
			want: true,
		},
		{
			name: "king and knight vs king",
			pieces: map[string]IPiece{
				"e1": newKing(common.White), "e8": newKing(common.Black), "g8": newKnight(common.Black),
			},
			want: true,
		},
		{
			name: "bishops on same square color, one per side",
			pieces: map[string]IPiece{
				"e1": newKing(common.White), "e8": newKing(common.Black),
				"c1": newBishop(common.White), "f8": newBishop(common.Black), // both dark squares
			},
			want: true,
		},
		{
			name: "bishops on opposite square colors, one per side",
			pieces: map[string]IPiece{
				"e1": newKing(common.White), "e8": newKing(common.Black),
				"c1": newBishop(common.White), "c8": newBishop(common.Black), // dark vs light
			},
			want: false,
		},
		{
			name: "two bishops on opposite colors, same side",
			pieces: map[string]IPiece{
				"e1": newKing(common.White), "e8": newKing(common.Black),
				"c1": newBishop(common.White), "f1": newBishop(common.White),
			},
			want: false,
		},
		{
			name: "two bishops on same color, same side",
			pieces: map[string]IPiece{
				"e1": newKing(common.White), "e8": newKing(common.Black),
				"c1": newBishop(common.White), "e3": newBishop(common.White),
			},
			want: true,
		},
		{
			name: "two knights vs king",
			pieces: map[string]IPiece{
				"e1": newKing(common.White), "e8": newKing(common.Black),
				"b1": newKnight(common.White), "g1": newKnight(common.White),
			},
			want: false,
		},
		{
			name: "king and pawn vs king",
			pieces: map[string]IPiece{
				"e1": newKing(common.White), "e8": newKing(common.Black), "e2": newPawn(common.White),
			},
			want: false,
		},
		{
			name: "king and rook vs king",
			pieces: map[string]IPiece{
				"e1": newKing(common.White), "e8": newKing(common.Black), "a1": newRook(common.White),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := newGameFromSquares(t, tt.pieces)
			if got := game.isInsufficientMaterial(); got != tt.want {
				t.Errorf("isInsufficientMaterial() = %v, want %v", got, tt.want)
			}
		})
	}
}
