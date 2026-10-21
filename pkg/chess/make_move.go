package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
)

// moveUndo is what makeMove overwrites and unmakeMove needs back.
type moveUndo struct {
	captured        pieceCode
	capturedAt      int8 // differs from the destination only on en passant
	castlingRights  CastlingRights
	enPassantTarget *Position
	halfMoveClock   int
	hash            uint64
}

// makeMove plays a pseudo-legal move on the game's own board: the piece, the
// rook when castling, the pawn taken en passant, the promotion, and the
// rights, counters and turn. Game.Move, the legality filter and perft all go
// through it, so a tested move can never differ from the played one (#4).
// It does not check legality; undo it with unmakeMove.
//
//	undo := game.makeMove(move)
//	defer game.unmakeMove(move, undo)
func (g *Game) makeMove(move SearchMove) moveUndo {
	board := g.board
	mover := board.squares[move.from]
	undo := moveUndo{board.squares[move.to], move.to, g.castlingRights, g.enPassantTarget, g.halfMoveClock, g.hash}
	if mover.pieceType() == common.Pawn && g.isEnPassantSquare(int(move.to)) {
		undo.capturedAt = move.to - int8(8*pawnForward(g.currentTurn))
		undo.captured = board.squares[undo.capturedAt]
		g.putCode(int(undo.capturedAt), noPiece)
	}
	g.putCode(int(move.from), noPiece)
	g.putCode(int(move.to), promotedCode(mover, move))
	if mover.pieceType() == common.King {
		g.moveKing(move)
	}
	g.advanceCounters(mover, undo.captured, move)
	return undo
}

// unmakeMove restores the position from before makeMove(move).
func (g *Game) unmakeMove(move SearchMove, undo moveUndo) {
	g.currentTurn = opposite(g.currentTurn)
	board := g.board
	mover := board.squares[move.to]
	if move.promotion != common.Pawn {
		mover = codeOf(common.Pawn, g.currentTurn)
	}
	board.squares[move.to] = noPiece
	board.squares[undo.capturedAt] = undo.captured
	board.squares[move.from] = mover
	if mover.pieceType() == common.King {
		g.unmoveKing(move)
	}
	g.castlingRights, g.enPassantTarget, g.halfMoveClock = undo.castlingRights, undo.enPassantTarget, undo.halfMoveClock
	g.hash = undo.hash
	if g.currentTurn == common.Black {
		g.fullMoveNumber--
	}
}

// promotedCode is the piece that lands on the destination.
func promotedCode(mover pieceCode, move SearchMove) pieceCode {
	if move.promotion == common.Pawn {
		return mover
	}
	return codeOf(move.promotion, mover.color())
}

// moveKing tracks the king and, on a two-square king move, the castling rook.
func (g *Game) moveKing(move SearchMove) {
	g.board.kings[g.currentTurn] = squarePositions[move.to]
	if rookFrom, rookTo, castles := castlingRookSquares(move); castles {
		g.putCode(rookTo, g.board.squares[rookFrom])
		g.putCode(rookFrom, noPiece)
	}
}

func (g *Game) unmoveKing(move SearchMove) {
	g.board.kings[g.currentTurn] = squarePositions[move.from]
	if rookFrom, rookTo, castles := castlingRookSquares(move); castles {
		g.board.squares[rookFrom] = g.board.squares[rookTo]
		g.board.squares[rookTo] = noPiece
	}
}

// castlingRookSquares returns where the rook goes for a king move of two
// columns: the h-rook (short) or a-rook (long) lands next to the king.
func castlingRookSquares(move SearchMove) (rookFrom, rookTo int, castles bool) {
	switch move.to - move.from {
	case 2:
		return int(move.from) + 3, int(move.from) + 1, true
	case -2:
		return int(move.from) - 4, int(move.from) - 1, true
	default:
		return 0, 0, false
	}
}

// advanceCounters updates the halfmove clock (#5), castling rights, the en
// passant target, the fullmove number, the turn and their share of the hash
// after mover moved.
func (g *Game) advanceCounters(mover, captured pieceCode, move SearchMove) {
	g.halfMoveClock++
	if mover.pieceType() == common.Pawn || captured != noPiece {
		g.halfMoveClock = 0
	}
	g.hash ^= zobrist.castling[g.castlingRights] ^ enPassantKey(g.enPassantTarget) ^ zobrist.black
	g.castlingRights = g.castlingRights.afterMove(mover, int(move.from), int(move.to))
	g.enPassantTarget = enPassantTargetAfter(mover, move)
	g.hash ^= zobrist.castling[g.castlingRights] ^ enPassantKey(g.enPassantTarget)
	if g.currentTurn == common.Black {
		g.fullMoveNumber++
	}
	g.currentTurn = opposite(g.currentTurn)
}

// enPassantTargetAfter is the square a pawn skipped on its double push.
func enPassantTargetAfter(mover pieceCode, move SearchMove) *Position {
	distance := move.to - move.from
	if mover.pieceType() != common.Pawn || (distance != 16 && distance != -16) {
		return nil
	}
	return &squarePositions[(move.from+move.to)/2]
}
