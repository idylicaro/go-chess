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
// result with published reference counts (#8). It panics if a move it
// generated as legal is then rejected, since that is a move-generator bug.
//
//	NewGame().Perft(3) // 8902
func (g *Game) Perft(depth int) int {
	if depth <= 0 {
		return 1
	}
	moves := g.legalUCIMoves()
	if depth == 1 {
		return len(moves)
	}
	nodes := 0
	for _, move := range moves {
		nodes += g.childAfter(move).Perft(depth - 1)
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
	for _, move := range g.legalUCIMoves() {
		counts[move.String()] = g.childAfter(move).Perft(depth - 1)
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
	var moves []uci.Move
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			moves = append(moves, g.legalUCIMovesFrom(x, y)...)
		}
	}
	return moves
}

func (g *Game) legalUCIMovesFrom(fromX, fromY int) []uci.Move {
	piece, _ := g.board.GetPiece(fromX, fromY)
	if piece == nil || piece.GetColor() != g.currentTurn {
		return nil
	}
	destinations, _ := g.getLegalMovesForPiece(fromX, fromY)
	var moves []uci.Move
	for _, to := range destinations {
		move := uci.Move{FromX: fromX, FromY: fromY, ToX: to.X, ToY: to.Y}
		if !isPromotionMove(piece, move) {
			moves = append(moves, move)
			continue
		}
		for _, promotion := range promotionChoices {
			move.Promotion = promotion
			moves = append(moves, move)
		}
	}
	return moves
}
