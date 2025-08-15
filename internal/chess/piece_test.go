package chess

import (
	"reflect"
	"testing"
)

func emptyBoard() *Board {
	return &Board{
		board:   [8][8]IPiece{},
		kingPos: make(map[Color]Position),
	}
}

func TestPawnTable(t *testing.T) {
	tests := []struct {
		name      string
		piece     *Pawn
		hasMoved  bool
		pos       Position
		setup     func(board *Board)
		enPassant *Position
		expected  []Position
	}{
		{
			name:      "Initial double move",
			piece:     NewPawn(White),
			hasMoved:  false,
			pos:       Position{X: 4, Y: 6},
			setup:     func(board *Board) {},
			enPassant: nil,
			expected:  []Position{{X: 4, Y: 5}, {X: 4, Y: 4}},
		},
		{
			name:     "White Attack left",
			piece:    NewPawn(White),
			hasMoved: true,
			pos:      Position{X: 4, Y: 4},
			setup: func(board *Board) {
				board.board[3][3] = NewPawn(Black)
				board.board[5][3] = NewPawn(White)
			},
			enPassant: nil,
			expected:  []Position{{X: 4, Y: 3}, {X: 3, Y: 3}},
		},
		{
			name:     "White Attack right",
			piece:    NewPawn(White),
			hasMoved: true,
			pos:      Position{X: 4, Y: 4},
			setup: func(board *Board) {
				board.board[3][3] = NewPawn(White)
				board.board[5][3] = NewPawn(Black)
			},
			enPassant: nil,
			expected:  []Position{{X: 4, Y: 3}, {X: 5, Y: 3}},
		},
		{
			name:     "Black Attack Left",
			piece:    NewPawn(Black),
			hasMoved: true,
			pos:      Position{X: 4, Y: 4},
			setup: func(board *Board) {
				board.board[3][5] = NewPawn(White)
				board.board[5][5] = NewPawn(Black)
			},
			enPassant: nil,
			expected:  []Position{{X: 4, Y: 5}, {X: 3, Y: 5}},
		},
		{
			name:     "Black Attack right",
			piece:    NewPawn(Black),
			hasMoved: true,
			pos:      Position{X: 4, Y: 4},
			setup: func(board *Board) {
				board.board[3][5] = NewPawn(Black)
				board.board[5][5] = NewPawn(White)
			},
			enPassant: nil,
			expected:  []Position{{X: 4, Y: 5}, {X: 5, Y: 5}},
		},
		{
			name:     "En passant",
			piece:    NewPawn(White),
			hasMoved: true,
			pos:      Position{X: 4, Y: 3},
			setup: func(board *Board) {
				p := NewPawn(Black)
				p.SetHasMoved(true)
				board.board[5][3] = p
			},
			enPassant: &Position{X: 5, Y: 2},
			expected:  []Position{{X: 4, Y: 2}, {X: 5, Y: 2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//TODO: Need reset hasMoved for some cases.
			board := emptyBoard()
			tt.piece.SetHasMoved(tt.hasMoved)
			board.board[tt.pos.X][tt.pos.Y] = tt.piece
			tt.setup(board)
			got := tt.piece.GetValidMovesWithEnPassant(tt.pos.X, tt.pos.Y, board, tt.enPassant)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("%s: got %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestKnightMoves(t *testing.T) {
	board := emptyBoard()
	knight := NewKnight(White)
	board.board[4][4] = knight

	moves := knight.GetValidMoves(4, 4, board)
	if len(moves) != 8 {
		t.Errorf("Knight moves: got %d, want 8", len(moves))
	}
}

func TestBishopMoves(t *testing.T) {
	board := emptyBoard()
	bishop := NewBishop(White)
	board.board[4][4] = bishop

	moves := bishop.GetValidMoves(4, 4, board)
	if len(moves) != 13 {
		t.Errorf("Bishop moves: got %d, want 13", len(moves))
	}
}

func TestRookMoves(t *testing.T) {
	board := emptyBoard()
	rook := NewRook(White)
	board.board[4][4] = rook

	moves := rook.GetValidMoves(4, 4, board)
	if len(moves) != 14 {
		t.Errorf("Rook moves: got %d, want 14", len(moves))
	}
}

func TestQueenMoves(t *testing.T) {
	board := emptyBoard()
	queen := NewQueen(White)
	board.board[4][4] = queen

	moves := queen.GetValidMoves(4, 4, board)
	if len(moves) != 27 {
		t.Errorf("Queen moves: got %d, want 27", len(moves))
	}
}

func TestKingMoves(t *testing.T) {
	board := emptyBoard()
	king := NewKing(White)
	board.board[4][4] = king

	moves := king.GetValidMoves(4, 4, board)
	if len(moves) != 8 {
		t.Errorf("King moves: got %d, want 8", len(moves))
	}
}
