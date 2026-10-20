package chess

import (
	"errors"
	"fmt"
	common "idylicaro/go-chess/internal/common"
)

type Position struct {
	X int
	Y int
}

type IBoard interface {
	GetPiece(x, y int) (IPiece, error)
	MovePiece(fromX, fromY, toX, toY int) error

	IsValidCoordinates(x, y int) error

	GetKingPosition(color common.Color) (int, int)
	SetKingPosition(color common.Color, x, y int)
	IsSquareAttacked(x, y int, byColor common.Color) bool

	AddPiece(piece IPiece, x, y int) error
	SetPiece(piece IPiece, x, y int) error
	RemovePiece(x, y int) error
	Copy() IBoard
}

type Board struct {
	board   [8][8]IPiece
	kingPos map[common.Color]Position
}

func (b *Board) SetKingPosition(color common.Color, x, y int) {
	b.kingPos[color] = Position{X: x, Y: y}
}

func (b *Board) GetKingPosition(color common.Color) (int, int) {
	return b.kingPos[color].X, b.kingPos[color].Y
}

func (b *Board) MovePiece(fromX int, fromY int, toX int, toY int) error {
	err := b.IsValidCoordinates(fromX, fromY)
	if err != nil {
		return err
	}
	err = b.IsValidCoordinates(toX, toY)
	if err != nil {
		return err
	}

	piece, err := b.GetPiece(fromX, fromY)
	if err != nil {
		return err
	}

	if piece == nil {
		return errors.New("could not get piece from (" + fmt.Sprintf("%d, %d", fromX, fromY) + "): no piece found at coordinates")
	}

	err = b.RemovePiece(toX, toY)
	if err != nil {
		return err
	}

	b.board[toX][toY] = piece
	_ = b.RemovePiece(fromX, fromY)

	if piece.GetType() == common.King {
		b.kingPos[piece.GetColor()] = Position{X: toX, Y: toY}
	}

	return nil
}

func NewInitialBoard() IBoard {
	// Setup initial positions for all pieces
	initialBoard := &Board{
		board:   [8][8]IPiece{},
		kingPos: make(map[common.Color]Position),
	}
	// Pawn setup
	for i := 0; i < 8; i++ {
		initialBoard.board[i][6] = NewPawn(common.White)
		initialBoard.board[i][1] = NewPawn(common.Black)
	}
	// Rook setup
	initialBoard.board[0][7] = NewRook(common.White)
	initialBoard.board[7][7] = NewRook(common.White)
	initialBoard.board[0][0] = NewRook(common.Black)
	initialBoard.board[7][0] = NewRook(common.Black)

	// Knight setup
	initialBoard.board[1][7] = NewKnight(common.White)
	initialBoard.board[6][7] = NewKnight(common.White)
	initialBoard.board[1][0] = NewKnight(common.Black)
	initialBoard.board[6][0] = NewKnight(common.Black)

	// Bishop setup
	initialBoard.board[2][7] = NewBishop(common.White)
	initialBoard.board[5][7] = NewBishop(common.White)
	initialBoard.board[2][0] = NewBishop(common.Black)
	initialBoard.board[5][0] = NewBishop(common.Black)

	// Queen setup
	initialBoard.board[3][7] = NewQueen(common.White)
	initialBoard.board[3][0] = NewQueen(common.Black)

	// King setup
	initialBoard.board[4][7] = NewKing(common.White)
	initialBoard.board[4][0] = NewKing(common.Black)
	initialBoard.kingPos[common.White] = Position{X: 4, Y: 7}
	initialBoard.kingPos[common.Black] = Position{X: 4, Y: 0}

	// Place pieces on the board
	return initialBoard
}

func (b *Board) RemovePiece(x int, y int) error {
	err := b.IsValidCoordinates(x, y)
	if err != nil {
		return err
	}
	b.board[x][y] = nil
	return nil
}

func NewBoard(board *[8][8]IPiece) IBoard {
	if board == nil {
		return &Board{
			board:   [8][8]IPiece{},
			kingPos: make(map[common.Color]Position),
		}
	}
	return &Board{
		board:   *board,
		kingPos: make(map[common.Color]Position),
	}
}

func (b *Board) GetPiece(x, y int) (IPiece, error) {
	err := b.IsValidCoordinates(x, y)
	if err != nil {
		return nil, err
	}
	// Return the piece (can be nil) with no error - this is correct behavior
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

func (b *Board) SetPiece(piece IPiece, x, y int) error {
	err := b.IsValidCoordinates(x, y)
	if err != nil {
		return err
	}

	b.board[x][y] = piece // Force placement regardless of existing piece
	return nil
}

func (b *Board) IsValidCoordinates(x, y int) error {
	if x < 0 || x > 7 || y < 0 || y > 7 {
		return errors.New("invalid coordinates")
	}
	return nil
}

// Copy is a method to create a deep copy of the board
func (b *Board) Copy() IBoard {
	newBoard := &Board{
		board:   [8][8]IPiece{},
		kingPos: make(map[common.Color]Position),
	}
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			if b.board[i][j] != nil {
				newBoard.board[i][j] = b.board[i][j].Copy()
			}
		}
	}
	for color, pos := range b.kingPos {
		newBoard.kingPos[color] = pos
	}
	return newBoard
}

// FormatForDebug implements debug.BoardFormatter interface
func (b *Board) FormatForDebug() string {
	pieces := make([][]string, 8)
	for i := range pieces {
		pieces[i] = make([]string, 8)
	}

	// Unicode chess symbols
	symbols := map[string]string{
		"WhitePawn": "♙", "BlackPawn": "♟",
		"WhiteRook": "♖", "BlackRook": "♜",
		"WhiteKnight": "♘", "BlackKnight": "♞",
		"WhiteBishop": "♗", "BlackBishop": "♝",
		"WhiteQueen": "♕", "BlackQueen": "♛",
		"WhiteKing": "♔", "BlackKing": "♚",
	}

	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			piece, err := b.GetPiece(x, y)
			if err != nil || piece == nil {
				pieces[y][x] = ""
			} else {
				color := "White"
				if piece.GetColor().String() == "Black" {
					color = "Black"
				}
				key := color + piece.GetType().String()
				if symbol, exists := symbols[key]; exists {
					pieces[y][x] = symbol
				} else {
					pieces[y][x] = "?"
				}
			}
		}
	}

	// Use the generic formatter from debug package
	return formatBoardGeneric(pieces)
}

// Helper function for board formatting
func formatBoardGeneric(pieces [][]string) string {
	var result string
	result += "  a b c d e f g h\n"

	for y := 0; y < 8; y++ {
		result += fmt.Sprintf("%d ", 8-y)
		for x := 0; x < 8; x++ {
			if pieces[y][x] != "" {
				result += fmt.Sprintf("%s ", pieces[y][x])
			} else {
				result += ". "
			}
		}
		result += fmt.Sprintf(" %d\n", 8-y)
	}
	result += "  a b c d e f g h"

	return result
}
