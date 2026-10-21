package chess

import (
	"github.com/idylicaro/go-chess/internal/common"
	"reflect"
	"testing"
)

func emptyBoard() *Board {
	return newBoard()
}

func TestPawnTable(t *testing.T) {
	tests := []struct {
		name      string
		piece     *pawnPiece
		pos       Position
		setup     func(board *Board)
		enPassant *Position
		expected  []Position
	}{
		{
			name:      "Initial double move",
			piece:     newPawn(common.White),
			pos:       Position{X: 4, Y: 6},
			setup:     func(board *Board) {},
			enPassant: nil,
			expected:  []Position{{X: 4, Y: 5}, {X: 4, Y: 4}},
		},
		{
			name:  "White Attack left",
			piece: newPawn(common.White),
			pos:   Position{X: 4, Y: 4},
			setup: func(board *Board) {
				_ = board.SetPiece(newPawn(common.Black), 3, 3)
				_ = board.SetPiece(newPawn(common.White), 5, 3)
			},
			enPassant: nil,
			expected:  []Position{{X: 4, Y: 3}, {X: 3, Y: 3}},
		},
		{
			name:  "White Attack right",
			piece: newPawn(common.White),
			pos:   Position{X: 4, Y: 4},
			setup: func(board *Board) {
				_ = board.SetPiece(newPawn(common.White), 3, 3)
				_ = board.SetPiece(newPawn(common.Black), 5, 3)
			},
			enPassant: nil,
			expected:  []Position{{X: 4, Y: 3}, {X: 5, Y: 3}},
		},
		{
			name:  "Black Attack Left",
			piece: newPawn(common.Black),
			pos:   Position{X: 4, Y: 4},
			setup: func(board *Board) {
				_ = board.SetPiece(newPawn(common.White), 3, 5)
				_ = board.SetPiece(newPawn(common.Black), 5, 5)
			},
			enPassant: nil,
			expected:  []Position{{X: 4, Y: 5}, {X: 3, Y: 5}},
		},
		{
			name:  "Black Attack right",
			piece: newPawn(common.Black),
			pos:   Position{X: 4, Y: 4},
			setup: func(board *Board) {
				_ = board.SetPiece(newPawn(common.Black), 3, 5)
				_ = board.SetPiece(newPawn(common.White), 5, 5)
			},
			enPassant: nil,
			expected:  []Position{{X: 4, Y: 5}, {X: 5, Y: 5}},
		},
		{
			name:  "En passant",
			piece: newPawn(common.White),
			pos:   Position{X: 4, Y: 3},
			setup: func(board *Board) {
				_ = board.SetPiece(newPawn(common.Black), 5, 3)
			},
			enPassant: &Position{X: 5, Y: 2},
			expected:  []Position{{X: 4, Y: 2}, {X: 5, Y: 2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board := emptyBoard()
			_ = board.SetPiece(tt.piece, tt.pos.X, tt.pos.Y)
			tt.setup(board)
			got := tt.piece.GetValidMovesWithEnPassant(tt.pos.X, tt.pos.Y, board, tt.enPassant)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("%s: got %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestKnightEmptyBoard(t *testing.T) {
	board := emptyBoard()
	knight := newKnight(common.White)
	_ = board.SetPiece(knight, 4, 4)

	moves := knight.GetValidMoves(4, 4, board)
	if len(moves) != 8 {
		t.Errorf("Knight moves: got %d, want 8", len(moves))
	}
	expected := []Position{
		{X: 3, Y: 2}, {X: 5, Y: 2}, {X: 2, Y: 3}, {X: 6, Y: 3},
		{X: 2, Y: 5}, {X: 3, Y: 6}, {X: 6, Y: 5}, {X: 5, Y: 6},
	}
	for _, pos := range expected {
		found := false
		for _, m := range moves {
			if m == pos {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Knight moves: missing expected position %v", pos)
		}
	}
}

func TestKnightBlockedByOwnPiece(t *testing.T) {
	board := emptyBoard()
	knight := newKnight(common.White)
	_ = board.SetPiece(knight, 4, 4)
	// Block some moves with own pieces
	_ = board.SetPiece(newPawn(common.White), 3, 2)
	_ = board.SetPiece(newPawn(common.White), 5, 2)

	moves := knight.GetValidMoves(4, 4, board)
	expected := []Position{
		{X: 2, Y: 3}, {X: 6, Y: 3}, {X: 2, Y: 5}, {X: 3, Y: 6},
		{X: 6, Y: 5}, {X: 5, Y: 6},
	}
	if len(moves) != len(expected) {
		t.Errorf("Knight blocked: got %d moves, want %d", len(moves), len(expected))
	}
	for _, pos := range expected {
		found := false
		for _, m := range moves {
			if m == pos {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Knight blocked: missing expected position %v", pos)
		}
	}
}

func TestKnightCanCaptureEnemy(t *testing.T) {
	board := emptyBoard()
	knight := newKnight(common.White)
	_ = board.SetPiece(knight, 4, 4)
	_ = board.SetPiece(newPawn(common.Black), 3, 2)
	_ = board.SetPiece(newPawn(common.Black), 5, 2)

	moves := knight.GetValidMoves(4, 4, board)
	expected := []Position{
		{X: 3, Y: 2}, {X: 5, Y: 2}, {X: 2, Y: 3}, {X: 6, Y: 3},
		{X: 2, Y: 5}, {X: 3, Y: 6}, {X: 6, Y: 5}, {X: 5, Y: 6},
	}
	for _, pos := range expected {
		found := false
		for _, m := range moves {
			if m == pos {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Knight capture: missing expected position %v", pos)
		}
	}
}

func TestBishopEmptyBoard(t *testing.T) {
	board := emptyBoard()
	bishop := newBishop(common.White)
	_ = board.SetPiece(bishop, 4, 4)

	moves := bishop.GetValidMoves(4, 4, board)
	if len(moves) != 13 {
		t.Errorf("Bishop moves: got %d, want 13", len(moves))
	}
}

func TestBishopBlockedByOwnPiece(t *testing.T) {
	board := emptyBoard()
	bishop := newBishop(common.White)
	_ = board.SetPiece(bishop, 4, 4)
	// Block some moves with own pieces
	_ = board.SetPiece(newPawn(common.White), 3, 3)
	_ = board.SetPiece(newPawn(common.White), 5, 3)

	moves := bishop.GetValidMoves(4, 4, board)
	expected := []Position{
		{X: 3, Y: 5}, {X: 2, Y: 6}, {X: 1, Y: 7}, {X: 5, Y: 5}, {X: 6, Y: 6}, {X: 7, Y: 7},
	}
	if len(moves) != len(expected) {
		t.Errorf("Bishop blocked: got %d moves, want %d", len(moves), len(expected))
	}
	for _, pos := range expected {
		found := false
		for _, m := range moves {
			if m == pos {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Bishop blocked: missing expected position %v", pos)
		}
	}
}
func TestBishopCanCaptureEnemy(t *testing.T) {
	board := emptyBoard()
	bishop := newBishop(common.White)
	_ = board.SetPiece(bishop, 4, 4)
	_ = board.SetPiece(newPawn(common.Black), 3, 3)
	_ = board.SetPiece(newPawn(common.Black), 5, 3)

	moves := bishop.GetValidMoves(4, 4, board)
	expected := []Position{
		{X: 3, Y: 3}, {X: 5, Y: 3}, {X: 3, Y: 5}, {X: 2, Y: 6}, {X: 1, Y: 7}, {X: 5, Y: 5}, {X: 6, Y: 6}, {X: 7, Y: 7},
	}
	for _, pos := range expected {
		found := false
		for _, m := range moves {
			if m == pos {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Bishop capture: missing expected position %v", pos)
		}
	}
}

func TestRookMovesEmptyBoard(t *testing.T) {
	board := emptyBoard()
	rook := newRook(common.White)
	_ = board.SetPiece(rook, 4, 4)

	moves := rook.GetValidMoves(4, 4, board)
	if len(moves) != 14 {
		t.Errorf("Rook moves: got %d, want 14", len(moves))
	}
}

func TestRookBlockedByOwnPiece(t *testing.T) {
	board := emptyBoard()
	rook := newRook(common.White)
	_ = board.SetPiece(rook, 4, 4)
	// Block some moves with own pieces
	_ = board.SetPiece(newPawn(common.White), 4, 3)
	_ = board.SetPiece(newPawn(common.White), 4, 5)

	moves := rook.GetValidMoves(4, 4, board)
	expected := []Position{
		{X: 3, Y: 4}, {X: 2, Y: 4}, {X: 1, Y: 4}, {X: 0, Y: 4}, {X: 5, Y: 4}, {X: 6, Y: 4}, {X: 7, Y: 4},
	}
	if len(moves) != len(expected) {
		t.Errorf("Rook blocked: got %d moves, want %d", len(moves), len(expected))
	}
	for _, pos := range expected {
		found := false
		for _, m := range moves {
			if m == pos {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Rook blocked: missing expected position %v", pos)
		}
	}
}

func TestRookCanCaptureEnemy(t *testing.T) {
	board := emptyBoard()
	rook := newRook(common.White)
	_ = board.SetPiece(rook, 4, 4)
	_ = board.SetPiece(newPawn(common.Black), 3, 4)
	_ = board.SetPiece(newPawn(common.Black), 5, 4)

	moves := rook.GetValidMoves(4, 4, board)
	expected := []Position{
		{X: 3, Y: 4}, {X: 5, Y: 4}, {X: 4, Y: 5}, {X: 4, Y: 3},
	}
	for _, pos := range expected {
		found := false
		for _, m := range moves {
			if m == pos {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Rook capture: missing expected position %v", pos)
		}
	}
}

func TestQueenMoves(t *testing.T) {
	board := emptyBoard()
	queen := newQueen(common.White)
	_ = board.SetPiece(queen, 4, 4)

	moves := queen.GetValidMoves(4, 4, board)
	if len(moves) != 27 {
		t.Errorf("Queen moves: got %d, want 27", len(moves))
	}
}

func TestKingMoves(t *testing.T) {
	board := emptyBoard()
	king := newKing(common.White)
	_ = board.SetPiece(king, 4, 4)

	moves := king.GetValidMoves(4, 4, board)
	if len(moves) != 8 {
		t.Errorf("King moves: got %d, want 8", len(moves))
	}
}
