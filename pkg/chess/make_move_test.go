package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
	"testing"
)

// TestMakeUnmake_RestoresEveryPosition plays and takes back every legal move
// two plies deep in the reference positions, which between them cover
// castling, en passant and promotion, and requires the exact FEN back.
func TestMakeUnmake_RestoresEveryPosition(t *testing.T) {
	for name, fen := range perftReferenceFENs {
		game := mustParseFEN(t, fen)
		assertMakeUnmakeRestores(t, name, game, 2)
	}
}

func assertMakeUnmakeRestores(t *testing.T, name string, game *Game, depth int) {
	t.Helper()
	if depth == 0 {
		return
	}
	before, kings := game.FEN(), game.board.kings
	for _, move := range game.appendLegalMoves(nil) {
		undo := game.makeMove(move)
		assertMakeUnmakeRestores(t, name, game, depth-1)
		game.unmakeMove(move, undo)
		if after := game.FEN(); after != before || game.board.kings != kings {
			t.Fatalf("%s: make/unmake %s turned %q into %q", name, move.toUCI(), before, after)
		}
	}
}

func TestMakeMove_EnPassantRemovesTheCapturedPawn(t *testing.T) {
	game := mustParseFEN(t, "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1")
	game.makeMove(genMove{from: int8(squareIndex(4, 3)), to: int8(squareIndex(3, 2))})
	if got, want := game.FEN(), "4k3/8/3P4/8/8/8/8/4K3 b - - 0 1"; got != want {
		t.Errorf("after exd6 e.p. FEN = %q, want %q", got, want)
	}
}

func TestMakeMove_CastlingMovesTheRookAndDropsRights(t *testing.T) {
	game := mustParseFEN(t, "r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 3 10")
	game.makeMove(genMove{from: int8(squareIndex(4, 0)), to: int8(squareIndex(2, 0))})
	if got, want := game.FEN(), "2kr3r/8/8/8/8/8/8/R3K2R w KQ - 4 11"; got != want {
		t.Errorf("after O-O-O FEN = %q, want %q", got, want)
	}
}

func TestMakeMove_PromotionPlacesTheChosenPiece(t *testing.T) {
	game := mustParseFEN(t, "1n2k3/P7/8/8/8/8/8/4K3 w - - 0 1")
	game.makeMove(genMove{from: int8(squareIndex(0, 1)), to: int8(squareIndex(1, 0)), promotion: common.Knight})
	if got, want := game.FEN(), "1N2k3/8/8/8/8/8/8/4K3 b - - 0 1"; got != want {
		t.Errorf("after axb8=N FEN = %q, want %q", got, want)
	}
}

func TestCastlingRookSquares(t *testing.T) {
	cases := []struct {
		move             genMove
		rookFrom, rookTo int
		castles          bool
	}{
		{genMove{from: 60, to: 62}, 63, 61, true}, // e1g1: h1 -> f1
		{genMove{from: 4, to: 2}, 0, 3, true},     // e8c8: a8 -> d8
		{genMove{from: 60, to: 61}, 0, 0, false},  // e1f1
	}
	for _, tc := range cases {
		from, to, castles := castlingRookSquares(tc.move)
		if from != tc.rookFrom || to != tc.rookTo || castles != tc.castles {
			t.Errorf("castlingRookSquares(%v) = (%d, %d, %v), want (%d, %d, %v)", tc.move, from, to, castles, tc.rookFrom, tc.rookTo, tc.castles)
		}
	}
}

func TestEnPassantTargetAfter_OnlyForADoublePawnPush(t *testing.T) {
	pawn, knight := codeOf(common.Pawn, common.White), codeOf(common.Knight, common.White)
	if target := enPassantTargetAfter(pawn, genMove{from: 52, to: 36}); target == nil || *target != (Position{X: 4, Y: 5}) {
		t.Errorf("e2e4 en passant target = %v, want e3 (4,5)", target)
	}
	if target := enPassantTargetAfter(pawn, genMove{from: 52, to: 44}); target != nil {
		t.Errorf("e2e3 en passant target = %v, want nil", target)
	}
	if target := enPassantTargetAfter(knight, genMove{from: 62, to: 45}); target != nil {
		t.Errorf("Ng1f3 en passant target = %v, want nil", target)
	}
}
