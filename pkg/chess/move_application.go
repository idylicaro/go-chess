package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
	"github.com/idylicaro/go-chess/internal/uci"
)

// Predicates over a uci.Move and the piece making it, for notation code
// that works with public moves. Board changes live in makeMove.

func isCastlingMove(piece IPiece, move uci.Move) bool {
	return piece.GetType() == common.King && (move.ToX == move.FromX+2 || move.ToX == move.FromX-2)
}

func isEnPassantCapture(piece IPiece, move uci.Move, enPassantTarget *Position) bool {
	return piece.GetType() == common.Pawn && enPassantTarget != nil &&
		move.ToX == enPassantTarget.X && move.ToY == enPassantTarget.Y
}

func isPromotionMove(piece IPiece, move uci.Move) bool {
	if piece.GetType() != common.Pawn {
		return false
	}
	return (piece.GetColor() == common.White && move.ToY == 0) ||
		(piece.GetColor() == common.Black && move.ToY == 7)
}
