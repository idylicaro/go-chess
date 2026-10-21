package chess

import (
	"fmt"
	"github.com/idylicaro/go-chess/internal/common"
	"github.com/idylicaro/go-chess/internal/uci"
)

// promotionChoices are the pieces a pawn reaching the last rank may become.
var promotionChoices = []common.PieceType{common.Queen, common.Rook, common.Bishop, common.Knight}

// Perft counts the leaf nodes of the legal move tree to the given depth.
// It is the standard correctness check for move generation: compare the
// result with published reference counts (#8). It searches a copy with
// make/unmake, so the game itself is left untouched (#12).
//
//	NewGame().Perft(3) // 8902
func (g *Game) Perft(depth int) int {
	return g.clone().perft(depth)
}

// perft plays and takes back each move on g itself; the move lists live in
// stack buffers, so a search does not allocate.
func (g *Game) perft(depth int) int {
	if depth <= 0 {
		return 1
	}
	var buffer [maxMovesPerPosition]SearchMove
	moves := g.appendLegalMoves(buffer[:0])
	if depth == 1 {
		return len(moves)
	}
	nodes := 0
	for _, move := range moves {
		undo := g.makeMove(move)
		nodes += g.perft(depth - 1)
		g.unmakeMove(move, undo)
	}
	return nodes
}

// Divide is Perft split by first move, keyed by UCI notation. Diffing it
// against a reference engine's "go perft" output pinpoints a wrong branch.
//
//	NewGame().Divide(2)["e2e4"] // 20
func (g *Game) Divide(depth int) map[string]int {
	counts := make(map[string]int)
	if depth <= 0 {
		return counts
	}
	search := g.clone()
	for _, move := range search.appendLegalMoves(nil) {
		undo := search.makeMove(move)
		counts[move.toUCI().String()] = search.perft(depth - 1)
		search.unmakeMove(move, undo)
	}
	return counts
}

// childAfter returns a copy of the game with move played.
func (g *Game) childAfter(move uci.Move) *Game {
	child := g.clone()
	if err := child.play(move); err != nil {
		panic(fmt.Sprintf("perft: legal move %s rejected in position %q: %v", move, g.FEN(), err))
	}
	return child
}

// clone copies the position. The copy has no clock and an empty history:
// it is meant for search, not for continuing the original game.
func (g *Game) clone() *Game {
	child := *g
	board := *g.board
	child.board = &board
	child.gameTimer = nil
	child.positionHistory = nil
	child.moveHistory = nil
	child.undoStack = nil
	child.drawOfferedBy = nil
	if g.enPassantTarget != nil {
		target := *g.enPassantTarget
		child.enPassantTarget = &target
	}
	return &child
}

// legalUCIMoves lists every legal move of the side to move, expanding each
// promotion into its four piece choices.
func (g *Game) legalUCIMoves() []uci.Move {
	var buffer [maxMovesPerPosition]SearchMove
	generated := g.appendLegalMoves(buffer[:0])
	moves := make([]uci.Move, len(generated))
	for i, move := range generated {
		moves[i] = move.toUCI()
	}
	return moves
}
