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

	GetKingPosition(color Color) (int, int)
	SetKingPosition(color Color, x, y int)
	IsSquareAttacked(x, y int, byColor Color) bool

	AddPiece(piece IPiece, x, y int) error
	RemovePiece(x, y int) error
	Copy() IBoard
}

type Board struct {
	board   [8][8]IPiece
	kingPos map[Color]Position
}

func (b *Board) SetKingPosition(color Color, x, y int) {
	b.kingPos[color] = Position{X: x, Y: y}
}

func (b *Board) GetKingPosition(color Color) (int, int) {
	return b.kingPos[color].X, b.kingPos[color].Y
}

// IsSquareAttacked determines whether the square at coordinates (x, y) is attacked by any piece of the specified color (byColor).
// It iterates over all squares on the board, checks for pieces of the given color, and evaluates their valid moves.
// If any valid move targets the specified square, the function returns true, indicating the square is under attack.
// Returns false if no attacking piece is found.
//
// Parameters:
//
//	x, y    - The coordinates of the square to check.
//	byColor - The color of the pieces to consider as attackers.
//
// Returns:
//
//	bool - True if the square is attacked by any piece of the specified color, false otherwise.
func (b *Board) IsSquareAttacked(x, y int, byColor Color) bool {
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			piece, _ := b.GetPiece(i, j)

			if piece != nil && piece.GetColor() == byColor {
				moves := piece.GetValidMoves(i, j, b)
				for _, move := range moves {
					if move.X == x && move.Y == y {
						return true
					}
				}
			}

		}
	}
	return false
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

	if piece.GetType() == common.King || piece.GetType() == common.Rook || piece.GetType() == common.Pawn {
		if piece.GetType() == common.King {
			piece.(*King).SetHasMoved(true)
			b.kingPos[piece.GetColor()] = Position{X: toX, Y: toY}
		} else if piece.GetType() == common.Rook {
			piece.(*Rook).SetHasMoved(true)
		} else if piece.GetType() == common.Pawn {
			piece.(*Pawn).SetHasMoved(true)
		}
	}

	return nil
}

func NewInitialBoard() IBoard {
	// Setup initial positions for all pieces
	initialBoard := &Board{
		board:   [8][8]IPiece{},
		kingPos: make(map[Color]Position),
	}
	// Pawn setup
	for i := 0; i < 8; i++ {
		initialBoard.board[i][6] = NewPawn(White)
		initialBoard.board[i][1] = NewPawn(Black)
	}
	// Rook setup
	initialBoard.board[0][7] = NewRook(White)
	initialBoard.board[7][7] = NewRook(White)
	initialBoard.board[0][0] = NewRook(Black)
	initialBoard.board[7][0] = NewRook(Black)

	// Knight setup
	initialBoard.board[1][7] = NewKnight(White)
	initialBoard.board[6][7] = NewKnight(White)
	initialBoard.board[1][0] = NewKnight(Black)
	initialBoard.board[6][0] = NewKnight(Black)

	// Bishop setup
	initialBoard.board[2][7] = NewBishop(White)
	initialBoard.board[5][7] = NewBishop(White)
	initialBoard.board[2][0] = NewBishop(Black)
	initialBoard.board[5][0] = NewBishop(Black)

	// Queen setup
	initialBoard.board[3][7] = NewQueen(White)
	initialBoard.board[3][0] = NewQueen(Black)

	// King setup
	initialBoard.board[4][7] = NewKing(White)
	initialBoard.board[4][0] = NewKing(Black)
	initialBoard.kingPos[White] = Position{X: 4, Y: 7}
	initialBoard.kingPos[Black] = Position{X: 4, Y: 0}

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
			kingPos: make(map[Color]Position),
		}
	}
	return &Board{
		board:   *board,
		kingPos: make(map[Color]Position),
	}
}

func (b *Board) GetPiece(x, y int) (IPiece, error) {
	err := b.IsValidCoordinates(x, y)
	if err != nil {
		return nil, err
	}
	if b.board[x][y] == nil {
		return nil, nil
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

// Copy is a method to create a deep copy of the board
func (b *Board) Copy() IBoard {
	newBoard := &Board{
		board:   [8][8]IPiece{},
		kingPos: make(map[Color]Position),
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
