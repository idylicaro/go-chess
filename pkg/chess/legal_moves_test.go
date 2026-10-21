package chess

import (
	"fmt"
	"github.com/idylicaro/go-chess/internal/common"
	"github.com/idylicaro/go-chess/internal/uci"
	"testing"
)

func mustParseUCI(t *testing.T, notation string) uci.Move {
	t.Helper()
	move, err := uci.ParseMove(notation)
	if err != nil {
		t.Fatalf("uci.ParseMove(%q): %v", notation, err)
	}
	return move
}

// legalTargets returns the algebraic squares the piece on from can legally move to.
func legalTargets(t *testing.T, game *Game, from string) map[string]bool {
	t.Helper()
	converter := &coordinateConverter{}
	x, y, err := converter.FromAlgebraic(from)
	if err != nil {
		t.Fatalf("invalid square %q: %v", from, err)
	}
	moves, err := game.getLegalMovesForPiece(x, y)
	if err != nil {
		t.Fatalf("getLegalMovesForPiece(%s): %v", from, err)
	}
	targets := make(map[string]bool, len(moves))
	for _, move := range moves {
		square, _ := converter.ToAlgebraic(move.X, move.Y)
		targets[square] = true
	}
	return targets
}

func assertLegal(t *testing.T, targets map[string]bool, square string, want bool) {
	t.Helper()
	if targets[square] != want {
		t.Errorf("move to %s legal = %v, want %v (legal targets: %v)", square, targets[square], want, targets)
	}
}

// Regression for #3: castling through a square attacked by a pawn was allowed.
func TestCastling_ThroughPawnAttackedSquare(t *testing.T) {
	game := mustParseFEN(t, "k7/8/8/8/8/8/4p3/4K2R w K - 0 1") // black pawn e2 attacks d1 and f1
	assertLegal(t, legalTargets(t, game, "e1"), "g1", false)
}

func TestCastling_AllowedWhenPathIsSafe(t *testing.T) {
	game := mustParseFEN(t, "k7/p7/8/8/8/8/8/4K2R w K - 0 1")
	assertLegal(t, legalTargets(t, game, "e1"), "g1", true)
}

func TestKing_CannotCaptureDefendedPiece(t *testing.T) {
	game := newGameFromSquares(t, map[string]IPiece{
		"e1": newKing(common.White), "a8": newKing(common.Black),
		"e2": newQueen(common.Black), "e8": newRook(common.Black),
	})
	assertLegal(t, legalTargets(t, game, "e1"), "e2", false)
}

// Regression for #4: the legality simulation did not remove the pawn captured
// en passant, so a capture exposing the own king along the rank was allowed.
func TestEnPassant_RejectedWhenItExposesKing(t *testing.T) {
	game := newGameFromSquares(t, map[string]IPiece{
		"a5": newKing(common.White), "e5": newPawn(common.White),
		"d5": newPawn(common.Black), "h5": newRook(common.Black), "h8": newKing(common.Black),
	})
	game.enPassantTarget = &Position{X: 3, Y: 2} // d6, after ...d7-d5
	targets := legalTargets(t, game, "e5")
	assertLegal(t, targets, "d6", false)
	assertLegal(t, targets, "e6", true)
}

func TestEnPassant_AllowedAndRemovesCapturedPawn(t *testing.T) {
	game := newGameFromSquares(t, map[string]IPiece{
		"e1": newKing(common.White), "e5": newPawn(common.White),
		"d5": newPawn(common.Black), "h8": newKing(common.Black),
	})
	game.enPassantTarget = &Position{X: 3, Y: 2} // d6
	assertLegal(t, legalTargets(t, game, "e5"), "d6", true)

	if err := game.Move(mustParseUCI(t, "e5d6")); err != nil {
		t.Fatalf("e5d6 en passant: %v", err)
	}
	if captured, _ := game.board.GetPiece(3, 3); captured != nil {
		t.Errorf("pawn on d5 should have been captured en passant, found %v", captured.GetType())
	}
}

// getLegalMovesForPiece is a test and benchmark helper: it lists the squares
// the piece on (fromX, fromY) can legally reach; a promotion counts once,
// whatever the piece.
func (g *Game) getLegalMovesForPiece(fromX, fromY int) ([]Position, error) {
	piece, err := g.board.GetPiece(fromX, fromY)
	if err != nil {
		return nil, err
	}
	if piece == nil || piece.GetColor() != g.currentTurn {
		return nil, fmt.Errorf("the square %s holds no piece of %v, whose turn it is", squareName(fromX, fromY), g.currentTurn)
	}
	from := int8(squareIndex(fromX, fromY))
	var destinations []Position
	for _, move := range g.appendLegalMoves(nil) {
		if move.from == from && (move.promotion == common.Pawn || move.promotion == common.Queen) {
			destinations = append(destinations, squarePositions[move.to])
		}
	}
	return destinations, nil
}
