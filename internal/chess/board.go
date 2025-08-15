package chess

import (
	"errors"
)

type Position struct {
	X int
	Y int
}

type IBoard interface {
	GetPiece(x, y int) (IPiece, error)
	MovePiece(fromX, fromY, toX, toY int) error

	IsValidCoordinates(x, y int) error
	GetValidMoves(fromX, fromY int) ([]Position, error)
	NewInitialBoard() IBoard

	//verificar o estado do tabuleiro de forma geral
	FindKingPosition(color Color) (int, int, error)
	IsSquareAttacked(x, y int, byColor Color) (bool, error)

	Copy() IBoard
	AddPiece(piece IPiece, x, y int) error
	RemovePiece(x, y int) error
}

type Board struct {
	board [8][8]IPiece
}

// Copy implements IBoard.
func (b *Board) Copy() IBoard {
	panic("unimplemented")
}

// FindKingPosition implements IBoard.
func (b *Board) FindKingPosition(color Color) (int, int, error) {
	panic("unimplemented")
}

// GetValidMoves implements IBoard.
func (b *Board) GetValidMoves(fromX int, fromY int) ([]Position, error) {
	panic("unimplemented")
}

// IsSquareAttacked implements IBoard.
func (b *Board) IsSquareAttacked(x int, y int, byColor Color) (bool, error) {
	panic("unimplemented")
}

// MovePiece implements IBoard.
func (b *Board) MovePiece(fromX int, fromY int, toX int, toY int) error {
	panic("unimplemented")
}

// NewInitialBoard implements IBoard.
func (b *Board) NewInitialBoard() IBoard {
	panic("unimplemented")
}

// RemovePiece implements IBoard.
func (b *Board) RemovePiece(x int, y int) error {
	panic("unimplemented")
}

func NewBoard(board *[8][8]IPiece) *Board {
	if board == nil {
		return &Board{
			board: [8][8]IPiece{},
		}
	}
	return &Board{
		board: *board,
	}
}

func (b *Board) GetPiece(x, y int) (IPiece, error) {
	err := b.IsValidCoordinates(x, y)
	if err != nil {
		return nil, err
	}
	return b.board[x][y], nil
}

func (b *Board) AddPiece(piece IPiece, x, y int) error {
	err := b.IsValidCoordinates(x, y)
	if err != nil {
		return err
	}

	if b.board[x][y] != nil {
		return errors.New("there is already a piece")
	}

	b.board[x][y] = piece
	return nil
}

func (b *Board) IsValidCoordinates(x, y int) error {
	if x < 0 || x > 7 || y < 0 || y > 7 {
		return errors.New("invalid coordinates")
	}
	return nil
}
