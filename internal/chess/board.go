package chess

import (
	"errors"
	"idylicaro/go-chess/internal/uci"
)

type IBoard interface {
	GetPiece(x, y int) (Piece, error)
	MovePiece(move uci.Move) error
	ValidateMove(piece Piece, toX, toY int) error //?? This best way, and name...

	Copy() IBoard
	AddPiece(piece Piece, x, y int) error
	RemovePiece(x, y int) error
	//verificar o estado do tabuleiro de forma geral
	FindKingPosition(color Color) (int, int, error)
	IsSquareAttacked(x, y int, byColor Color) (bool, error)
}

type Board struct {
	board [8][8]Piece
}

func NewBoard(board *[8][8]Piece) *Board {
	if board == nil {
		return &Board{
			board: [8][8]Piece{},
		}
	}
	return &Board{
		board: *board,
	}
}

func (b *Board) GetPiece(x, y int) (Piece, error) {
	if x < 0 || x > 7 || y < 0 || y > 7 {
		return Piece{}, errors.New("invalid coordinates")
	}
	return b.board[x][y], nil
}
