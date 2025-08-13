package chess

import (
	"errors"
)

type Position struct {
	X int
	Y int
}

type IBoard interface {
	GetPiece(x, y int) (Piece, error)
	MovePiece(fromX, fromY, toX, toY int) error

	GetValidMoves(fromX, fromY int) ([]Position, error)
	NewInitialBoard() IBoard

	//verificar o estado do tabuleiro de forma geral
	FindKingPosition(color Color) (int, int, error)
	IsSquareAttacked(x, y int, byColor Color) (bool, error)

	Copy() IBoard
	AddPiece(piece Piece, x, y int) error
	RemovePiece(x, y int) error
}

type Board struct {
	board [8][8]*Piece
}

func NewBoard(board *[8][8]*Piece) *Board {
	if board == nil {
		return &Board{
			board: [8][8]*Piece{},
		}
	}
	return &Board{
		board: *board,
	}
}

func (b *Board) GetPiece(x, y int) (*Piece, error) {
	err := b.isValidCoordinates(x, y)
	if err != nil {
		return nil, err
	}
	return b.board[x][y], nil
}

func (b *Board) AddPiece(piece *Piece, x, y int) error {
	err := b.isValidCoordinates(x, y)
	if err != nil {
		return err
	}

	if b.board[x][y] != nil {
		return errors.New("there is already a piece")
	}

	b.board[x][y] = piece
	return nil
}

func (b *Board) isValidCoordinates(x, y int) error {
	if x < 0 || x > 7 || y < 0 || y > 7 {
		return errors.New("invalid coordinates")
	}
	return nil
}
