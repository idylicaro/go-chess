package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
	"github.com/idylicaro/go-chess/internal/uci"
)

// maxMovesPerPosition bounds a move list: the most legal moves known in any
// position is 218, so a fixed buffer of this size never grows.
const maxMovesPerPosition = 256

// SearchMove is a move in the generator's own form, with square indexes so
// make/unmake needs no conversion. Promotion is common.Pawn (the zero value)
// unless the move promotes, as in uci.Move. It is exported for engines
// through SearchBoard (#37); the zero value is NullSearchMove.
type SearchMove struct {
	from, to  int8
	promotion common.PieceType
}

// toUCI converts a generated move to the public form.
func (m SearchMove) toUCI() uci.Move {
	from, to := squarePositions[m.from], squarePositions[m.to]
	return uci.Move{FromX: from.X, FromY: from.Y, ToX: to.X, ToY: to.Y, Promotion: m.promotion}
}

// appendLegalMoves appends every legal move of the side to move. Pass a
// slice of a stack array (buffer[:0]) to generate without allocating.
//
//	var buffer [maxMovesPerPosition]SearchMove
//	moves := game.appendLegalMoves(buffer[:0])
func (g *Game) appendLegalMoves(moves []SearchMove) []SearchMove {
	start := len(moves)
	moves = g.appendPseudoLegalMoves(moves)
	legal := moves[:start]
	// Filtering in place is safe: legal never grows past the move being read.
	for _, move := range moves[start:] {
		if g.leavesKingSafe(move) {
			legal = append(legal, move)
		}
	}
	return legal
}

// hasLegalMove stops at the first legal move, which is all mate and
// stalemate detection needs.
func (g *Game) hasLegalMove() bool {
	var buffer [maxMovesPerPosition]SearchMove
	for _, move := range g.appendPseudoLegalMoves(buffer[:0]) {
		if g.leavesKingSafe(move) {
			return true
		}
	}
	return false
}

// leavesKingSafe plays move, checks the mover's king and takes it back.
func (g *Game) leavesKingSafe(move SearchMove) bool {
	mover := g.currentTurn
	undo := g.makeMove(move)
	safe := !g.isKingAttacked(mover)
	g.unmakeMove(move, undo)
	return safe
}

// isKingAttacked reports whether the king of color stands attacked.
func (g *Game) isKingAttacked(color common.Color) bool {
	king := g.board.kings[color]
	return g.board.isAttacked(squareIndex(king.X, king.Y), opposite(color))
}

// appendPseudoLegalMoves appends the moves the pieces can make, ignoring
// whether they leave the own king in check.
func (g *Game) appendPseudoLegalMoves(moves []SearchMove) []SearchMove {
	for from := 0; from < 64; from++ {
		if code := g.board.squares[from]; code.isColor(g.currentTurn) {
			moves = g.appendPieceMoves(moves, from, code.pieceType())
		}
	}
	return moves
}

func (g *Game) appendPieceMoves(moves []SearchMove, from int, pieceType common.PieceType) []SearchMove {
	switch pieceType {
	case common.Pawn:
		return g.appendPawnMoves(moves, from)
	case common.Knight:
		return g.appendStepMoves(moves, from, moveTables.knight[from])
	case common.Bishop:
		return g.appendSlideMoves(moves, from, diagonalRays(from))
	case common.Rook:
		return g.appendSlideMoves(moves, from, orthogonalRays(from))
	case common.Queen:
		return g.appendSlideMoves(moves, from, moveTables.rays[from][:])
	default:
		moves = g.appendStepMoves(moves, from, moveTables.king[from])
		return g.appendCastlingMoves(moves, from)
	}
}

// appendStepMoves adds each target not occupied by an own piece.
func (g *Game) appendStepMoves(moves []SearchMove, from int, targets []int8) []SearchMove {
	for _, to := range targets {
		if !g.board.squares[to].isColor(g.currentTurn) {
			moves = append(moves, SearchMove{from: int8(from), to: to})
		}
	}
	return moves
}

// appendSlideMoves walks each ray up to and including the first enemy piece.
func (g *Game) appendSlideMoves(moves []SearchMove, from int, rays [][]int8) []SearchMove {
	for _, ray := range rays {
		moves = g.appendRayMoves(moves, from, ray)
	}
	return moves
}

func (g *Game) appendRayMoves(moves []SearchMove, from int, ray []int8) []SearchMove {
	for _, to := range ray {
		code := g.board.squares[to]
		if code.isColor(g.currentTurn) {
			return moves
		}
		moves = append(moves, SearchMove{from: int8(from), to: to})
		if code != noPiece {
			return moves
		}
	}
	return moves
}

// appendCastlingMoves adds each castle available from the king's square;
// the final check on the destination is left to the legality filter.
func (g *Game) appendCastlingMoves(moves []SearchMove, from int) []SearchMove {
	for _, side := range castlingSides {
		if side.color == g.currentTurn && side.kingFromSquare() == from && g.canCastle(side) {
			moves = append(moves, SearchMove{from: int8(from), to: int8(squareIndex(side.kingTo.X, side.kingTo.Y))})
		}
	}
	return moves
}
