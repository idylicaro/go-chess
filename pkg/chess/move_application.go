package chess

import (
	"fmt"
	"github.com/idylicaro/go-chess/internal/common"
	"github.com/idylicaro/go-chess/internal/uci"
)

// applyMoveToBoard performs every board change of move: the piece itself plus
// the rook when castling, the captured pawn on en passant and the promoted
// piece. Both Game.Move and the legality simulation go through it, so a
// simulated move can never differ from the real one (#4).
//
//	err := applyMoveToBoard(board.Copy(), uci.Move{FromX: 4, FromY: 3, ToX: 3, ToY: 2}, &Position{X: 3, Y: 2})
func applyMoveToBoard(board IBoard, move uci.Move, enPassantTarget *Position) error {
	piece, err := board.GetPiece(move.FromX, move.FromY)
	if err != nil {
		return err
	}
	if piece == nil {
		return fmt.Errorf("move %s: there is no piece on %s", move, squareName(move.FromX, move.FromY))
	}
	if isCastlingMove(piece, move) {
		if err := moveCastlingRook(board, move); err != nil {
			return err
		}
	}
	if isEnPassantCapture(piece, move, enPassantTarget) {
		removeEnPassantVictim(board, move, piece.GetColor())
	}
	if isPromotionMove(piece, move) {
		return promotePawn(board, move, piece.GetColor())
	}
	return board.MovePiece(move.FromX, move.FromY, move.ToX, move.ToY)
}

func isCastlingMove(piece IPiece, move uci.Move) bool {
	return piece.GetType() == common.King && (move.ToX == move.FromX+2 || move.ToX == move.FromX-2)
}

// moveCastlingRook moves the h-rook (short) or a-rook (long) next to the king.
func moveCastlingRook(board IBoard, move uci.Move) error {
	rookFromX, rookToX := 7, move.FromX+1
	if move.ToX < move.FromX {
		rookFromX, rookToX = 0, move.FromX-1
	}
	return board.MovePiece(rookFromX, move.FromY, rookToX, move.FromY)
}

func isEnPassantCapture(piece IPiece, move uci.Move, enPassantTarget *Position) bool {
	return piece.GetType() == common.Pawn && enPassantTarget != nil &&
		move.ToX == enPassantTarget.X && move.ToY == enPassantTarget.Y
}

// removeEnPassantVictim removes the pawn that sits behind the en passant target square.
func removeEnPassantVictim(board IBoard, move uci.Move, moverColor common.Color) {
	capturedPawnY := move.ToY - 1
	if moverColor == common.White {
		capturedPawnY = move.ToY + 1
	}
	_ = board.RemovePiece(move.ToX, capturedPawnY)
}

func isPromotionMove(piece IPiece, move uci.Move) bool {
	if piece.GetType() != common.Pawn {
		return false
	}
	return (piece.GetColor() == common.White && move.ToY == 0) ||
		(piece.GetColor() == common.Black && move.ToY == 7)
}

// promotePawn replaces the pawn with the requested piece on the last rank.
func promotePawn(board IBoard, move uci.Move, color common.Color) error {
	if err := board.RemovePiece(move.FromX, move.FromY); err != nil {
		return fmt.Errorf("move %s: failed to remove the promoting pawn: %w", move, err)
	}
	if err := board.SetPiece(newPromotedPiece(move.Promotion, color), move.ToX, move.ToY); err != nil {
		return fmt.Errorf("move %s: failed to place the promoted piece: %w", move, err)
	}
	return nil
}

// newPromotedPiece builds the promotion piece; anything other than R/B/N
// (including the zero value) promotes to a queen.
func newPromotedPiece(promotion common.PieceType, color common.Color) IPiece {
	switch promotion {
	case common.Rook, common.Bishop, common.Knight:
		return newPiece(promotion, color)
	default:
		return NewQueen(color)
	}
}
