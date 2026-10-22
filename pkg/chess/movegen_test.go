package chess

import (
	"github.com/idylicaro/go-chess/internal/uci"
	"testing"
)

func TestGenMove_ToUCI(t *testing.T) {
	move := SearchMove{from: int8(squareIndex(4, 1)), to: int8(squareIndex(4, 0)), promotion: PromoteToRook}
	if got := move.toUCI(); got != (uci.Move{FromX: 4, FromY: 1, ToX: 4, ToY: 0, Promotion: PromoteToRook}) || got.String() != "e7e8r" {
		t.Errorf("toUCI() = %v (%s), want e7e8r", got, got)
	}
}

func TestHasLegalMove(t *testing.T) {
	cases := map[string]bool{
		StartingFEN:                      true,
		"7k/5Q2/6K1/8/8/8/8/8 b - - 0 1": false, // stalemate
		"7k/6Q1/6K1/8/8/8/8/8 b - - 0 1": false, // checkmate
		"7k/8/6K1/8/8/8/8/6Q1 b - - 0 1": true,
	}
	for fen, want := range cases {
		if got := mustParseFEN(t, fen).hasLegalMove(); got != want {
			t.Errorf("hasLegalMove(%q) = %v, want %v", fen, got, want)
		}
	}
}

// TestAppendLegalMoves_KeepsPrefix checks the in-place filter leaves moves
// already in the slice alone.
func TestAppendLegalMoves_KeepsPrefix(t *testing.T) {
	sentinel := SearchMove{from: 1, to: 2}
	moves := NewGame().appendLegalMoves([]SearchMove{sentinel})
	if len(moves) != 21 || moves[0] != sentinel {
		t.Errorf("appendLegalMoves kept %d moves starting with %v, want 21 starting with %v", len(moves), moves[0], sentinel)
	}
}

func TestIsEnPassantSquare_RequiresTheEnemyPawn(t *testing.T) {
	withPawn := mustParseFEN(t, "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1")
	withoutPawn := mustParseFEN(t, "4k3/8/8/4P3/8/8/8/4K3 w - d6 0 1")
	d6 := squareIndex(3, 2)
	if !withPawn.isEnPassantSquare(d6) || withoutPawn.isEnPassantSquare(d6) {
		t.Error("d6 must be an en passant square only when a black pawn stands on d5")
	}
}
