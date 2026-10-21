package chess

import (
	"fmt"
	"github.com/idylicaro/go-chess/internal/common"
	"strings"
)

// Position is a square as zero-based column X (a=0) and row Y (rank 1=0).
type Position struct {
	X int
	Y int
}

// IBoard is the board a Game plays on. Game.GetBoard returns it for display;
// mutating it directly bypasses the rules and corrupts the game (see docs/API.md).
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

// Board is the 8x8 implementation of IBoard. Squares are packed codes, so
// Copy is a value copy (#12).
type Board struct {
	squares [64]pieceCode
	kings   [2]Position // indexed by common.Color
}

// SetKingPosition records where the king of color stands.
func (b *Board) SetKingPosition(color common.Color, x, y int) {
	b.kings[color] = Position{X: x, Y: y}
}

// GetKingPosition returns the column and row of the king of color.
func (b *Board) GetKingPosition(color common.Color) (int, int) {
	return b.kings[color].X, b.kings[color].Y
}

// MovePiece moves the piece on (fromX, fromY) to (toX, toY), capturing any piece there.
func (b *Board) MovePiece(fromX int, fromY int, toX int, toY int) error {
	if err := b.IsValidCoordinates(fromX, fromY); err != nil {
		return err
	}
	if err := b.IsValidCoordinates(toX, toY); err != nil {
		return err
	}
	code := b.squares[squareIndex(fromX, fromY)]
	if code == noPiece {
		return fmt.Errorf("cannot move from (%d, %d): there is no piece there", fromX, fromY)
	}
	b.squares[squareIndex(toX, toY)] = code
	b.squares[squareIndex(fromX, fromY)] = noPiece
	if code.pieceType() == common.King {
		b.kings[code.color()] = Position{X: toX, Y: toY}
	}
	return nil
}

// newInitialBoard is the standard starting position.
func newInitialBoard() *Board {
	board, err := parseFENPlacement(strings.Fields(StartingFEN)[0])
	if err != nil {
		panic(fmt.Sprintf("StartingFEN %q does not parse: %v", StartingFEN, err))
	}
	return board
}

// RemovePiece empties the square (x, y).
func (b *Board) RemovePiece(x int, y int) error {
	return b.SetPiece(nil, x, y)
}

// newBoard returns an empty board.
func newBoard() *Board {
	return &Board{}
}

// GetPiece returns the piece on (x, y), or nil for an empty square.
func (b *Board) GetPiece(x, y int) (IPiece, error) {
	if err := b.IsValidCoordinates(x, y); err != nil {
		return nil, err
	}
	// Empty squares map to a nil IPiece, which is the documented "no piece".
	return pieceViews[b.squares[squareIndex(x, y)]], nil
}

// AddPiece places piece on an empty square and fails if it is occupied.
func (b *Board) AddPiece(piece IPiece, x, y int) error {
	if err := b.IsValidCoordinates(x, y); err != nil {
		return err
	}
	if b.squares[squareIndex(x, y)] != noPiece {
		return fmt.Errorf("cannot add a piece at (%d, %d): the square is occupied", x, y)
	}
	return b.SetPiece(piece, x, y)
}

// SetPiece places piece on (x, y), replacing whatever stands there.
func (b *Board) SetPiece(piece IPiece, x, y int) error {
	if err := b.IsValidCoordinates(x, y); err != nil {
		return err
	}
	b.squares[squareIndex(x, y)] = codeOfPiece(piece)
	return nil
}

// IsValidCoordinates returns an error unless (x, y) is inside the board.
func (b *Board) IsValidCoordinates(x, y int) error {
	if !onBoard(x, y) {
		return fmt.Errorf("invalid coordinates (%d, %d): expected both between 0 and 7", x, y)
	}
	return nil
}

func onBoard(x, y int) bool {
	return x >= 0 && x <= 7 && y >= 0 && y <= 7
}

// Copy returns an independent copy of the board.
func (b *Board) Copy() IBoard {
	copied := *b
	return &copied
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
